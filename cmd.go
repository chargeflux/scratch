package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func askForConfirmation(prompt string) (bool, error) {
	for {
		fmt.Printf("%s [y/n]: ", prompt)
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			return false, err
		}

		if input == "y" || input == "yes" {
			return true, nil
		}
		if input == "n" || input == "no" {
			return false, nil
		}
	}
}

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

// DeleteCmd represents the command to delete an environment or environments
type DeleteCmd struct {
	ID    string `help:"The ID of environment"`
	Name  string `short:"n" help:"The name of environment"`
	Type  string `short:"t" help:"The type of environment" default:"python"`
	Force bool   `short:"f" help:"Delete without confirmation"`
	All   bool   `help:"Delete all environments"`
}

// Validate checks the combination of flags
func (d DeleteCmd) Validate() error {
	if d.All {
		if d.ID != "" || d.Name != "" {
			return fmt.Errorf("--all cannot be used with a specific ID or Name")
		}
		return nil
	}

	if d.ID != "" {
		if d.Name != "" {
			return fmt.Errorf("specify either --id OR --name, not both")
		}
		return nil
	}

	if d.Name != "" {
		return nil
	}

	return fmt.Errorf("must specify --id, --name, or --all")
}

// deleteKeyEnv deletes key and environment if it exists
func (d DeleteCmd) deleteKeyEnv(store Storer, key string, force bool) error {
	l := slog.With(slog.String("id", key))
	l.Debug("Get environment data")
	data, err := store.Get(key)
	if err != nil {
		return fmt.Errorf("get environment %q data: %w", key, err)
	}

	spec, err := LoadSpec(data)
	if err != nil {
		return err
	}

	if !force {
		ok, err := askForConfirmation(fmt.Sprintf("Delete %s?", key))
		if err != nil {
			return err
		}
		if !ok {
			l.Info("Not deleting environment")
			return nil
		}
	}

	if spec.Exists() {
		l.Info("Removing environment directory")
		if err := os.RemoveAll(spec.Path); err != nil {
			return fmt.Errorf("remove environment %q: %w", key, err)
		}
	}

	l.Debug("Deleting environment key")
	if err := store.Delete(key); err != nil {
		return err
	}

	slog.Info("Deleted environment", slog.String("id", key))
	return nil
}

// Run deletes environment by key, name and type or all enviroments
func (d DeleteCmd) Run(ctx *CLIContext) error {
	store, err := ctx.Store()
	if err != nil {
		return err
	}
	if !d.All {
		key := d.ID
		if key == "" {
			key = Spec{Name: d.Name, Type: SpecType(d.Type)}.ID()
		}

		if err := d.deleteKeyEnv(store, key, d.Force); err != nil {
			return err
		}
		return nil
	}

	slog.Info("Deleting all environments")
	keys := []string{}
	for key, err := range store.List() {
		if err != nil {
			return fmt.Errorf("get all keys: %w", err)
		}
		keys = append(keys, key)
	}

	for _, key := range keys {
		if err := d.deleteKeyEnv(store, key, d.Force); err != nil {
			return err
		}
	}
	return nil
}

// CLI describes available commands and flags
var CLI struct {
	Verbose bool      `short:"v" help:"Enable verbose logging"`
	New     NewCmd    `cmd:"" help:"Create a new environment"`
	List    ListCmd   `cmd:"" help:"List environments"`
	Delete  DeleteCmd `cmd:"" help:"Delete environments"`
}
