package crypto

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

func EnsureSecretKey(dataDir, filename string) ([]byte, error) {
	path := filepath.Join(dataDir, filename)

	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return nil, fmt.Errorf("rand: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, seed, 0600); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return nil, fmt.Errorf("rename: %w", err)
	}

	return seed, nil
}
