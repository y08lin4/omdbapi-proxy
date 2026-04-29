package main

import (
	"sync"
	"time"
)

type ManagedKey struct {
	Index         int
	Value         string
	Masked        string
	InFlight      int
	Total         int64
	Successes     int64
	Failures      int64
	DisabledUntil time.Time
	LastError     string
	LastUsedAt    time.Time
}

type KeySnapshot struct {
	Index         int    `json:"index"`
	Key           string `json:"key"`
	InFlight      int    `json:"inFlight"`
	Total         int64  `json:"total"`
	Successes     int64  `json:"successes"`
	Failures      int64  `json:"failures"`
	Available     bool   `json:"available"`
	DisabledForMS int64  `json:"disabledForMs"`
	LastError     string `json:"lastError,omitempty"`
	LastUsedAt    string `json:"lastUsedAt,omitempty"`
}

type KeyPoolStats struct {
	TotalKeys     int           `json:"totalKeys"`
	AvailableKeys int           `json:"availableKeys"`
	CooldownMS    int64         `json:"cooldownMs"`
	Keys          []KeySnapshot `json:"keys,omitempty"`
}

type KeyPool struct {
	mu       sync.Mutex
	keys     []*ManagedKey
	cursor   int
	cooldown time.Duration
}

func NewKeyPool(keys []string, cooldown time.Duration) *KeyPool {
	p := &KeyPool{cooldown: cooldown}
	p.Reload(keys)
	return p
}

func (p *KeyPool) Reload(keys []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	oldByValue := make(map[string]*ManagedKey, len(p.keys))
	for _, key := range p.keys {
		oldByValue[key.Value] = key
	}

	newKeys := make([]*ManagedKey, 0, len(keys))
	for idx, value := range keys {
		if old, ok := oldByValue[value]; ok {
			copyKey := *old
			copyKey.Index = idx
			copyKey.Masked = MaskKey(value)
			newKeys = append(newKeys, &copyKey)
			continue
		}
		newKeys = append(newKeys, &ManagedKey{
			Index:  idx,
			Value:  value,
			Masked: MaskKey(value),
		})
	}

	p.keys = newKeys
	if len(p.keys) == 0 {
		p.cursor = 0
	} else {
		p.cursor %= len(p.keys)
	}
}

func (p *KeyPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.keys)
}

func (p *KeyPool) Acquire(tried map[int]bool) *ManagedKey {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.keys) == 0 {
		return nil
	}
	now := time.Now()
	for step := 0; step < len(p.keys); step++ {
		idx := (p.cursor + step) % len(p.keys)
		if tried != nil && tried[idx] {
			continue
		}
		key := p.keys[idx]
		if key.DisabledUntil.After(now) {
			continue
		}
		p.cursor = (idx + 1) % len(p.keys)
		key.InFlight++
		key.Total++
		key.LastUsedAt = now
		return key
	}
	return nil
}

func (p *KeyPool) Release(key *ManagedKey) {
	if key == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if key.InFlight > 0 {
		key.InFlight--
	}
}

func (p *KeyPool) ReportSuccess(key *ManagedKey) {
	if key == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	key.Successes++
	key.LastError = ""
}

func (p *KeyPool) ReportFailure(key *ManagedKey, reason string) {
	if key == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	key.Failures++
	key.LastError = reason
	key.DisabledUntil = time.Now().Add(p.cooldown)
}

func (p *KeyPool) Stats(includeKeys bool) KeyPoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	stats := KeyPoolStats{
		TotalKeys:  len(p.keys),
		CooldownMS: p.cooldown.Milliseconds(),
	}
	if includeKeys {
		stats.Keys = make([]KeySnapshot, 0, len(p.keys))
	}
	for _, key := range p.keys {
		available := !key.DisabledUntil.After(now)
		if available {
			stats.AvailableKeys++
		}
		if includeKeys {
			lastUsed := ""
			if !key.LastUsedAt.IsZero() {
				lastUsed = key.LastUsedAt.Format(time.RFC3339)
			}
			stats.Keys = append(stats.Keys, KeySnapshot{
				Index:         key.Index,
				Key:           key.Masked,
				InFlight:      key.InFlight,
				Total:         key.Total,
				Successes:     key.Successes,
				Failures:      key.Failures,
				Available:     available,
				DisabledForMS: max(0, key.DisabledUntil.Sub(now).Milliseconds()),
				LastError:     key.LastError,
				LastUsedAt:    lastUsed,
			})
		}
	}
	return stats
}
