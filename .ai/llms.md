# jp-go-errors - AI Documentation

Progressive loading map for AI assistants working with jp-go-errors package.

**Entry Point**: This file should be referenced from CLAUDE.md.

## Package Overview

**Purpose**: Standardized error handling for Go projects

**Key Features**:

- Sentinel errors (ErrNotFound, ErrValidation, etc.)
- Error constructors (NewNotFoundError, NewValidationError)
- HTTP status code integration (HTTPError interface)
- Retry detection (IsRetryable, Retryable interface)
- Error wrapping compatibility with standard library

## Always Load

- `.ai/llms.md` (this file)

## Load for Complex Tasks

- `.ai/memory.md` - Design decisions, gotchas, backward compatibility notes
- `.ai/context.md` - Current changes (if exists and is current)

## Common Standards (Portable Patterns)

**See** `.ai/common/common-llms.md` for the complete list of common standards.

Load these common standards when working on this package:

### Core Go Patterns

- `common/standards/go/constructors.md` - New* constructor functions
- `common/standards/go/error-wrapping.md` - Error wrapping with %w
- `common/standards/go/type-organization.md` - Interface and type placement

### Testing

- `common/standards/testing/bdd-testing.md` - Ginkgo/Gomega patterns
- `common/standards/testing/test-categories.md` - Test organization

### Documentation

- `common/standards/documentation/pattern-documentation.md` - Documentation structure
- `common/standards/documentation/code-references.md` - Code examples

## Project Standards (Package-Specific)

This package has minimal package-specific standards since it IS a standard itself.

Any package-specific patterns should go in `.ai/project-standards/`

## Loading Strategy

| Task Type | Load These Standards |
|-----------|---------------------|
| Adding new error type | constructors.md, error-wrapping.md, type-organization.md |
| Writing tests | bdd-testing.md, test-categories.md |
| Documenting errors | pattern-documentation.md, code-references.md |
| Ensuring compatibility | memory.md (for backward compatibility notes) |

## File Organization

```
jp-go-errors/
├── CLAUDE.md                   # Entry point
├── .gitignore                  # Ignores context.md, memory.md, tasks/
└── .ai/
    ├── llms.md                 # This file (loading map)
    ├── README.md               # Documentation about .ai setup
    ├── context.md              # Current work (gitignored)
    ├── memory.md               # Stable knowledge (gitignored)
    ├── tasks/                  # Scratchpad (gitignored)
    ├── project-standards/      # Package-specific (if needed)
    └── common -> ~/code/ai-common  # Symlink to shared standards
```

## Key Principles

1. **Backward Compatibility**: Never break existing error types or behavior
2. **Generic Design**: No project-specific error types in this package
3. **Standard Library Compatible**: Works with errors.Is(), errors.As()
4. **HTTP Integration**: All errors can provide HTTP status codes
5. **Retry-Aware**: Errors indicate if retry makes sense

## Related Documentation

- Common standard: `common/standards/go/jp-go-errors.md` - How to USE this package
- This is the implementation, that is the usage guide
