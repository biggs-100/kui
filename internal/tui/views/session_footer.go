package views

import "github.com/biggs-100/kui/internal/tui/theme"

// SessionFooterModel is the session footer with tick •⊙ welcome→connected.
// It is an alias to FooterModel (footer.go) for file-per-task traceability (2.4).
type SessionFooterModel = FooterModel

// NewSessionFooterModel creates a SessionFooterModel.
func NewSessionFooterModel(styles *theme.Styles) SessionFooterModel {
	return NewFooterModel(styles)
}
