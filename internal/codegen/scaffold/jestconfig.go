package scaffold

import "strings"

// generateJestConfig produces node/jest.config.js so that Jest uses ts-jest
// to transform TypeScript (.ts) and TSX (.tsx) test files. Without this
// config Jest falls back to Babel which cannot parse TypeScript or JSX.
func generateJestConfig() string {
	var b strings.Builder

	b.WriteString("/** @type {import('ts-jest').JestConfigWithTsJest} */\n")
	b.WriteString("module.exports = {\n")
	b.WriteString("  preset: 'ts-jest',\n")
	b.WriteString("  testEnvironment: 'jsdom',\n")
	b.WriteString("  roots: ['<rootDir>/src'],\n")
	b.WriteString("  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json'],\n")
	b.WriteString("};\n")

	return b.String()
}
