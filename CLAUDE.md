# CLAUDE.md

Configuration for Claude Code when working with jp-go-errors package.

## Standards

Use `/ai-common` skill to load development standards and patterns as needed.

## Package Purpose

jp-go-errors provides standardized error handling for Go projects with:

- Sentinel errors for common conditions
- Error constructors with consistent formatting
- HTTP status code integration
- Retry detection for transient failures

## Development Guidelines

This is a **shared package** used across multiple projects. Changes must be:

- Backward compatible
- Well-tested
- Generic (not project-specific)
- Documented in examples
