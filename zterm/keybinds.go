package zterm

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

	// tab thru next
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, changeView); err != nil {
		log.Panicln(err)
	}

	// help
	if err := g.SetKeybinding("", gocui.KeyF1, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		for _, w := range widgets {
			if w.GetName() == "help-window" {
				return closeFloatyWidget(g, v)
			}
		}
		PopupHelpWidget()
		return nil
	}); err != nil {
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
