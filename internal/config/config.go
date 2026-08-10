package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config holds everything one cbt-server instance needs. Every instance is
// one sekolah: its own DB_SCHEMA, its own PORT, its own SEKOLAH_ID. There is
// no multi-tenant switching inside the binary itself — that's handled by
// running one process per sekolah, each with its own .env.
type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSchema   string
	Port       int
	JWTSecret  string
	SekolahID  string
}

// LoadDotEnv reads KEY=VALUE lines from path into the process environment.
// Vars already set (e.g. by systemd's EnvironmentFile) win over the file.
// A missing file is not an error — plain env vars are enough in that case.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

// Load reads Config from the environment. It panics on missing required
// vars on purpose: a cbt-server instance with a bad config should fail to
// start loudly (and get restarted by systemd) rather than run half-blind.
func Load() Config {
	return Config{
		DBHost:     envOr("DB_HOST", "127.0.0.1"),
		DBPort:     envIntOr("DB_PORT", 5432),
		DBUser:     mustGetenv("DB_USER"),
		DBPassword: mustGetenv("DB_PASSWORD"),
		DBName:     mustGetenv("DB_NAME"),
		DBSchema:   mustGetenv("DB_SCHEMA"),
		Port:       envIntOr("PORT", 13000),
		JWTSecret:  mustGetenv("JWT_SECRET"),
		SekolahID:  mustGetenv("SEKOLAH_ID"),
	}
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required env var: " + key)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		panic("invalid int for env var " + key + ": " + v)
	}
	return n
}
