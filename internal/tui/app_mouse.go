package tui

import (
	"regexp"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripVisibleANSI removes SGR sequences so a painted row becomes its plain
// visible text.
func stripVisibleANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func clampSel(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// modalOverlayOpen reports whether a modal view currently replaces the whole
// frame, in which case mouse selection over the base frame must be ignored.
func (a *App) modalOverlayOpen() bool {
	return (a.statusMode && a.statusModel != nil) ||
		(a.paletteMode && a.commandPalette != nil) ||
		(a.providerListMode && a.providerList != nil) ||
		(a.modelListMode && a.modelList != nil) ||
		(a.listMode && a.sessionList != nil) ||
		a.loginMode
}

// writeClipboard sends text through the injectable sink (tests) or the real
// platform clipboard tool.
func (a *App) writeClipboard(text string) error {
	if a.copySink != nil {
		return a.copySink(text)
	}
	return copyToClipboard(text)
}

// handleMouse drives the in-app selection (upstream copy-on-select): left
// press anchors, drag extends the highlight, release copies the covered
// text and keeps it highlighted until dismissed. The wheel scrolls the
// conversation.
func (a *App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if a.modalOverlayOpen() {
		return a, nil
	}

	switch msg.Type {
	case tea.MouseLeft:
		a.selActive = true
		a.selStartX = clampSel(msg.X, 0, max(0, a.width-1))
		a.selStartY = clampSel(msg.Y, 0, max(0, a.height-1))
		a.selEndX, a.selEndY = a.selStartX, a.selStartY

	case tea.MouseMotion:
		if a.selActive {
			a.selEndX = clampSel(msg.X, 0, max(0, a.width-1))
			a.selEndY = clampSel(msg.Y, 0, max(0, a.height-1))
		}

	case tea.MouseRelease:
		if !a.selActive {
			break
		}
		a.selEndX = clampSel(msg.X, 0, max(0, a.width-1))
		a.selEndY = clampSel(msg.Y, 0, max(0, a.height-1))
		text := a.extractSelectedText()
		if strings.TrimSpace(text) != "" {
			captured := text
			return a, func() tea.Msg {
				return copyDoneMsg{err: a.writeClipboard(captured)}
			}
		}

	case tea.MouseWheelUp:
		a.scrollVP.SetYOffset(clampSel(a.scrollVP.YOffset-3, 0, max(0, a.vpContentHeight-a.scrollVP.Height)))

	case tea.MouseWheelDown:
		maxOff := max(0, a.vpContentHeight-a.scrollVP.Height)
		a.scrollVP.SetYOffset(clampSel(a.scrollVP.YOffset+3, 0, maxOff))
	}
	return a, nil
}

// selColumns resolves the anchor/head columns into first-row and last-row
// columns for the normalized row range (dragging upward swaps them).
func selColumns(sx, sy, ex, ey int) (firstCol, lastCol int) {
	firstCol, lastCol = sx, ex
	if sy > ey { // dragged upward: anchor is on the LAST row
		firstCol, lastCol = ex, sx
	}
	return
}

// extractSelectedText pulls the visible text of the selected cells from the
// last rendered frame using terminal-style line semantics: the first row
// runs from its start column to end-of-line, middle rows fully, and the last
// row from column zero to its end column.
func (a *App) extractSelectedText() string {
	rows := a.lastRows
	if len(rows) == 0 {
		return ""
	}
	y1, y2 := min(a.selStartY, a.selEndY), max(a.selStartY, a.selEndY)
	y1 = clampSel(y1, 0, len(rows)-1)
	y2 = clampSel(y2, 0, len(rows)-1)
	firstCol, lastCol := selColumns(a.selStartX, a.selStartY, a.selEndX, a.selEndY)

	var out []string
	for y := y1; y <= y2; y++ {
		rs := []rune(rows[y])
		xa, xb := 0, len(rs)-1
		if y == y1 {
			xa = clampSel(firstCol, 0, len(rs)-1)
		}
		if y == y2 {
			xb = clampSel(lastCol, 0, len(rs)-1)
		}
		if xb < xa {
			continue
		}
		out = append(out, strings.TrimRight(string(rs[xa:xb+1]), " "))
	}
	return strings.Join(out, "\n")
}

// applySelectionHighlight repaints the selected cell spans with reverse
// video. Inner styling inside a highlighted span is intentionally flattened
// — terminal selections read as one solid block.
func applySelectionHighlight(rows []string, sx, sy, ex, ey int) []string {
	x1, x2 := min(sx, ex), max(sx, ex)
	y1, y2 := min(sy, ey), max(sy, ey)
	firstCol, lastCol := selColumns(sx, sy, ex, ey)

	for y := y1; y <= y2 && y < len(rows); y++ {
		row := rows[y]
		total := runewidth.StringWidth(row)
		xa, xb := x1, x2
		switch {
		case y1 == y2:
			xa, xb = firstCol, lastCol
		case y == y1:
			xa, xb = firstCol, total-1
		case y == y2:
			xa, xb = 0, lastCol
		default:
			xa, xb = 0, total-1
		}
		xa = clampSel(xa, 0, max(0, total-1))
		xb = clampSel(xb, 0, max(0, total-1))
		if xb < xa {
			continue
		}

		var b strings.Builder
		var span strings.Builder
		v := 0

		flushSpan := func() {
			if span.Len() == 0 {
				return
			}
			b.WriteString("\x1b[7m")
			b.WriteString(span.String())
			b.WriteString("\x1b[0m")
			span.Reset()
		}

		i := 0
		for i < len(row) {
			if row[i] == 0x1b {
				flushSpan()
				end := strings.IndexByte(row[i:], 'm')
				if end < 0 {
					b.WriteString(row[i:])
					i = len(row)
					continue
				}
				b.WriteString(row[i : i+end+1])
				i += end + 1
				continue
			}
			r, size := utf8.DecodeRuneInString(row[i:])
			w := runewidth.RuneWidth(r)
			if w == 0 {
				w = 1 // combining marks ride along their base cell
			}
			if v >= xa && v <= xb {
				span.WriteRune(r)
			} else {
				flushSpan()
				b.WriteString(row[i : i+size])
			}
			v += w
			i += size
		}
		flushSpan()
		rows[y] = b.String()
	}
	return rows
}
