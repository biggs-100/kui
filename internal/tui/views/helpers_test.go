package views

import (
	"flag"

	"github.com/biggs-100/kui/internal/tui/theme"
)

// update regenerates golden files (PR4 owns regeneration; PR2 must NOT pass
// -update — goldens stay failing and are reported).
var update = flag.Bool("update", false, "update golden files")

// testStyles returns default styles for testing.
func testStyles() *theme.Styles {
	return theme.NewStyles(theme.DefaultTheme())
}
