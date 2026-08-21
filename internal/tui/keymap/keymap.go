package keymap

import "strings"

const BaseLayer = "base"
const ModalLayer = "modal"
const Leader = "<leader>"

type Binding struct {
	Layer, Name string
	Keys        []string
	Desc        string
}
type Keymap struct{ stack []string }

func New() *Keymap              { return &Keymap{stack: []string{BaseLayer}} }
func (k *Keymap) Push(l string) { k.stack = append(k.stack, l) }
func (k *Keymap) Pop() {
	if len(k.stack) > 1 {
		k.stack = k.stack[:len(k.stack)-1]
	}
}
func (k *Keymap) Current() string {
	if len(k.stack) == 0 {
		return BaseLayer
	}
	return k.stack[len(k.stack)-1]
}

func FormatKeyBindings(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		switch strings.ToLower(k) {
		case "leader":
			parts[i] = Leader
		case "pgup", "pageup":
			parts[i] = "pgup"
		case "pgdown", "pagedown":
			parts[i] = "pgdown"
		default:
			parts[i] = k
		}
	}
	return strings.Join(parts, "+")
}

func AllBindings() []Binding {
	return []Binding{
		{BaseLayer, "command.palette.show", []string{"ctrl+p"}, "Show palette"},
		{BaseLayer, "session.list", []string{"ctrl+s"}, "List sessions"},
		{BaseLayer, "session.new", []string{"ctrl+n"}, "New session"},
		{ModalLayer, "dialog.close", []string{"esc"}, "Close"},
		{ModalLayer, "dialog.close.ctrlc", []string{"ctrl+c"}, "Close"},
		{ModalLayer, "dialog.select.up", []string{"up", "k"}, "Move up"},
		{ModalLayer, "dialog.select.down", []string{"down", "j"}, "Move down"},
		{ModalLayer, "dialog.select.confirm", []string{"enter"}, "Confirm"},
		{ModalLayer, "dialog.select.cancel", []string{"esc"}, "Cancel"},
		{ModalLayer, "dialog.select.filter", []string{"backspace"}, "Filter"},
	}
}
