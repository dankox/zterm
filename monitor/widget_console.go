package monitor

import (
	"fmt"
	"log"
	"strings"

	"github.com/awesome-gocui/gocui"
)

// WidgetConsole structure for GUI
type WidgetConsole struct {
	gview      *gocui.View
	lastView   string
	cmdHistory []string
	histIndex  int
	Enabled    bool
}

var (
	cmdView       = "console"
	cmdPrompt     = "console-prompt"
	consoleHeight = 3
)

// NewWidgetConsole creates a widget for GUI which doesn't contribute to the layout.
// This type of widget is displayed on top over the layout.
func NewWidgetConsole() *WidgetConsole {
	return &WidgetConsole{Enabled: false}
}

// Layout setup for console widget
func (wc *WidgetConsole) Layout(g *gocui.Gui) error {
	// do not display if disabled
	if !wc.Enabled {
		g.DeleteView(cmdView)   // if doesn't exist, don't care
		g.DeleteView(cmdPrompt) // ditto...
		wc.gview = nil
		// check if current view was pointing to this view before (just to be sure!)
		if g.CurrentView() != nil && g.CurrentView().Name() == cmdPrompt {
			if wc.lastView != "" {
				g.SetCurrentView(wc.lastView)
				wc.lastView = ""
			} else {
				setDefaultView(g)
			}
		}
		return nil
	}
	// Enabled, display...
	maxX, maxY := g.Size()
	// compute correct position and width
	yPos := maxY - 1 - consoleHeight
	width := maxX - 1

	// set console "outer" window
	v, err := g.SetView(cmdView, 0, yPos, width, yPos+consoleHeight, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", cmdView, err)
		}
		fmt.Fprint(v, ">> ")
	}
	wc.gview = v // set pointer to GUI View (only for view, not for input)

	// set title
	v.Title = fmt.Sprintf("< %v >", cmdView)
	g.SetViewOnTop(cmdView)

	// set console "input" line
	v, err = g.SetView(cmdPrompt, 3, yPos, width, yPos+2, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", cmdView, err)
		}
		// fmt.Fprint(v, "hello danko")
	}

	// save last CurrentView
	if cv := g.CurrentView(); cv != nil && cv.Name() != cmdPrompt {
		wc.lastView = cv.Name()
	}
	// set editing
	v.Editable = true
	v.Autoscroll = false
	v.Frame = false
	g.SetViewOnTop(cmdPrompt)
	g.SetCurrentView(cmdPrompt)
	v.Editor = gocui.EditorFunc(consoleEditor)

	return nil
}

// Keybinds for specific widget
func (wc *WidgetConsole) Keybinds(g *gocui.Gui) {
	// setup Tab for autocompletion (because it's global key, so to work in console, overwrite)
	if err := g.SetKeybinding(cmdPrompt, gocui.KeyTab, gocui.ModNone, autoComplete); err != nil {
		log.Panicln(err)
	}
}

// GetName returns console widget name
func (wc *WidgetConsole) GetName() string {
	return cmdView
}

// GetView returns console widget GUI View
func (wc *WidgetConsole) GetView() *gocui.View {
	return wc.gview
}

// IsHidden checks if console widget is disabled
func (wc *WidgetConsole) IsHidden() bool {
	return wc.Enabled == false
}

// ExecCmd execute command in the Console Widget
func (wc *WidgetConsole) ExecCmd(cmd string) {
	// add to history and update index
	wc.cmdHistory = append(wc.cmdHistory, cmd)
	wc.histIndex = len(wc.cmdHistory)
	// executing command
	out, msg, err := commandExecute(cmd)
	// clear console output view (cmdView)
	wc.gview.Clear()
	// handle command outputs
	if len(out) > 0 {
		addPopupWidget("console-output", out)
		gui.Cursor = false
	}
	if err != nil {
		fmt.Fprintf(wc.gview, ">> \nerror: %v", err)
	} else if len(msg) > 0 {
		fmt.Fprintf(wc.gview, ">> \n%v", msg)
	} else {
		fmt.Fprint(wc.gview, ">> \n")
	}
}

// PrevHistory go back in history and return command from it
func (wc *WidgetConsole) PrevHistory() string {
	if len(wc.cmdHistory) > 0 {
		wc.histIndex--
		if wc.histIndex >= len(wc.cmdHistory) {
			wc.histIndex = len(wc.cmdHistory)
			return ""
		} else if wc.histIndex < 0 {
			wc.histIndex = 0
			return wc.cmdHistory[0]
		} else {
			return wc.cmdHistory[wc.histIndex]
		}
	}
	return ""
}

// NextHistory go forward in history and return command from it
func (wc *WidgetConsole) NextHistory() string {
	if len(wc.cmdHistory) > 0 {
		wc.histIndex++
		if wc.histIndex >= len(wc.cmdHistory) {
			wc.histIndex = len(wc.cmdHistory)
			return ""
		} else if wc.histIndex < 0 {
			wc.histIndex = 0
			return wc.cmdHistory[0]
		} else {
			return wc.cmdHistory[wc.histIndex]
		}
	}
	return ""
}

// ShowConsole is an update function which should be bound to a key
func showConsole(g *gocui.Gui, v *gocui.View) error {
	if wc := getConsoleWidget(); wc != nil {
		if wc.IsHidden() {
			wc.Enabled = true
			g.Cursor = true
		} else {
			wc.Enabled = false
			g.Cursor = false
			// check if current view was pointing to this view before (just to be sure!)
			if g.CurrentView() != nil && g.CurrentView().Name() == cmdPrompt {
				if wc.lastView != "" {
					g.SetCurrentView(wc.lastView)
					wc.lastView = ""
				} else {
					setDefaultView(g)
				}
			}
		}
		return nil
	}
	// WTF?? what did I do? Where is my console??
	return gocui.ErrUnknownView // this error doesn't comfort the new errors in gocui (but whatev)
}

// Console Editor (special setup for keys)
func consoleEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case ch != 0 && mod == 0:
		v.EditWrite(ch)
	case key == gocui.KeySpace:
		v.EditWrite(' ')
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	case key == gocui.KeyDelete:
		v.EditDelete(false)
	case key == gocui.KeyInsert:
		v.Overwrite = !v.Overwrite
	case key == gocui.KeyEnter:
		// command exec
		if line, err := v.Line(0); err == nil {
			if wc := getConsoleWidget(); wc != nil {
				wc.ExecCmd(line)
			}
			v.Clear()
			v.SetCursor(0, 0)
		}
	case key == gocui.KeyHome:
		x, _ := v.Cursor()
		v.MoveCursor(-x, 1, false)
	case key == gocui.KeyEnd:
		x, y := v.Cursor()
		if line, err := v.Line(y); err == nil {
			v.MoveCursor(len(line)-x, 0, false)
		}
	case key == gocui.KeyArrowDown:
		// command history
		if wc := getConsoleWidget(); wc != nil {
			newcmd := wc.NextHistory()
			v.Clear()
			fmt.Fprint(v, newcmd)
			v.SetCursor(len(newcmd), 0)
		}
	case key == gocui.KeyArrowUp:
		// command history
		if wc := getConsoleWidget(); wc != nil {
			newcmd := wc.PrevHistory()
			v.Clear()
			fmt.Fprint(v, newcmd)
			v.SetCursor(len(newcmd), 0)
		}
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	// from awesome-gocui (new addition?)
	case key == gocui.KeyCtrlU:
		v.EditDeleteToStartOfLine()
	case key == gocui.KeyCtrlA:
		v.EditGotoToStartOfLine()
	case key == gocui.KeyCtrlE:
		v.EditGotoToEndOfLine()
	default:
		v.EditWrite(ch)
	}

}

func autoComplete(g *gocui.Gui, v *gocui.View) error {
	// autocompletion
	if line, err := v.Line(0); err == nil {
		for _, c := range cmdList {
			if strings.HasPrefix(c, line) {
				v.Clear()
				fmt.Fprint(v, c)
				v.SetCursor(len(c), 0)
			}
		}
	}
	return nil
}