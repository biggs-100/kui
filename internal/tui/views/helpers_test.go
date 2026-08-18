package views

import "github.com/biggs-100/kui/internal/tui/theme"

// testStyles returns default styles for testing.
func testStyles() *theme.Styles {
	return theme.NewStyles(theme.DefaultTheme())
}
