---
applyTo: '**/*.go'
---

# Go Style & Conventions

Follow these general Go conventions when generating code:

## Packages
- Each Go file must begin with a single `package` statement at the top.
- Do not include multiple `package` statements in the same file.
- Use lowercase, no underscore or camel case for package names.

## Imports
- All `import` blocks must appear immediately after the `package` statement.
- Group standard library imports first, followed by third-party, then internal packages.
- Do not insert `import` statements mid-file.

## Naming
- Use `PascalCase` for exported types, functions, and constants (e.g. `NewDNSQuery`).
- Use `camelCase` for unexported variables and functions.
- Acronyms should be uppercase (e.g. `DNSQuery`, `HTTPServer`).

## Structure
- All type declarations should go near the top of the file.
- Constructors and validation methods should follow the type they belong to.
- Group related functions and types in the same file when logical.

## Comments
- All exported types and functions must have GoDoc-style comments.
- Comments must begin with the name of the item they describe.

## File Layout
- One top-level type per file when possible.
- Use `types.go` only for shared primitive declarations (enums, constants).

## Invariants
- the first line (and component) of a Go file must be the `package` statement.
- `package` may only appear once in a file.
- the second component must be the `import` block.
- `import` may only appear once in a file.
- no code may appear before the `package` statement, or between the `package` and `import` statements.
