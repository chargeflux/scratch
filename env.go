package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SpecType describes supported environment types
type SpecType string

var (
	PythonSpec SpecType = "python"
)

func SpecID(t SpecType, name string) string {
	return fmt.Sprintf("%s:%s", t, name)
}

// Spec defines the environment
type Spec struct {
	Name string
	Type SpecType
	Path string
}

// NewSpec creates a new Spec
func NewSpec(name string, t SpecType, wd string) Spec {
	return Spec{name, t, filepath.Join(wd, name)}
}

// LoadSpec loads spec for environment
func LoadSpec(data []byte) (Spec, error) {
	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		return Spec{}, fmt.Errorf("unmarshal spec: %w", err)
	}

	return s, nil
}

// String returns a string represntation of Spec
func (s Spec) String() string {
	return fmt.Sprintf("%s (%s) - %s", s.Name, s.Type, s.Path)
}

// ID returns a unique identifier for the spec
func (s Spec) ID() string {
	return SpecID(s.Type, s.Name)
}

// Exists checks if the environment created by the spec exists
func (s Spec) Exists() bool {
	_, err := os.Stat(s.Path)
	return err == nil
}

// Save saves the spec to storage
func (s Spec) Save(storer Writer) error {
	data, err := json.MarshalIndent(&s, "", " ")
	if err != nil {
		return fmt.Errorf("marshal spec to json: %w", err)
	}
	return storer.Put(s.ID(), data)
}

// Scaffolder applies the spec and builds out the environment
type Scaffolder struct {
	spec Spec
}

// Build creates the environment based on the spec
func (s Scaffolder) Build() error {
	p, err := s.Provisioner(s.spec.Type)
	if err != nil {
		return fmt.Errorf("unknown environment type: %w", err)
	}

	slog.Debug("Checking if provisioner is ready")
	if err := p.Ready(); err != nil {
		return fmt.Errorf("provisioner not ready: %w", err)
	}

	slog.Debug("Checking if output directory already exists")
	if _, err := os.Stat(s.spec.Path); err == nil {
		return fmt.Errorf("environment already exists at location")
	}

	slog.Debug("Ensuring all folders in output path are created")
	if err := os.MkdirAll(s.spec.Path, 0755); err != nil {
		return fmt.Errorf("ensure output directory: %w", err)
	}

	slog.Debug("Provisioning environment")
	if err := p.Provision(s.spec.Path); err != nil {
		return err
	}

	return nil
}

// Provisioner returns the Provisioner associated with the SpecType
func (s Scaffolder) Provisioner(specType SpecType) (Provisioner, error) {
	switch specType {
	case PythonSpec:
		return PythonEnvironment{}, nil
	default:
		return nil, fmt.Errorf("unknown spec type %q", specType)
	}
}

// CommandsExist checks if the command exists in PATH
func CommandsExist(names ...string) error {
	for _, name := range names {
		_, err := exec.LookPath(name)
		if err != nil {
			return fmt.Errorf("command not found: %s", name)
		}
	}
	return nil
}

// RunCommand executes the named program and argument with provided working directory
func RunCommand(wd string, name string, args ...string) error {
	slog.Debug(fmt.Sprintf("Running %q", fmt.Sprintf("%s %s", name, strings.Join(args, " "))))
	initCmd := exec.Command(name, args...)
	initCmd.Dir = wd

	if out, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Provisioner interface for creating a specific type of environment
type Provisioner interface {
	Ready() error
	Provision(dir string) error
}

// PythonEnvironment represents a Python environment to be created
type PythonEnvironment struct{}

// Ready checks if the environment is ready to be created
func (p PythonEnvironment) Ready() error {
	if err := CommandsExist("uv"); err != nil {
		return fmt.Errorf("missing required commands: %w", err)
	}
	return nil
}

// Provision creates the environment at provided directory
func (p PythonEnvironment) Provision(dir string) error {
	if err := EnsureDirectory(dir); err != nil {
		return err
	}

	if err := RunCommand(dir, "uv", "init"); err != nil {
		return fmt.Errorf("init uv: %w", err)
	}

	if err := RunCommand(dir, "uv", "venv"); err != nil {
		return fmt.Errorf("uv venv: %w", err)
	}

	slog.Info("Created environment at " + dir)
	return nil
}

// OpenFolder opens folder with specified program
func OpenFolder(program string, dir string) error {
	if err := RunCommand("", program, dir); err != nil {
		return fmt.Errorf("open folder: %w", err)
	}
	return nil
}
