package scaffold

import "strings"

// generateJestConfig produces node/jest.config.js so that Jest uses ts-jest
// to transform TypeScript (.ts) and TSX (.tsx) test files. Without this
// config Jest falls back to Babel which cannot parse TypeScript or JSX.
//
// The default testEnvironment is 'node' for API/supertest tests. Component
// test files (.test.tsx) use a @jest-environment docblock to switch to jsdom.
func generateJestConfig() string {
	var b strings.Builder

	b.WriteString("/** @type {import('ts-jest').JestConfigWithTsJest} */\n")
	b.WriteString("module.exports = {\n")
	b.WriteString("  preset: 'ts-jest',\n")
	b.WriteString("  testEnvironment: 'node',\n")
	b.WriteString("  roots: ['<rootDir>/src'],\n")
	b.WriteString("  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json'],\n")
	b.WriteString("};\n")

	return b.String()
}

// generateReactJestConfig produces react/jest.config.cjs for component tests.
// Uses jsdom environment since component tests render React components.
// Overrides tsconfig settings because the Vite-oriented tsconfig uses
// ESNext/bundler modules which ts-jest cannot process.
func generateReactJestConfig() string {
	var b strings.Builder

	b.WriteString("/** @type {import('ts-jest').JestConfigWithTsJest} */\n")
	b.WriteString("module.exports = {\n")
	b.WriteString("  preset: 'ts-jest',\n")
	b.WriteString("  testEnvironment: 'jsdom',\n")
	b.WriteString("  roots: ['<rootDir>/src'],\n")
	b.WriteString("  setupFiles: ['<rootDir>/jest.setup.cjs'],\n")
	b.WriteString("  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json'],\n")
	b.WriteString("  transform: {\n")
	b.WriteString("    '^.+\\\\.tsx?$': ['ts-jest', {\n")
	b.WriteString("      tsconfig: {\n")
	b.WriteString("        jsx: 'react-jsx',\n")
	b.WriteString("        module: 'commonjs',\n")
	b.WriteString("        moduleResolution: 'node',\n")
	b.WriteString("        esModuleInterop: true,\n")
	b.WriteString("      },\n")
	b.WriteString("    }],\n")
	b.WriteString("  },\n")
	b.WriteString("};\n")

	return b.String()
}

// generateReactJestSetup produces react/jest.setup.cjs that polyfills
// globals missing from jsdom (TextEncoder, TextDecoder) which are needed
// by react-router-dom v7+.
func generateReactJestSetup() string {
	var b strings.Builder

	b.WriteString("const { TextEncoder, TextDecoder } = require('util');\n")
	b.WriteString("global.TextEncoder = TextEncoder;\n")
	b.WriteString("global.TextDecoder = TextDecoder;\n")

	return b.String()
}
