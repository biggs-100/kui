# Tasks: tui-redesign — OpenCode-style TUI

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 350-450 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Theme + Logo) → PR 2 (Home View) → PR 3 (Route + Footer) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

---

## PR 1: Theme + Logo (Foundation)

### 1. Create OpenCode Theme

- [x] Create `internal/tui/theme/opencode.go` with color palette
  - BG: #1a1a1a, Text: #e0e0e0, Muted: #808080
  - Accent: #569cd6, Border: #333333, Success: #4ec9b0, Error: #f44747
- [x] Register theme in `internal/tui/theme/theme.go`
- [x] Write test `internal/tui/theme/opencode_test.go` — theme loads, colors valid
  <!-- sdd-owner: implementation -->

### 2. Create ASCII Logo Component

- [x] Create `internal/tui/views/logo.go` — LogoModel struct
  - Method: `View() string` renders ASCII art
  - Uses lipgloss for centering and coloring
  - Accepts width parameter for responsive sizing
- [x] Write test `internal/tui/views/logo_test.go`
  - Test: Logo renders without error
  - Test: Logo uses accent color from theme
  - Test: Logo centers within given width
  <!-- sdd-owner: implementation -->

### 3. Create Home Footer Component

- [x] Create `internal/tui/views/home_footer.go` — HomeFooterModel
  - Renders: `directory • LSP • MCP • /status`
  - LSP/MCP dots: green if connected, gray if not
  - Uses muted colors for text
- [x] Write test `internal/tui/views/home_footer_test.go`
  - Test: Footer renders with all elements
  - Test: LSP dot color changes based on status
  - Test: MCP dot color changes based on status
  <!-- sdd-owner: implementation -->

### 4. PR 1 Verification

- [x] Run `go test ./internal/tui/theme/... ./internal/tui/views/...`
- [x] Run `go build ./cmd/kui`
- [ ] Verify theme loads with `./kui.exe --theme opencode`
  <!-- sdd-owner: parent -->

---

## PR 2: Home View (Core UI)

### 5. Create Home View Component

- [x] Create `internal/tui/views/home.go` — HomeView struct
  - Fields: logo, prompt, footer, width, height
  - Method: `View() string` — renders centered layout
  - Uses `lipgloss.Place()` for centering
  - Composes: logo + spacer + bordered prompt + spacer + footer
- [x] Write test `internal/tui/views/home_test.go`
  - Test: HomeView renders all components
  - Test: Layout is vertically centered
  - Test: Prompt has rounded border
  <!-- sdd-owner: implementation -->

### 6. Create Bordered Prompt Component

- [x] Create `internal/tui/views/home_prompt.go` — HomePromptModel
  - Wraps input with `lipgloss.Border(lipgloss.RoundedBorder())`
  - Placeholder: "Ask kui..."
  - Border color from theme
- [x] Write test `internal/tui/views/home_prompt_test.go`
  - Test: Prompt renders with border
  - Test: Placeholder shows when empty
  - Test: Text input works correctly
  <!-- sdd-owner: implementation -->

### 7. PR 2 Verification

- [x] Run `go test ./internal/tui/views/...`
- [x] Run `go build ./cmd/kui`
- [ ] Visual test: home screen renders centered logo + prompt
  <!-- sdd-owner: parent -->

---

## PR 3: Route System + Integration

### 8. Add Route State to App

- [x] Modify `internal/tui/app.go`
  - Add `route` field: `home` | `session`
  - Add `homeView` field for HomeView
  - Modify `View()` to render based on route
  - Modify `Update()` to handle home-specific keys
- [x] Write test `internal/tui/app_route_test.go`
  - Test: Initial route is home
  - Test: Enter on home switches to session
  - Test: View renders correctly for each route
  <!-- sdd-owner: implementation -->

### 9. Modify Footer for Dual Mode

- [x] Modify `internal/tui/views/footer.go`
  - Add `FooterMode` type: `FooterModeHome` | `FooterModeSession`
  - Add `SetMode(mode FooterMode)` method
  - Home mode: minimal (dir • LSP • MCP)
  - Session mode: detailed (tokens • cost • MCP • LSP)
- [x] Update existing footer tests for backward compatibility
- [x] Add new tests for home mode
  <!-- sdd-owner: implementation -->

### 10. Wire Home Submit to Session

- [x] Modify `internal/tui/app.go`
  - On Enter in home route: capture prompt, switch to session, submit
  - Preserve prompt text for submission
  - Clear home prompt after submission
- [x] Write test for submit transition
  <!-- sdd-owner: implementation -->

### 11. Add Keyboard Shortcuts to Home

- [x] Modify `internal/tui/app.go` handleKey()
  - Ctrl+C: quit (works on both routes)
  - Ctrl+P: command palette (works on both routes)
  - Tab: switch profile (works on both routes)
- [x] Test shortcuts work on home screen
  <!-- sdd-owner: implementation -->

### 12. Integration Testing

- [x] Run full test suite: `go test ./...`
- [x] Run `go build ./cmd/kui`
- [ ] Manual test: `./kui.exe tui` shows home screen
- [ ] Manual test: submit prompt transitions to chat
- [ ] Manual test: all commands work from home
  <!-- sdd-owner: parent -->

### 13. Visual Polish

- [ ] Compare home screen with OpenCode screenshots
- [ ] Adjust spacing, colors, borders as needed
- [ ] Test resize behavior on home screen
- [ ] Test theme switching from home
  <!-- sdd-owner: parent -->

---

## PR 3 Verification

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build -o kui.exe ./cmd/kui`
- [ ] No regressions in existing TUI functionality
- [ ] Home screen matches OpenCode visual style
  <!-- sdd-owner: parent -->
