package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Reader struct {
	cache map[string]string
}

func NewReader() (*Reader, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	return &Reader{map[string]string{}}, nil
}

func (r Reader) get(key string) string {
	if value, ok := r.cache[key]; ok {
		return value
	}

	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Errorf("Env var not found: %s", key))
	}

	r.cache[key] = value
	return value
}

func (r Reader) DatabaseURL() string {
	return r.get("DATABASE_URL")
}
