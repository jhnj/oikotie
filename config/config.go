package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/joho/godotenv"
)

type SearchConfig struct {
	Areas []string `json:"areas"`
	Price *struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"price"`
	Size *struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"size"`
}

type Reader struct {
	cache        map[string]string
	searchConfig *SearchConfig
}

func NewReader() (*Reader, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	return &Reader{map[string]string{}, nil}, nil
}

func (r *Reader) get(key string) string {
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

func (r *Reader) DatabaseURL() string {
	return r.get("DATABASE_URL")
}

func (r *Reader) SearchConfig() *SearchConfig {
	if r.searchConfig == nil {
		raw, err := ioutil.ReadFile(r.get("SEARCH_CONFIG_PATH"))
		if err != nil {
			panic(fmt.Errorf("Failed to read config, %w", err))
		}

		err = json.Unmarshal(raw, &r.searchConfig)
		if err != nil {
			panic(fmt.Errorf("Failed to parse config, %w", err))
		}
	}

	return r.searchConfig
}
