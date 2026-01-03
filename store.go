package main

import (
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cockroachdb/pebble"
)

const (
	AppName = "scratch"
)

// DefaultConfigDir gets configuration directory defined by XDG_CONFIG_HOME
// or defaults to platform equivalent of $HOME/.config/scratch
func DefaultConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName), nil
	}

	// Adapted from os.UserConfigDir()
	switch runtime.GOOS {
	case "windows", "plan9":
		home, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("user config dir: %w", err)
		}
		return filepath.Join(home, AppName), nil
	default:
		// Use .config for Linux and macOS
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("user home dir: %w", err)
		}
		return filepath.Join(home, ".config", AppName), nil
	}
}

// DefaultConfigDir gets data directory defined by XDG_DATA_HOME
// or defaults to platform equivalent of $HOME/.local/share/scratch
func DefaultDataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName), nil
	}

	// Adapted from os.UserCacheDir()
	switch runtime.GOOS {
	case "windows", "plan9":
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("user cache dir: %w", err)
		}
		return filepath.Join(cache, AppName), nil
	default:
		// Use .local/share for Linux and macOS
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
		return filepath.Join(home, ".local", "share", AppName), nil
	}
}

// EnsureDirectory ensures directory exists
func EnsureDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ensure directory exists: %w", err)
	}
	return nil
}

// Reader interface for fetching key-value pairs
type Reader interface {
	Get(key string) ([]byte, error)
	Exists(key string) (bool, error)
}

// Writer interface for adding or deleting key-value pairs
type Writer interface {
	Put(key string, data []byte) error
	Delete(key string) error
}

// Lister interface for listing
type Lister interface {
	List() iter.Seq2[string, error]
	ListFunc(handle func(key string, data []byte) error) error
}

// Storer interface for a storage backend for key-value pairs
type Storer interface {
	Reader
	Writer
	Lister
}

// PebbleStore is a Storer for Pebble DB
type PebbleStore struct {
	db *pebble.DB
}

// pebbleLogger is a logger for Pebble DB
type pebbleLogger struct{}

func (pl pebbleLogger) Infof(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...), slog.String("component", "pebble"))
}

func (pl pebbleLogger) Fatalf(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...), slog.String("component", "pebble"))
}

// NewPebbleStore initializes the database at the default config folder
func NewPebbleStore() (*PebbleStore, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return nil, err
	}
	db, err := pebble.Open(filepath.Join(dir, "data"), &pebble.Options{
		Logger: pebbleLogger{},
	})
	if err != nil {
		return &PebbleStore{}, err
	}
	return &PebbleStore{db}, nil
}

// Exists checks if a key exists
func (p *PebbleStore) Exists(key string) (bool, error) {
	if _, err := p.Get(key); err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("check key exists: %w", err)
	}
	return true, nil
}

// Get fetches data by key
func (p *PebbleStore) Get(key string) ([]byte, error) {
	val, closer, err := p.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("get key %q: %w", key, err)
	}
	defer closer.Close()
	// Value must not be mutated directly
	result := make([]byte, len(val))
	copy(result, val)
	return result, nil
}

// List lists all keys in store
func (p *PebbleStore) List() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		iter, err := p.db.NewIter(nil)
		if err != nil {
			yield("", err)
			return
		}
		defer iter.Close()
		for valid := iter.First(); valid; valid = iter.Next() {
			if !yield(string(iter.Key()), nil) {
				return
			}
		}
	}
}

// ListFunc processes each key-value pair with provided function
func (p *PebbleStore) ListFunc(handle func(key string, data []byte) error) error {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return fmt.Errorf("list keys: %w", err)
	}
	defer iter.Close()
	for valid := iter.First(); valid; valid = iter.Next() {
		// Val must not be mutated directly
		val := make([]byte, len(iter.Value()))
		copy(val, iter.Value())
		if err := handle(string(iter.Key()), val); err != nil {
			return err
		}
	}
	return nil
}

// Put adds or replaces a key with its data
func (p *PebbleStore) Put(key string, data []byte) error {
	if err := p.db.Set([]byte(key), data, pebble.Sync); err != nil {
		return fmt.Errorf("put key %q: %w", key, err)
	}
	return nil
}

// Delete removes key with its data
func (p *PebbleStore) Delete(key string) error {
	if err := p.db.Delete([]byte(key), pebble.Sync); err != nil {
		return fmt.Errorf("delete key %q: %w", key, err)
	}
	return nil
}
