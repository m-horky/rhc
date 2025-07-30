package main

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/redhatinsights/rhc/pkg/facts"
)

// canonicalFactsAction collects canonical facts about the system and prints them as JSON
func canonicalFactsAction(ctx *cli.Context) error {
	canonicalFacts, err := facts.GetCanonicalFacts()
	if err != nil {
		return fmt.Errorf("unable to collect canonical facts: %v", err)
	}

	data, err := json.MarshalIndent(canonicalFacts, "", "    ")
	if err != nil {
		return fmt.Errorf("unable to marshal canonical facts: %v", err)
	}

	fmt.Println(string(data))
	return nil
}