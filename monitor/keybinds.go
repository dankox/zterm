package monitor

import (
	"log"

	"github.com/jroimartin/gocui"
)

// Keybinds setup key shortcuts for the UI handling
func keybinds(g *gocui.Gui) {
	// quit
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyF10, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, updateLayout); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, changeView); err != nil {
		log.Panicln(err)
	}

	// console
	if err := g.SetKeybinding("", gocui.KeyF1, gocui.ModNone, showConsole); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", '`', gocui.ModAlt, showConsole); err != nil {
		log.Panicln(err)
	}

}
