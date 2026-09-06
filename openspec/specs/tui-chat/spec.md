# tui-chat Specification

## Purpose

The chat view shows the conversation: user prompts, streaming assistant answers, and explicit `{profile, model}` context on every prompt.

## Requirements

### Requirement: REQ-TUI-CHAT-1 — Prompt Submission

The user MUST submit with Enter carrying session `{profile, model}`; empty input MUST be ignored. Submitted user messages MUST render as a `Box` with bg `#343541`, full-width, with padding — NO `┃` SplitBorder, NO `you:` label.
(Previously: left-border SplitBorder with agent color)

#### Scenario: User box renders

- GIVEN prompt "hello" under profile coder
- WHEN chat dumps
- THEN message appears in `#343541` Box without `┃` or `you:`

#### Scenario: Empty ignored

- GIVEN whitespace-only input
- WHEN Enter pressed
- THEN nothing submits and view is unchanged

### Requirement: REQ-TUI-CHAT-2 — Streaming Answer Rendering

Assistant messages MUST render as plain markdown with NO background and NO border, followed by `Spacer(1)`. Streaming chunks MUST append incrementally to the same block. `┃`/`╹` end-caps, hover background, `QUEUED` badge and stickyScroll acceleration are REMOVED.
(Previously: per-part SplitBorder `┃` + `╹` terminator, hover backgroundElement, QUEUED badge)

#### Scenario: Plain assistant + spacer

- GIVEN assistant answer with two parts
- WHEN dumped
- THEN text has no `┃`/`╹` and blocks are separated by one blank line

#### Scenario: Streaming appends

- GIVEN partial stream `hel` then `lo`
- WHEN dumped after each chunk
- THEN second dump shows `hello` in the same block

### Requirement: REQ-TUI-CHAT-3 — Per-Prompt Context Stability

Each prompt MUST capture its own `{profile, model}` at submission time via the resolution chain `store.Get` → resolved → default (REQ-CLI-4). A later profile switch via TAB MUST NOT change the context of already-submitted prompts.

#### Scenario: Resolution chain on submit

- GIVEN profile "coder" with no saved model
- WHEN a prompt is submitted
- THEN its model resolves through the chain and lands on the default

#### Scenario: Prior prompts keep their context

- GIVEN a prompt submitted under profile "coder"
- WHEN the user switches to "writer" with TAB
- THEN the earlier prompt still shows `{profile: "coder"}` with its original model

### Requirement: REQ-TUI-CHAT-4 — Markdown Tokens and Syntax

System MUST render markdown via Theme tokens: `markdownText/Heading/Link/LinkText/Code/BlockQuote/Emph/Strong/HRule/ListItem` and syntax `syntax*(comment/keyword/function/variable/string/number/type/operator/punctuation)` (Previously: regex-only, `Background #252525`, `Thought #e0af68`). Fenced blocks MUST use `SyntaxStyle.fromTheme(getSyntaxRules)` not single `HighlightCode(DefaultTheme())`. Inline code MUST use `markdownCode` bg.

#### Scenario: Heading uses markdown token

- GIVEN markdown `# Title`
- WHEN rendered
- THEN style token is `markdownHeading` (code asserts token branch)

#### Scenario: Fenced code uses syntax rules

- GIVEN ```go block
- WHEN rendered
- THEN highlights use per-token syntax colors from theme

### Requirement: REQ-TUI-CHAT-5 — Locale Timestamps and Money

System MUST show timestamps via `Locale.todayTimeOrDateTime` (today → time, older → dateTime), tokens via `toLocaleString`, cost via `Intl.NumberFormat` 2-decimals with `$`, durations via `formatDuration`.

#### Scenario: Recent timestamp shows time

- GIVEN message from today 14:05
- WHEN locale renders
- THEN dump shows `14:05` not full date

#### Scenario: Tokens locale formatted

- GIVEN 319 tokens
- WHEN Chat footer meta renders
- THEN `319 tokens` shows without grouping below 1k, `1,024` with grouping above

### Requirement: REQ-TUI-CHAT-6 — NotAvailable vs Fabrication

System MUST render `workspace`/`permission`/`editor` as muted `NotAvailable` placeholder when backing stores absent. It MUST never fabricate literals `mimo/319k/context7`. `InstallationVersion` only shows `• kui <ver>` when `debug.ReadBuildInfo` present else omitted.

#### Scenario: Workspace falls back to real working directory

- GIVEN no workspace KV store entry
- WHEN session sidebar renders
- THEN the footer shows the process working directory with the user home prefix shortened to `~`
- AND no fabricated path literal appears (Getwd failure renders muted `NotAvailable`)

#### Scenario: Version omitted when empty

- GIVEN `ReadBuildInfo` Main.Version == ""
- WHEN footer renders
- THEN no `• kui` version line appears

#### Scenario: Goldens lock chat

- GIVEN chat with user+assistant+tool parts at 120 cols
- WHEN dumped
- THEN `testdata/chat_*.txt` golden passes

### Requirement: REQ-TUI-CHAT-7 — Shell, Compaction, Thinking, Error Lines

Shell output MUST render as a muted block; compaction as a full-width divider line; thinking as dim italic; errors/status as plain muted transcript lines (never toasts). Unknown tokens MUST be omitted, never fabricated.

#### Scenario: Thinking + compaction

- GIVEN thinking text and a compaction event
- WHEN dumped
- THEN thinking is dim italic and a divider separates pre/post compaction

#### Scenario: Narrow + empty

- GIVEN width 60 and empty transcript
- WHEN dumped
- THEN blocks wrap without panic; empty shows only editor + footer
