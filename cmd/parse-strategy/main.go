package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AinurKadirov/crypto-parser/internal/strategyparser"
)

type strategyYAML struct {
	Strategy struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
		Source      string `yaml:"source,omitempty"`
	} `yaml:"strategy"`
	Parameters map[string]paramYAML `yaml:"parameters"`
}

type paramYAML struct {
	Type    string    `yaml:"type"`
	Default any       `yaml:"default"`
	Range   []float64 `yaml:"range,omitempty"`
	Step    float64   `yaml:"step,omitempty"`
	Options []string  `yaml:"options,omitempty"`
	Label   string    `yaml:"label,omitempty"`
}

func main() {
	in := flag.String("in", "strategies/example.pine", "Path to Pine script text file")
	out := flag.String("out", "configs/strategies/example.generated.yaml", "Path for generated YAML")
	name := flag.String("name", "example_ema", "Strategy name in YAML")
	desc := flag.String("desc", "Generated from Pine strategy inputs", "Strategy description")
	flag.Parse()

	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}

	params, err := strategyparser.ParsePineInputs(string(raw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse inputs: %v\n", err)
		os.Exit(1)
	}

	cfg := strategyYAML{Parameters: make(map[string]paramYAML, len(params))}
	cfg.Strategy.Name = *name
	cfg.Strategy.Version = "1.0"
	cfg.Strategy.Description = *desc
	cfg.Strategy.Source = *in

	for _, p := range params {
		item := paramYAML{
			Type:    normalizeType(p.Type),
			Default: p.Default,
			Options: p.Options,
			Label:   p.Label,
		}
		if p.Min != nil && p.Max != nil {
			item.Range = []float64{*p.Min, *p.Max}
		}
		if p.Step != nil {
			item.Step = *p.Step
		}
		cfg.Parameters[p.Name] = item
	}

	blob, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "yaml marshal: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir output dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, blob, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed %d parameters from %s\n", len(params), *in)
	fmt.Printf("Generated YAML: %s\n", *out)
}

func normalizeType(t string) string {
	switch strings.TrimSpace(strings.ToLower(t)) {
	case "int", "float", "bool", "string":
		return t
	default:
		return "string"
	}
}
