package views

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/biggs-100/kui/internal/credentials"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// AvailableModels returns the list of selectable model names shown in the
// interactive /model selector. It includes every key from defaultModelPricing
// plus common additional models.
func AvailableModels() []string {
	return []string{
		"gpt-4",
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-3.5-turbo",
		"claude-3.5-sonnet",
		"claude-3.5-haiku",
		"claude-3-opus",
		"claude-3-haiku",
		"gemini-2.0-flash",
		"gemini-1.5-pro",
	}
}

// providerEnvVar maps provider to its required env var.
var providerEnvVar = map[string]string{
	"openai":      "OPENAI_API_KEY",
	"anthropic":   "ANTHROPIC_API_KEY",
	"gemini":      "GEMINI_API_KEY",
	"opencode":    "OPENCODE_API_KEY",
	"opencode-go": "OPENCODE_GO_API_KEY",
}

// globalConfigRoot returns the global config directory (same as store.Store root).
func globalConfigRoot() string {
	if home := os.Getenv("KUI_HOME"); home != "" {
		return home
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "kui")
	}
	return "."
}

// credentialRoots returns global and project roots for credential lookup.
func credentialRoots() []string {
	roots := []string{globalConfigRoot()}
	if cwd, err := os.Getwd(); err == nil && cwd != "" && cwd != roots[0] {
		roots = append(roots, cwd)
	}
	return roots
}

// IsProviderConfigured reports whether the provider has credentials via env, OpenCode auth, or credential stores.
func IsProviderConfigured(provider string) bool {
	if envVar, ok := providerEnvVar[provider]; ok {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	if key, err := credentials.ReadOpenCodeAuth(provider); err == nil && key != "" {
		return true
	}
	for _, root := range credentialRoots() {
		cs := credentials.NewCredentialStore(root)
		if err := cs.Load(); err == nil {
			if key, err := cs.GetAPIKey(provider); err == nil && key != "" {
				return true
			}
		}
	}
	return false
}

// --- Live model discovery (pi-opencode-provider style) ---

var (
	liveCacheMu sync.Mutex
	liveCache   = map[string]cachedModels{}
	liveTTL     = 5 * time.Minute
)

type cachedModels struct {
	models []string
	expiry time.Time
}

func getAPIKeyForProvider(provider string) string {
	if envVar, ok := providerEnvVar[provider]; ok {
		if v := os.Getenv(envVar); v != "" {
			return v
		}
	}
	if key, err := credentials.ReadOpenCodeAuth(provider); err == nil && key != "" {
		return key
	}
	for _, root := range credentialRoots() {
		cs := credentials.NewCredentialStore(root)
		if err := cs.Load(); err == nil {
			if key, err := cs.GetAPIKey(provider); err == nil && key != "" {
				return key
			}
		}
	}
	return ""
}

func fetchOpencodeModels(ctx context.Context, endpoint, apiKey string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("opencode models %d: %s", resp.StatusCode, string(body))
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Object string `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var ids []string
	for _, d := range payload.Data {
		if d.ID != "" {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}

func fetchOpenAIModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	endpoint := baseURL
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	if len(endpoint) > 0 && endpoint[len(endpoint)-1] == '/' {
		endpoint = endpoint[:len(endpoint)-1]
	}
	endpoint += "/models"
	return fetchOpencodeModels(ctx, endpoint, apiKey)
}

func fetchAnthropicModels(ctx context.Context, apiKey string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("anthropic models %d: %s", resp.StatusCode, string(body))
	}
	var payload struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var ids []string
	for _, d := range payload.Data {
		if d.ID != "" {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}

func fetchGeminiModels(ctx context.Context, apiKey string) ([]string, error) {
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("gemini models %d: %s", resp.StatusCode, string(body))
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var ids []string
	for _, m := range payload.Models {
		id := m.Name
		if len(id) > 7 && id[:7] == "models/" {
			id = id[7:]
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func liveModelsForProvider(provider string) []string {
	// check cache
	liveCacheMu.Lock()
	if c, ok := liveCache[provider]; ok && time.Now().Before(c.expiry) && len(c.models) > 0 {
		models := c.models
		liveCacheMu.Unlock()
		return models
	}
	liveCacheMu.Unlock()

	apiKey := getAPIKeyForProvider(provider)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	var ids []string
	var err error
	switch provider {
	case "opencode":
		ids, err = fetchOpencodeModels(ctx, "https://opencode.ai/zen/v1/models", apiKey)
	case "opencode-go":
		ids, err = fetchOpencodeModels(ctx, "https://opencode.ai/zen/go/v1/models", apiKey)
	case "openai":
		ids, err = fetchOpenAIModels(ctx, "https://api.openai.com/v1", apiKey)
	case "anthropic":
		if apiKey == "" {
			err = fmt.Errorf("no api key")
		} else {
			ids, err = fetchAnthropicModels(ctx, apiKey)
		}
	case "gemini":
		if apiKey == "" {
			err = fmt.Errorf("no api key")
		} else {
			ids, err = fetchGeminiModels(ctx, apiKey)
		}
	default:
		return nil
	}
	if err != nil || len(ids) == 0 {
		return nil
	}
	liveCacheMu.Lock()
	liveCache[provider] = cachedModels{models: ids, expiry: time.Now().Add(liveTTL)}
	liveCacheMu.Unlock()
	return ids
}

// AvailableModelsFiltered returns only models whose provider is configured.
// It attempts live discovery (pi-opencode-provider style) for each configured
// provider via their /models endpoints. When nothing is configured, or no
// provider returns models, it returns an empty list (no fabricated IDs); the
// caller renders a help/empty state.
func AvailableModelsFiltered() []string {
	// Try live discovery for configured providers
	providers := []string{"openai", "anthropic", "gemini", "opencode", "opencode-go"}
	var live []string
	for _, prov := range providers {
		if !IsProviderConfigured(prov) {
			continue
		}
		if ids := liveModelsForProvider(prov); len(ids) > 0 {
			live = append(live, ids...)
		}
	}
	if len(live) > 0 {
		// deduplicate while preserving order
		seen := map[string]bool{}
		var out []string
		for _, id := range live {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
		return out
	}
	// No provider configured (or none returned live models): show an empty list
	// rather than fabricated model IDs. The caller renders a help/empty state.
	return nil
}

// model helpers for DialogSelect

func isNanoDisabled(name string) bool {
	return strings.HasPrefix(name, "opencode/") && strings.Contains(name, "-nano")
}

func isFreeModel(name string) bool {
	// Heuristic: models with "free" or known free-tier
	if strings.Contains(strings.ToLower(name), "free") {
		return true
	}
	// Treat mini/haiku as not free; only explicit free
	return false
}

func providerForModel(name string) string {
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		return parts[0]
	}
	if strings.HasPrefix(name, "gpt-") {
		return "openai"
	}
	if strings.HasPrefix(name, "claude-") {
		return "anthropic"
	}
	if strings.HasPrefix(name, "gemini-") {
		return "gemini"
	}
	return "other"
}

func sortModelsFreeTitle(models []string) []string {
	out := make([]string, len(models))
	copy(out, models)
	sort.Slice(out, func(i, j int) bool {
		fi := isFreeModel(out[i])
		fj := isFreeModel(out[j])
		if fi != fj {
			return fi // free first
		}
		// releaseDate not available, fall back to title
		return out[i] < out[j]
	})
	return out
}

func buildModelSelectItems(models []string, current string) []ui.SelectItem[string] {
	// favorites/recent/provider sections: for now, treat first as favorites if name in fav set, second as recent if in recent set
	// Hardcode favorites for demo: gpt-4o, claude-3.5-sonnet
	favorites := map[string]bool{"gpt-4o": true, "claude-3.5-sonnet": true}
	recent := map[string]bool{}
	// recent could be populated from controller; empty for now

	// Sort models by free→title but preserve favorites/recent grouping order
	sorted := sortModelsFreeTitle(models)

	// Reorder: favorites first, then recent, then rest by provider
	var favItems, recentItems, rest []string
	for _, m := range sorted {
		if favorites[m] {
			favItems = append(favItems, m)
		} else if recent[m] {
			recentItems = append(recentItems, m)
		} else {
			rest = append(rest, m)
		}
	}
	ordered := append(append(favItems, recentItems...), rest...)

	items := make([]ui.SelectItem[string], 0, len(ordered))
	for _, name := range ordered {
		cat := providerForModel(name)
		// Determine section for display when flat false? But spec says flat:true, so category still provider for fuzzy
		title := name
		if name == current {
			title = name + " ●"
		}
		detail := ""
		if isFreeModel(name) {
			detail = "Free"
		}
		disabled := isNanoDisabled(name)
		if disabled {
			if detail != "" {
				detail += " • disabled"
			} else {
				detail = "disabled"
			}
		}
		items = append(items, ui.SelectItem[string]{
			Title:    title,
			Category: cat,
			Detail:   detail,
			Value:    name,
			Disabled: disabled,
		})
	}
	return items
}

// ModelListModel wraps DialogSelect for interactive model selection with
// favorites/recent/provider sections, nano disabled, Free badge, free→title sort,
// fuzzy title+category, current dot ●, flat:true.
type ModelListModel struct {
	ds       *ui.DialogSelect[string]
	models   []string
	selected string
	quitting bool
	width    int
	height   int
	styles   *theme.Styles
}

// NewModelListModel creates a ModelListModel from a slice of model names.
func NewModelListModel(models []string, current string, width, height int) ModelListModel {
	items := buildModelSelectItems(models, current)
	ds := ui.NewDialogSelect(items, width, height)
	ds.SetFlat(true)
	ds.SetEmptyView("No models")
	// Ensure current model is selected initially if present
	for i, it := range items {
		if it.Value == current {
			ds.SetSelected(i)
			break
		}
	}
	return ModelListModel{
		ds:     ds,
		models: models,
		width:  width,
		height: height,
		styles: theme.NewStyles(theme.OpenCode()),
	}
}

// SetStyles sets theme styles (for app integration).
func (m *ModelListModel) SetStyles(s *theme.Styles) {
	if s != nil {
		m.styles = s
		m.ds.SetStyles(s)
	}
}

// Update handles keyboard input for the model list with filter→close, preserveSelection, scrollAcceleration.
func (m ModelListModel) Update(msg tea.Msg) (ModelListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if it := m.ds.SelectedItem(); it != nil && !it.Disabled {
				m.selected = it.Value
			}
			return m, tea.Quit
		case tea.KeyEscape:
			if !m.ds.HandleEsc() {
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyBackspace:
			// delegate to ds filter handling via runes? For now, handle backspace as filter edit
			ft := m.ds.FilterText()
			if len(ft) > 0 {
				m.ds.Filter(ft[:len(ft)-1])
			}
			return m, nil
		case tea.KeyRunes:
			if len(msg.Runes) == 1 && msg.Runes[0] == 'q' && m.ds.FilterText() == "" {
				m.quitting = true
				return m, tea.Quit
			}
			ft := m.ds.FilterText() + string(msg.Runes)
			m.ds.Filter(ft)
			return m, nil
		case tea.KeyUp:
			m.ds.MoveUp()
			return m, nil
		case tea.KeyDown:
			m.ds.MoveDown()
			return m, nil
		}
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'k':
				if m.ds.FilterText() == "" {
					m.ds.MoveUp()
					return m, nil
				}
			case 'j':
				if m.ds.FilterText() == "" {
					m.ds.MoveDown()
					return m, nil
				}
			}
		}
	}
	return m, nil
}

// View renders the model list.
func (m ModelListModel) View() string {
	if m.quitting {
		return ""
	}
	m.ds.SetStyles(m.styles)
	return m.ds.View(m.width, m.height)
}

// Selected returns the model name the user selected.
func (m ModelListModel) Selected() string {
	return m.selected
}

// Quitting reports whether the user dismissed the list without selecting.
func (m ModelListModel) Quitting() bool {
	return m.quitting
}
