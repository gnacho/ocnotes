package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr     string
	DataDir  string
	GraphURL string
	AppName  string
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
		Addr:    get("OCNOTES_ADDR"),
		DataDir: get("OCNOTES_DATA_DIR"),
		AppName: "Notes",
	}

	if cfg.Addr == "" {
		cfg.Addr = ":8100"
	}

	mode := os.Getenv("OCNOTES_AUTH_MODE")
	if mode == "opencloud" || mode == "" {
		cfg.GraphURL = get("OCNOTES_GRAPH_URL")
	} else if mode != "local" {
		panic(fmt.Sprintf("unknown OCNOTES_AUTH_MODE=%q", mode))
	}

	return cfg
}
