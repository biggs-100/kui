# agent-cli Specification

## Purpose

A minimal command-line entry point that wires the core loop with the OpenAI-compatible provider and the built-in tools, proving the loop end-to-end.

## Requirements

### Requirement: REQ-CLI-1 — Run the Loop

The CLI MUST accept a prompt as its argument, run the agent loop, and print the final answer to standard output.

#### Scenario: Prompt with tool use

- GIVEN the prompt argument "list files in ." and a reachable provider
- WHEN the CLI runs
- THEN the loop completes and the final answer is printed to stdout

#### Scenario: No prompt

- GIVEN no prompt argument
- WHEN the CLI starts
- THEN the CLI prints usage text
- AND exits with a non-zero status

### Requirement: REQ-CLI-2 — Failure Reporting

The CLI MUST exit with a non-zero status and print an actionable message to standard error when provider configuration is invalid or the loop fails.

#### Scenario: Missing API key

- GIVEN OPENAI_API_KEY is unset
- WHEN the CLI starts
- THEN an actionable error naming OPENAI_API_KEY is printed to stderr
- AND the exit status is non-zero

#### Scenario: Successful completion

- GIVEN a prompt argument and a provider returning an answer
- WHEN the CLI runs
- THEN the exit status is zero
