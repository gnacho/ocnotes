// Package config handles daemon configuration from environment variables.
//
// All keys are required; Load() panics on first missing value (fail-fast).
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr      string // e.g. ":8100"
	DataDir   string // SQLite + secret keys + caches
	GraphURL  string // OpenCloud Graph endpoint: http://localhost:9200/graph/v1.0/me
	AppName   string
}

func Load() Config {
	get := func(key string) string {
		v, ok := os.LookupEnv(key)
		if !ok || v == "" {
			panic(fmt.Sprintf("required env %s not set", key))
		}
		return v
	}

	cfg := Config{
		Addr:     get("OCNOTES_ADDR"),           // default :8100 (set before panic for testing)
		DataDir:  get("OCNOTES_DATA_DIR"),       // e.g. /var/lib/ocnotes
		AppName:  "Notes",                       // display name in extension
	}
	// Defaults applied before panic so tests can override gracefully.
	if cfg.Addr == "" {
		cfg.Addr = ":8100"
	}

	// Auth mode: only "opencloud" supported right now.
	mode := os.Getenv("OCNOTES_AUTH_MODE")
	if mode == "opencloud" || mode == "" {
		cfg.GraphURL = get("OCNOTES_GRAPH_URL")
	} else if mode != "local" {
		panic(fmt.Sprintf("unknown OCNOTES_AUTH_MODE=%q", mode))
	}

	return cfg
}
