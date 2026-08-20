# Apply Progress: tui-redesign

## PR 1: Theme + Logo (Foundation) — COMPLETE

### Files Changed
- `internal/tui/theme/opencode.go` — Created OpenCode theme with dark palette (#1a1a1a BG, #569cd6 accent)
- `internal/tui/theme/theme.go` — Modified to register built-in themes via `builtinThemes` map
- `internal/tui/theme/styles.go` — Added LogoAccent, HomeBorder, HomeMuted styles
- `internal/tui/views/logo.go` — Created LogoModel with ASCII art and centering
- `internal/tui/views/home_footer.go` — Created HomeFooterModel with LSP/MCP dot indicators

### Tests Written
- `internal/tui/theme/opencode_test.go` — Theme loads, colors valid
- `internal/tui/views/logo_test.go` — Logo renders, uses accent, centers within width
- `internal/tui/views/home_footer_test.go` — Footer renders all elements, dot colors change

### Verification
- ✅ `go test ./internal/tui/theme/... ./internal/tui/views/...` — PASS
- ✅ `go build ./cmd/kui` — PASS

---

## PR 2: Home View (Core UI) — COMPLETE

### Files Changed
- `internal/tui/views/home.go` — Created HomeView compositing logo + prompt + vertical centering
- `internal/tui/views/home_prompt.go` — Created HomePromptModel with rounded border and placeholder

### Tests Written
- `internal/tui/views/home_test.go` — Renders all components, vertical centering, resize, input
- `internal/tui/views/home_prompt_test.go` — Border rendering, placeholder, input, submit/clear

### Verification
- ✅ `go test ./internal/tui/views/...` — PASS
- ✅ `go build ./cmd/kui` — PASS

---

## PR 3: Route System + Integration — COMPLETE

### Files Changed
- `internal/tui/app.go` — Modified to add route system:
  - Added `route` field (home/session) and `homeView` field
  - Added `renderHome()` method for home screen rendering
  - Modified `View()` to dispatch based on route
  - Modified `handleKey()` to handle Enter on home → session transition
  - Modified `rebuildViews()` to rebuild homeView

### Tests Written
- `internal/tui/app_route_test.go` — 8 tests covering:
  - Initial route is home
  - Enter on home switches to session
  - Empty enter stays on home
  - View renders correctly for each route
  - Ctrl+C works on home
  - Tab cycles profile on home
  - Ctrl+P opens palette on home
  - Submit captures prompt correctly

### Verification
- ✅ `go test ./...` — ALL PASS (27 packages)
- ✅ `go build -o kui.exe ./cmd/kui` — PASS
- ✅ No regressions in existing TUI functionality

---

## Design Decisions & Deviations

1. **Footer approach**: Created separate `HomeFooterModel` instead of modifying existing `FooterModel` with a mode parameter. This is cleaner and avoids breaking existing footer code. The spec's `FooterMode` type was not needed.

2. **Theme registration**: Added a `builtinThemes` map in `theme.go` for built-in themes (kui-default, opencode) alongside file-based discovery. This allows themes to work without JSON files on disk.

3. **Logo color**: Uses `LogoAccent` style from the theme's `Accent` color field, providing consistent branding across themes.

## Task Completion

- Implementation tasks: 26/39 complete
- Parent-owned tasks: 0/13 complete (deferred to parent lifecycle)
- Deferred parent actions: 13 remaining (manual tests, visual polish, verification)
