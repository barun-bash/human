# human-plugin-test-plugin

A Human compiler plugin for backend code generation.

## Category

backend

## Quick Start

```bash
# Build the plugin
make build

# Install into Human
make install
# or
human plugin install --binary ./test-plugin

# Verify installation
human plugin list
```

## Protocol

This plugin communicates with the Human compiler via two subcommands:

- `test-plugin meta` — prints plugin metadata as JSON
- `test-plugin generate --ir <path> --output <dir>` — generates code from IR

## Development

```bash
# Build
make build

# Test
make test

# Run meta manually
./test-plugin meta

# Run generate manually
./test-plugin generate --ir path/to/ir.json --output ./out
```
