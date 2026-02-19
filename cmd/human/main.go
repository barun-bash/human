package main

import (
	"fmt"
	"os"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("human %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	case "init":
		fmt.Println("human init: not yet implemented — coming in Phase 6 (CLI)")
	case "build":
		fmt.Println("human build: not yet implemented — coming in Phase 3 (Code Generation)")
	case "run":
		fmt.Println("human run: not yet implemented — coming in Phase 6 (CLI)")
	case "check":
		fmt.Println("human check: not yet implemented — coming in Phase 1 (Parser)")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`Human — English in, production-ready code out.

Usage:
  human <command>

Commands:
  init      Create a new Human project
  build     Compile .human files to target code
  run       Start the development server
  check     Validate .human files

Flags:
  --version, -v   Print the compiler version
  --help, -h      Show this help message

Documentation:
  https://github.com/barun-bash/human
`)
}
