# tui-chat Specification

## Purpose

The chat view shows the conversation: user prompts, streaming assistant answers, and explicit `{profile, model}` context on every prompt.

## Requirements

### Requirement: REQ-TUI-CHAT-1 — Prompt Submission

The user MUST be able to type a prompt in the input and submit it with Enter. The submitted prompt MUST be sent to the agent loop carrying the session's active `{profile, model}` and MUST appear in the message view. Submission display MUST use left-border `SplitBorder` with agent color (not plain `you` label).

#### Scenario: Submit a prompt

- GIVEN the TUI running with active profile "coder"
- WHEN the user types a prompt and presses Enter
- THEN the prompt appears in the message view
- AND it is sent carrying `{profile: "coder", model: <resolved>}`

#### Scenario: Empty input ignored

- GIVEN an empty or whitespace-only input
- WHEN the user presses Enter
- THEN no prompt is submitted
- AND the message view is unchanged

#### Scenario: Submitted prompt shows with border

- GIVEN prompt "hello" submitted under profile coder
- WHEN chat dumps
- THEN user part shows `┃` border not `you:` label

### Requirement: REQ-TUI-CHAT-2 — Streaming Answer Rendering

While provider streams, chat MUST render answer incrementally per-part (`text/reasoning/tool/file/compaction`) via `PART_MAPPING`. Each part MUST render with left `SplitBorder` (`┃` vertical, `╹` bottom) colored by agent, hover uses `backgroundElement`, `QUEUED` badge, compaction divider, `stickyScroll` with acceleration. Inline diagnostics MUST stay below affected lines without interrupting flow.
(Previously: regex markdown only, `(profile/model)` faint, `HomeMuted` status)

#### Scenario: Per-part split border

- GIVEN assistant answer with two parts
- WHEN dumped
- THEN each part has `┃` left border and `╹` terminator

#### Scenario: Queued badge shows

- GIVEN queued prompt part
- WHEN rendered
- THEN dump contains `QUEUED` badge

#### Scenario: Hover background

- GIVEN hover over user part
- WHEN rendered state with hover=true
- THEN dump marker indicates `backgroundElement` path (verified via style token presence in code, text fallback `hover`)

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

System MUST render `workspace`/`permission`/`editor` as muted `NotAvailable` placeholder when backing stores absent. It MUST never fabricate literals `mimo/319k/context7`. `InstallationVersion` only shows `• Open Code <ver>` when `debug.ReadBuildInfo` present else omitted.

#### Scenario: Missing workspace shows muted

- GIVEN no workspace store
- WHEN session sidebar/header renders
- THEN dump shows `NotAvailable` muted not fabricated path

#### Scenario: Version omitted when empty

- GIVEN `ReadBuildInfo` Main.Version == ""
- WHEN footer renders
- THEN no `• Open Code` version line appears

#### Scenario: Goldens lock chat

- GIVEN chat with user+assistant+tool parts at 120 cols
- WHEN dumped
- THEN `testdata/chat_*.txt` golden passes
