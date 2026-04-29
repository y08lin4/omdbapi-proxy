package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr            string
	OMDBKeysFile          string
	ClientKeysFile        string
	AdminKey              string
	OMDBAPIURL            string
	OMDBPosterURL         string
	HTTPTimeout           time.Duration
	KeyCooldown           time.Duration
	MaxAttemptsPerRequest int
	CORSOrigin            string
}

func LoadConfig() Config {
	loadDotEnv(".env")

	return Config{
		ListenAddr:            envString("LISTEN_ADDR", ":8080"),
		OMDBKeysFile:          envString("OMDB_KEYS_FILE", "omdb_keys.txt"),
		ClientKeysFile:        envString("CLIENT_KEYS_FILE", "client_keys.txt"),
		AdminKey:              os.Getenv("ADMIN_KEY"),
		OMDBAPIURL:            envString("OMDB_API_URL", "https://www.omdbapi.com/"),
		OMDBPosterURL:         envString("OMDB_POSTER_URL", "https://img.omdbapi.com/"),
		HTTPTimeout:           envDuration("HTTP_TIMEOUT", 10*time.Second),
		KeyCooldown:           envDuration("KEY_COOLDOWN", 5*time.Minute),
		MaxAttemptsPerRequest: envInt("MAX_ATTEMPTS_PER_REQUEST", 0),
		CORSOrigin:            os.Getenv("CORS_ORIGIN"),
	}
}

func envString(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid %s=%q, use %s", name, value, fallback)
		return fallback
	}
	return d
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		log.Printf("invalid %s=%q, use %d", name, value, fallback)
		return fallback
	}
	return n
}

func loadDotEnv(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		if len(value) >= 2 {
			first, last := value[0], value[len(value)-1]
			if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
				value = value[1 : len(value)-1]
			}
		}
		os.Setenv(key, value)
	}
}
