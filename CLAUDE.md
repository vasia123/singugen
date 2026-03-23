# Project

SinguGen - self build ai wrapper around claude code. Made to achieve singularity.

- [Roadmap](/docs/ROADMAP.md)

## Stack

- Language: Go

## Principles

- TDD: test first, implement second, refactor third
- Performance over convenience
- No premature abstraction — earn generality through repetition
- Errors are values — handle every error, never ignore
- No panic() in production paths
- Accept interfaces, return structs

## Code Style

- `gofmt` / `goimports` — non-negotiable
- `go vet` + `staticcheck` — zero warnings
- Names: descriptive, no abbreviations except domain-standard
- Comments: only "why", never "what"
- Tests: `_test.go` in the same package for unit, `/tests` for integration

## Architecture Decisions

Document in `/docs/adr/NNNN-title.md` when:
- Choosing between competing approaches
- Adding dependencies

## Quality Bar

Production-grade. Every line, every commit, every decision — as if it ships to thousands of users tomorrow. No prototyping mindset, no "fix later", no shortcuts. Code reviews would pass at a top-tier infra team.

## Agent Expectations

- Read before write. Always.
- Verify assumptions with code, not guesses
- If unsure — ask, don't assume
- Run `go vet ./...` after every edit
- Run `go test ./...` before declaring done
- No stubs. Either complete the implementation or document the gap with a detailed TODO.
- Workarounds require TODO with explanation of the proper fix.
- Use NOTE for non-obvious decisions or important context that isn't actionable.
- Think before generating. Less code that works > more code that might.
- Run `staticcheck ./...` before committing — zero warnings required.

## Pre-commit Hook

Install: `cp scripts/pre-commit .git/hooks/pre-commit`

Runs `go vet`, `staticcheck`, `go test` before each commit.