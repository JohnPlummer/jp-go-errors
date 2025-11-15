# CLAUDE.md

Configuration for Claude Code when working with jp-go-errors package.

## Load These First

**CRITICAL:** Always load these files at the start of every session:

- `.ai/llms.md` - Development standards and patterns (progressive loading map)

**Load as needed:**

- `.ai/memory.md` - Stable package knowledge, design decisions, gotchas
- `.ai/context.md` - Current active work, recent changes

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

---

For all development standards, patterns, and workflows, see `.ai/llms.md` and load relevant files on-demand.
