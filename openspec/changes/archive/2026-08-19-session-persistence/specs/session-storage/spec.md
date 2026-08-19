# Session Storage Specification

## Purpose

JSON file persistence for conversation sessions under `.kui/sessions/`. Provides CRUD operations and a metadata index for fast listing.

## Requirements

### Requirement: Session File Format

Each session SHALL be stored as a single JSON file at `.kui/sessions/{id}.json` containing `{id, profile, model, provider, created_at, updated_at, messages[]}`. The `id` SHALL be a human-friendly string (e.g., `coder-2026-08-18-1015`). Messages SHALL use the existing `core.Message` struct with JSON tags for serialization.

#### Scenario: Save new session

- GIVEN a new conversation with messages
- WHEN `Save(session)` is called
- THEN a JSON file is written to `.kui/sessions/{id}.json`
- AND the file contains all messages with correct role, content, and tool call data
- AND `created_at` and `updated_at` are set to the current timestamp

#### Scenario: Update existing session

- GIVEN an existing session file
- WHEN `Save(session)` is called with the same ID
- THEN the file is overwritten with updated messages and `updated_at` timestamp
- AND `created_at` is preserved from the original

### Requirement: Metadata Index

The system SHALL maintain `.kui/sessions/index.json` as a lightweight index of all sessions. The index SHALL contain an array of session metadata entries (id, profile, model, provider, created_at, message_count) without full message content.

#### Scenario: Index updated on save

- GIVEN a session is saved
- WHEN the save completes
- THEN the index file is updated with the session's metadata entry
- AND the entry reflects the current message count and timestamps

#### Scenario: Index rebuilt on drift

- GIVEN the index file is missing or corrupted
- WHEN `List()` is called
- THEN the system SHALL scan `.kui/sessions/*.json` files and rebuild the index
- AND the rebuilt index is persisted

### Requirement: Atomic Writes

Session files SHALL be written atomically: write to a temporary file, then rename to the target path. This prevents corruption from interrupted writes.

#### Scenario: Concurrent write safety

- GIVEN two kui instances saving different sessions simultaneously
- WHEN both write to `.kui/sessions/`
- THEN each session file is written atomically
- AND no session file is corrupted by concurrent access

### Requirement: CRUD Operations

The `SessionStore` port SHALL expose `Save`, `Load`, `List`, and `Delete` methods. `Load` returns the full session with messages. `List` returns metadata from the index. `Delete` removes both the session file and index entry.

#### Scenario: Load session by ID

- GIVEN a saved session with ID `coder-2026-08-18-1015`
- WHEN `Load("coder-2026-08-18-1015")` is called
- THEN the full session with all messages is returned

#### Scenario: Delete session

- GIVEN a saved session
- WHEN `Delete(id)` is called
- THEN the session file is removed
- AND the index entry is removed
