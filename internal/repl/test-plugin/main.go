package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: test-plugin <meta|generate> [flags]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "meta":
		meta := map[string]string{
			"name":        "test-plugin",
			"version":     "0.1.0",
			"description": "Test-Plugin code generator plugin",
			"category":    "backend",
		}
		json.NewEncoder(os.Stdout).Encode(meta)
	case "generate":
		if err := runGenerate(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
