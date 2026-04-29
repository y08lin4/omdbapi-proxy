package main

import (
	"bufio"
	"os"
	"strings"
)

func LoadKeys(filePath string, envValue string) ([]string, error) {
	keys := make([]string, 0)
	seen := make(map[string]struct{})

	appendKey := func(raw string) {
		key := strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
		if key == "" || strings.HasPrefix(key, "#") {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	for _, part := range strings.FieldsFunc(envValue, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	}) {
		appendKey(part)
	}

	if strings.TrimSpace(filePath) == "" {
		return keys, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return keys, nil
		}
		return keys, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		appendKey(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return keys, err
	}

	return keys, nil
}

func MaskKey(key string) string {
	if key == "" {
		return ""
	}
	runes := []rune(key)
	if len(runes) <= 4 {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:2]) + strings.Repeat("*", max(4, len(runes)-4)) + string(runes[len(runes)-2:])
}
