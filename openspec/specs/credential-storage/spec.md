# credential-storage Specification

## Purpose

Credential file management for provider API keys. Read, write, and validate `.kui/credentials.json` with layered resolution integrated with environment variables.

## Requirements

### Requirement: REQ-CRED-1 — Credential File Location

The system MUST store provider credentials in `.kui/credentials.json` relative to the project root.

#### Scenario: File path resolved from project root

- GIVEN project root `/home/user/myproject`
- WHEN credential store loads
- THEN file path is `/home/user/myproject/.kui/credentials.json`

#### Scenario: File does not exist

- GIVEN no `.kui/credentials.json` on disk
- WHEN credential store loads
- THEN an empty credential set is returned (no error)

### Requirement: REQ-CRED-2 — Credential File Format

The credentials file MUST be JSON with a `"providers"` map keyed by provider name, each containing an `"api_key"` field.

#### Scenario: Valid file read

- GIVEN `.kui/credentials.json` contains `{"providers":{"openai":{"api_key":"sk-123"}}}`
- WHEN credentials are loaded
- THEN the openai provider key is `sk-123`

#### Scenario: Malformed JSON

- GIVEN `.kui/credentials.json` contains `{invalid`
- WHEN credentials are loaded
- THEN a descriptive parse error is returned

### Requirement: REQ-CRED-3 — File Permissions

The system MUST set `.kui/credentials.json` permissions to `0600` (owner read/write only) on Unix systems.

#### Scenario: File created with restrictive permissions

- GIVEN credentials are saved
- WHEN the file is written
- THEN file permissions are `0600`

#### Scenario: Directory created if missing

- GIVEN `.kui/` directory does not exist
- WHEN credentials are saved
- THEN `.kui/` is created before writing

### Requirement: REQ-CRED-4 — GetAPIKey

The credential store MUST expose a `GetAPIKey(provider) (string, error)` method returning the stored key or an error when not found.

#### Scenario: Key found

- GIVEN openai key is stored as `sk-123`
- WHEN `GetAPIKey("openai")` is called
- THEN result is `"sk-123"`

#### Scenario: Key not found

- GIVEN no anthropic key is stored
- WHEN `GetAPIKey("anthropic")` is called
- THEN error is returned indicating no key for that provider

### Requirement: REQ-CRED-5 — SetAPIKey

The credential store MUST expose a `SetAPIKey(provider, key) error` method that persists the key and writes the file to disk.

#### Scenario: New key saved

- GIVEN credential store is empty
- WHEN `SetAPIKey("openai", "sk-123")` is called
- THEN `.kui/credentials.json` contains the openai key
- AND file permissions are `0600`

#### Scenario: Existing key overwritten

- GIVEN openai key is `sk-old`
- WHEN `SetAPIKey("openai", "sk-new")` is called
- THEN `.kui/credentials.json` contains `sk-new`

### Requirement: REQ-CRED-6 — Setup Wizard

`kui setup` MUST launch an interactive wizard that lists providers, accepts API key input with masked display, validates the key, and saves it.

#### Scenario: Interactive full setup

- GIVEN user runs `kui setup`
- WHEN wizard starts
- THEN available providers are listed (e.g. `openai`, `opencode`)
- AND user is prompted to select a provider
- AND API key input is masked (not echoed)

#### Scenario: Non-interactive single-provider setup

- GIVEN user runs `kui setup --provider openai`
- WHEN wizard starts
- THEN only openai is configured (no provider selection prompt)
- AND API key input is still prompted

#### Scenario: Empty input rejected

- GIVEN user provides empty API key
- WHEN wizard validates
- THEN error message is displayed
- AND wizard re-prompts for input

### Requirement: REQ-CRED-7 — API Key Validation

The system MUST validate the API key format before saving: non-empty, trimmed of whitespace, and basic prefix check per provider (e.g. `sk-` for openai).

#### Scenario: Valid key accepted

- GIVEN user enters `sk-abc123` for openai
- WHEN validation runs
- THEN key is accepted and saved

#### Scenario: Invalid format rejected

- GIVEN user enters `short` for openai (no `sk-` prefix)
- WHEN validation runs
- THEN error is returned suggesting the expected format

### Requirement: REQ-CRED-8 — Setup Success Output

After successful save, `kui setup` MUST print a success message with next steps.

#### Scenario: Successful setup

- GIVEN key is validated and saved
- WHEN setup completes
- THEN stdout shows "Credentials saved for {provider}."
- AND stdout shows "Next step: run `kui tui` to start chatting."
