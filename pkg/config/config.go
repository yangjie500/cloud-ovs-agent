package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	LogLevel string

	LogFile       string
	LogToStdout   bool
	LogMaxSizeMb  int
	LogMaxBackups int
	LogMaxAgeDays int
	LogCompress   bool
}

func LoadAll(dotenvPaths ...string) (Config, error) {
	// Load first .env found (if exists). Safe to call with no args, it tries ".env" by default.
	if len(dotenvPaths) > 0 {
		_ = godotenv.Load(dotenvPaths...)
	} else {
		_ = godotenv.Load()
	}

	return Load()
}

func Load() (Config, error) {
	var cfg Config
	var errs []string

	cfg.LogLevel = getenv("LOG_LEVEL", "info")
	cfg.LogFile = getenv("LOG_FILE", "./logs/app.log")
	cfg.LogToStdout = mustBool("LOG_TO_STDOUT", true, &errs)
	cfg.LogMaxSizeMb = mustInt("LOG_MAX_SIZE_MB", 100, &errs)
	cfg.LogMaxBackups = mustInt("LOG_MAX_BACKUPS", 7, &errs)
	cfg.LogMaxAgeDays = mustInt("LOG_MAX_AGE_DAYS", 14, &errs)
	cfg.LogCompress = mustBool("LOG_COMPRESS", true, &errs)

	if len(errs) > 0 {
		return cfg, errors.New(strings.Join(errs, "; "))
	}

	return cfg, nil

}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustBool(key string, def bool, errs *[]string) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		*errs = append(*errs, key+": invalid bool ("+err.Error()+")")
		return def
	}

	return b
}

func mustInt(key string, def int, errs *[]string) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*errs = append(*errs, key+": invalid int ("+err.Error()+")")
	}
	return n
}

func mustDuration(key string, def time.Duration, errs *[]string) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		*errs = append(*errs, key+": invalid duration ("+err.Error()+")")
		return def
	}
	return d
}
