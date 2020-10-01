package monitor

import (
	"log"

	"github.com/awesome-gocui/gocui"
)

// Global Keybinds setup
func keybindsGlobal(g *gocui.Gui) {
	// quit
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyF10, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	// test keybind for "refresh" -> show config
	if err := g.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, updateLayout); err != nil {
		log.Panicln(err)
	}

	// tab thru next
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, changeView); err != nil {
		log.Panicln(err)
	}

	// console - Esc or ` to turn on (Esc is to turn off too)
	if err := g.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, showConsole); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", '`', gocui.ModNone, showConsole); err != nil {
		log.Panicln(err)
	}
}
