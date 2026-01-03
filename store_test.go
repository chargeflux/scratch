package main_test

import (
	"os"
	"testing"

	main "github.com/chargeflux/scratch"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfigDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		dir, err := main.DefaultConfigDir()
		require.NoError(t, err)
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		require.Contains(t, dir, home)
	})

	t.Run("env", func(t *testing.T) {
		tdir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tdir)
		dir, err := main.DefaultConfigDir()
		require.NoError(t, err)
		require.Contains(t, dir, tdir)
	})
}

func TestDefaultDataDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		dir, err := main.DefaultDataDir()
		require.NoError(t, err)
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		require.Contains(t, dir, home)
	})

	t.Run("env", func(t *testing.T) {
		tdir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tdir)
		dir, err := main.DefaultDataDir()
		require.NoError(t, err)
		require.Contains(t, dir, tdir)
	})
}
