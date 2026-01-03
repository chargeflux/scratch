package main_test

import (
	"errors"
	"log/slog"
	"os"
	"path"
	"testing"

	main "github.com/chargeflux/scratch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MemoryStore struct {
	Data map[string][]byte
}

func NewMemoryStore() MemoryStore {
	return MemoryStore{map[string][]byte{}}
}

func (m MemoryStore) Get(key string) ([]byte, error) {
	if v, ok := m.Data[key]; !ok {
		return nil, errors.New("key does not exist")
	} else {
		return v, nil
	}
}

func (m MemoryStore) Exists(key string) (bool, error) {
	_, ok := m.Data[key]
	return ok, nil
}

func (m MemoryStore) Put(key string, data []byte) error {
	m.Data[key] = data
	return nil
}

func (m MemoryStore) Delete(key string) error {
	if _, ok := m.Data[key]; !ok {
		return errors.New("key does not exist")
	}
	delete(m.Data, key)
	return nil
}

func TestNewSpec(t *testing.T) {
	tdir := t.TempDir()
	name := "test"
	got := main.NewSpec(name, main.PythonSpec, tdir)
	expected := main.Spec{
		Name: name,
		Type: main.PythonSpec,
		Path: path.Join(tdir, name),
	}
	assert.Equal(t, expected, got)
}

func TestSpec_ID(t *testing.T) {
	tdir := t.TempDir()
	t.Run("regular", func(t *testing.T) {
		spec := main.NewSpec("test", main.PythonSpec, tdir)

		assert.Equal(t, "python:test", spec.ID())
	})

	t.Run("colon", func(t *testing.T) {
		spec := main.NewSpec(":test-bar", main.PythonSpec, tdir)

		assert.Equal(t, "python::test-bar", spec.ID())
	})

	t.Run("space", func(t *testing.T) {
		spec := main.NewSpec("test bar", main.PythonSpec, tdir)

		assert.Equal(t, "python:test bar", spec.ID())
	})
}

func TestSpec_Exists(t *testing.T) {
	tdir := t.TempDir()
	name := "test"
	spec := main.NewSpec(name, main.PythonSpec, tdir)

	assert.False(t, spec.Exists())

	err := os.Mkdir(path.Join(tdir, name), 0644)
	assert.NoError(t, err)

	assert.True(t, spec.Exists())
}

func TestSpec_SaveLoad(t *testing.T) {
	tdir := t.TempDir()
	name := "test"
	spec := main.NewSpec(name, main.PythonSpec, tdir)
	mw := NewMemoryStore()
	err := spec.Save(mw)
	require.NoError(t, err)

	require.Contains(t, mw.Data, spec.ID())

	lspec, err := main.LoadSpec(mw.Data[spec.ID()])

	require.NoError(t, err)
	require.Equal(t, spec, lspec)
}

func TestCommandsExist(t *testing.T) {
	require.NoError(t, main.CommandsExist("go"))

	require.Error(t, main.CommandsExist("foo"))
}

func TestRunCommand(t *testing.T) {
	require.NoError(t, main.RunCommand("", "echo", "Hello"))
	require.Error(t, main.RunCommand("", "foo"))
}

func TestPythonEnvironment_Ready(t *testing.T) {
	require.NoError(t, main.PythonEnvironment{}.Ready())
}

func TestPythonEnvironment_Provision(t *testing.T) {
	tdir := t.TempDir()
	slog.SetDefault(slog.New(slog.DiscardHandler))
	p := main.PythonEnvironment{}
	require.NoError(t, p.Provision(tdir))

	entries, err := os.ReadDir(tdir)
	require.NoError(t, err)
	items := []string{}
	for _, entry := range entries {
		items = append(items, entry.Name())
	}
	require.Contains(t, items, ".venv")
	require.Contains(t, items, "pyproject.toml")
}
