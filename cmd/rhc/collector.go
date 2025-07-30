package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/redhatinsights/rhc/pkg/config"
	"github.com/redhatinsights/rhc/pkg/facts"
)

// collectorAction collects canonical facts and writes them to a file
func collectorAction(ctx *cli.Context) error {
	canonicalFacts, err := facts.GetCanonicalFacts()
	if err != nil {
		return fmt.Errorf("unable to collect canonical facts: %v", err)
	}

	data, err := json.MarshalIndent(canonicalFacts, "", "    ")
	if err != nil {
		return fmt.Errorf("unable to marshal canonical facts: %v", err)
	}

	// Write facts to standard location
	factsDir := filepath.Join(config.LocalstateDir, "lib", "rhc")
	err = os.MkdirAll(factsDir, 0755)
	if err != nil {
		return fmt.Errorf("unable to create facts directory: %v", err)
	}

	factsFile := filepath.Join(factsDir, "canonical-facts.json")
	err = os.WriteFile(factsFile, data, 0644)
	if err != nil {
		return fmt.Errorf("unable to write facts file: %v", err)
	}

	fmt.Printf("Canonical facts written to %s\n", factsFile)
	return nil
}