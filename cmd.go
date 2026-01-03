package main

import (
	"fmt"
	"log/slog"
	"path/filepath"
)

// CLIContext has common structs for commands
type CLIContext struct {
	store Storer
}

// Store lazily retrieves Storer
func (c CLIContext) Store() (Storer, error) {
	if c.store != nil {
		return c.store, nil
	}

	dir, err := DefaultConfigDir()
	if err != nil {
		return nil, err
	}
	EnsureDirectory(dir)

	db, err := NewPebbleStore()
	if err != nil {
		return nil, fmt.Errorf("get db: %w", err)
	}

	c.store = db

	return db, nil
}

// NewCmd represents the command to create a new environment
type NewCmd struct {
	Name      string   `arg:"" help:"The name of environment" required:""`
	Type      SpecType `short:"t" help:"The type of environment" default:"python"`
	Directory string   `short:"d" help:"The parent output directory"`
	Open      string   `help:"Open folder in program" default:"code"`
	NoOpen    bool     `help:"Don't open folder"`
}

// resolveOutputDir resolves the absolute path to which the new environment is created in
func (c NewCmd) resolveOutputDir() (string, error) {
	var dir = c.Directory
	if dir == "" {
		ddir, err := DefaultDataDir()
		if err != nil {
			return "", err
		}
		dir = ddir
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// Spec creates a Spec based on user input
func (c NewCmd) spec() (Spec, error) {
	outputDir, err := c.resolveOutputDir()
	if err != nil {
		return Spec{}, err
	}
	return NewSpec(c.Name, c.Type, outputDir), nil
}

// Run provisions the new environment and saves the spec
func (c NewCmd) Run(ctx *CLIContext) error {
	spec, err := c.spec()
	if err != nil {
		return err
	}

	store, err := ctx.Store()
	if err != nil {
		return err
	}

	exists, err := store.Exists(spec.ID())
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("environment %q already exists elsewhere", spec.ID())
	}

	s := Scaffolder{spec}
	if err := s.Build(); err != nil {
		return err
	}

	if err := spec.Save(store); err != nil {
		return err
	}

	if !c.NoOpen {
		if err := OpenFolder(c.Open, spec.Path); err != nil {
			return err
		}
	}

	return nil
}

// ListCmd represents the command to list all available environments
type ListCmd struct {
	DirectoryOnly bool `short:"d" name:"directories" help:"List directories only"`
}

// Run retrieves all available environments and prints them out
func (l ListCmd) Run(ctx *CLIContext) error {
	listFunc := func(key string, data []byte) error {
		spec, err := LoadSpec(data)
		if err != nil {
			return err
		}

		if !spec.Exists() {
			slog.Error("Environment does not exist",
				slog.String("name", spec.Name),
				slog.String("path", spec.Path),
				slog.String("id", spec.ID()),
			)
			return nil
		}

		if l.DirectoryOnly {
			fmt.Println(spec.Path)
		} else {
			fmt.Println(spec)
		}

		return nil
	}

	store, err := ctx.Store()
	if err != nil {
		return err
	}

	return store.ListFunc(listFunc)
}

// CLI describes available commands and flags
var CLI struct {
	Verbose bool    `short:"v" help:"Enable verbose logging"`
	New     NewCmd  `cmd:"" help:"Create a new environment"`
	List    ListCmd `cmd:"" help:"List environments"`
}
