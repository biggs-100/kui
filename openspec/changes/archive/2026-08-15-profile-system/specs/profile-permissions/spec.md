# profile-permissions Specification

## Purpose

Permissions gate which tools an agent may use, expressed as ordered allow/ask/deny rules with wildcards and last-rule-wins semantics. Denied tools are hidden from the provider request payload — the model never sees them.

## Requirements

### Requirement: REQ-PERM-1 — Ruleset Evaluation

A ruleset MUST be an ordered list of rules matching tool names with wildcard support ("*", "read_*"). The LAST matching rule MUST win. An empty ruleset MUST allow all tools, and a tool matching no rule MUST be allowed.

#### Scenario: Last rule wins

- GIVEN rules ["allow read_*", "deny read_secrets"]
- WHEN read_secrets is evaluated
- THEN the result is deny

#### Scenario: Wildcard match

- GIVEN a rule "deny bash_*"
- WHEN bash_run is evaluated
- THEN the result is deny

#### Scenario: Empty ruleset

- GIVEN no rules
- WHEN any tool is evaluated
- THEN the result is allow

#### Scenario: Rule for unregistered tool

- GIVEN a rule naming a tool that is not registered
- WHEN the ruleset is evaluated
- THEN the rule is ignored without error

### Requirement: REQ-PERM-2 — Ask Rules Degrade to Deny

An ask rule MUST NOT trigger an interactive prompt in this change (interactive prompts are out of scope); it MUST evaluate to deny as the safe interim behavior.

#### Scenario: Ask evaluates to deny

- GIVEN a rule "ask write_*"
- WHEN write_file is evaluated
- THEN the result is deny

### Requirement: REQ-PERM-3 — Tool Hiding from Payload

A tool evaluated as deny MUST be excluded from the provider request payload: the outgoing request MUST NOT contain its definition. Tests MUST assert against the request payload, not only against execution blocking.

#### Scenario: Deny removes from payload

- GIVEN a rule "*": "deny" and read_file registered
- WHEN a provider request is built
- THEN the request payload does not include read_file

#### Scenario: Allow keeps in payload

- GIVEN rules ["*": "deny", "allow read_*"]
- WHEN a provider request is built
- THEN the payload includes read_file
- AND excludes write_file and bash

### Requirement: REQ-PERM-4 — Execution Blocking

The tool port MUST reject execution of a denied tool with a typed permission error even if a request arrives, as defense in depth.

#### Scenario: Denied execution rejected

- GIVEN read_secrets denied
- WHEN the loop dispatches it
- THEN a typed permission error is returned
- AND no tool side effect occurs
