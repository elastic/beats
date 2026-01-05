---
applyTo: '**'
---

# Context
- Senior Software Engineer with 15+ years experience (Go expert, 10+ years)
- Assume expert-level knowledge of languages, frameworks, tools, best practices
- Provide detailed technical responses without oversimplifying
- Show architectural/design considerations and security implications
- When code is requested, provide code only unless explanations are explicitly asked

# Project Structure
- OSS Beats (root level): `auditbeat/`, `filebeat/`, `heartbeat/`, `metricbeat/`, `packetbeat/`, `winlogbeat/`
- Elastic-licensed Beats: `x-pack/` folder (e.g., `x-pack/osquerybeat/`)
- Shared framework: `libbeat/` - common interfaces, processors, outputs, pipeline, publisher. Follows same licensing scheme as individual Beats
- NEVER import Elastic-licensed code (`x-pack/`) into OSS code
- Build/dev tools: `dev-tools/` - mage build system, packaging templates, testing utilities
- Documentation: `docs/` - markdown documentation
- Testing infrastructure: `testing/` - test utilities, environments, terraform configs
- Integration tests: `tests/` folder in each Beat. Some Beats also have `testing/` folder - these map to different testing frameworks in `libbeat`
- When modifying a Beat: check both OSS version (root) and x-pack version (if it exists in `x-pack/`)
- Config files: Beat root and module directories contain `.yml`, `.yaml` config files
- Current Beats version is in `libbeat/version/version.go:20`
- Documentation for Beats version < 9.0.0 is in AsciiDoc format

## Automation
- [Mage](https://magefile.org/) is used for project automation
- Common commands (run in Beat folder): `mage update` (regenerate files), `mage build` (compile), `mage check` (format, update, validate), `mage unitTest` (run tests), `mage clean` (remove build artifacts)
- After code changes: run `mage update` first if modifying fields/configs, then `mage check` for validation
- Root-level commands: `mage fmt` (format all), `mage unitTest` (test all Beats), `mage checkLicenseHeaders` (validate headers)
- Legacy Makefile targets exist for compatibility but Mage is preferred
- Never run mage targets like `test`, `unitTest`, `goIntegTest`, etc. without explicitly stating intent and asking for confirmation. These commands run the entire test suite, which can take a very long time.

# Communication
- Direct, honest, no sugar-coating
- Correct mistakes immediately, challenge incorrect assumptions
- Precise language, eliminate unnecessary words
- Be concise - avoid fluff, sacrifice grammar for conciseness
- Avoid speculation - state when unsure, suggest research methods
- Ask clarifying questions when requirements are vague

# Code References & Precision
- Format: `filepath:line_number` or `filepath:start-end` (ALWAYS exact locations)
- Never: "around line X", "similar to", vague references
- Always: specific element names (functions, classes, methods, variables)
- Quote exact text from code/logs - zero paraphrasing
- Read entire files, check related files (imports, dependencies, config)
- Trace execution flow across multiple files
- Identify ALL instances of patterns, provide multiple examples
- Reference exact variable names, function calls, error codes
- Never use approximations ("around", "similar to")

# Problem-Solving Workflow
1. Root cause analysis: reference exact files:line, provide evidence chain (symptoms → causes), quantify frequency/severity
2. Present 2-3 solution options with pros/cons, highlight recommended option
3. Wait for confirmation before implementing
4. Implementation: complete function/method modifications (never fragments), all imports/dependencies, exact config changes with parameter names/values, before/after comparisons, exact line numbers
5. Verification: provide exact commands and expected outputs, include tests with exact test names, use reverse logic to confirm resolution

# Code Changes
- Check workspace folders before suggesting changes, verify file existence/structure
- State which files/folders changes apply to, provide full file paths
- If file doesn't exist, clearly state it must be created and where
- When user claims fixes implemented: verify changes align with discussion, point out discrepancies with exact locations
- Maintain consistency with existing codebase patterns
- Always ensure the application complies, both versions, the OSS and the one in the `x-pack` folder.

# Log Analysis
Four-pass method: Overview → Error Focus → Context Analysis → Cross-Reference
- Search: exact error messages, variable/function names, state transitions (start/stop/fail/success)
- Patterns: correlation IDs, request IDs, session IDs, timestamps, user IDs
- Levels: ERROR, WARN, INFO, DEBUG
- Parse structured logs (JSON) for complete context
- Track event sequences with timestamp ranges/frequency, find anomalies, track state changes

# Response Standards
- Always provide solutions when discussing errors - never just diagnose
- Include specific files and line numbers for fixes
- Don't wait to be asked for implementation details
- If answer unknown, state clearly and suggest research methods
- Exact matches, precise quotes, specific counts, measurements

# Output
- Save intermediate markdown to `.tmp-ai-io` folder (exclude from git)
- Optimize .md files for token efficiency - concise, minimal formatting, no emojis
- At end of each plan, list unresolved questions if any
- Take time to think through problems - thorough, precise answers
- Solve step-by-step

# Go Code Standards
## General
- Follow Go best practices and idiomatic style
- Maintain consistency with existing codebase patterns
- Write clear, self-documenting code with meaningful variable names
- Add comments for complex logic or non-obvious behavior
- Prefer explicit error handling over silent failures
- Use `any` instead of `interface{}`
- If editing any .go file, add `// This file was contributed to by generative AI` below license header

## Style
- Use `gofmt` formatting standards
- Follow project's existing naming conventions
- Keep functions focused and single-purpose
- Prefer composition over inheritance
- Use interfaces for abstraction

## Error Handling
- Always handle errors explicitly; never ignore them
- Return errors from functions when operations can fail
- Use descriptive error messages with context
- Wrap errors with additional context when propagating

## Testing
- Write unit tests for new functionality
- Use table-driven tests when appropriate
- Follow existing test patterns in codebase
- Ensure tests are deterministic, don't rely on external state
- Only use `t.Helper()` on functions whose goal is to make assertions
- Use `t.Context()` instead of `context.Background()`
- When using testify assertions (assert.*, require.*), always include context messages as last argument

## Performance
- Consider performance implications for hot paths
- Use appropriate data structures for use case
- Avoid premature optimization
- Profile before optimizing

## Documentation
- Document exported functions and types
- Keep comments concise, focused on "why" not "what"
- Update documentation when changing behavior

## Project-Specific
- OSS beats are in the root folder, the ones in `x-pack` are Elastic licensed.
- Follow Elastic Beats module structure conventions
- Maintain compatibility with existing Beat interfaces
- Consider cross-platform compatibility (Linux, Windows, macOS)
- Follow project's module and package organization patterns
