package monitor

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/awesome-gocui/gocui"
)

// WidgetConsole structure for GUI
type WidgetConsole struct {
	Widget
	lastView   string
	cmdHistory []string
	histIndex  int
	cancel     context.CancelFunc
}

var (
	cmdView       = "console"
	cmdPrompt     = "console-prompt"
	cmdPromptPS1  = "console-prompt-ps1"
	consoleHeight = 3
	promptPS1     = "\x1b[36;2m>>\x1b[0m "
)

// NewWidgetConsole creates a widget for GUI which doesn't contribute to the layout.
// This type of widget is displayed on top over the layout.
func NewWidgetConsole() *WidgetConsole {
	return &WidgetConsole{Widget: Widget{name: cmdView, Enabled: false}}
}

// Layout setup for console widget
func (wc *WidgetConsole) Layout(g *gocui.Gui) error {
	// do not display if disabled
	if !wc.Enabled {
		g.DeleteView(cmdView)      // if doesn't exist, don't care
		g.DeleteView(cmdPrompt)    // ditto...
		g.DeleteView(cmdPromptPS1) // ditto...
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
	maxHeight := consoleHeight
	if wc.gview != nil {
		if wc.gview.LinesHeight() > 0 {
			maxHeight += wc.gview.LinesHeight() - 1
		}
		if maxHeight > int(float64(maxY)*0.6) {
			maxHeight = int(float64(maxY) * 0.6)
		}
	}
	yPos := maxY - 1 - maxHeight
	width := maxX - 1

	// set console "outer" window
	v, err := g.SetView(cmdView, 0, yPos, width, yPos+maxHeight, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", cmdView, err)
		}
		// wc.gview = v // set pointer to GUI View for wc.Clear() command
		// wc.Clear()
	}
	wc.gview = v // set pointer to GUI View (only for view, not for input)
	// hardcoded colors for frame and title
	v.FrameColor = gFrameOk
	v.TitleColor = gFrameOk

	// set title
	v.Title = fmt.Sprintf("< %v >", cmdView)
	g.SetViewOnTop(cmdView)

	// set consol prompt PS1
	v, err = g.SetView(cmdPromptPS1, 0, yPos+maxHeight-2, 4, yPos+maxHeight, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", cmdView, err)
		}
		fmt.Fprint(v, promptPS1)
	}
	v.Frame = false
	g.SetViewOnTop(cmdPromptPS1)

	// set console "input" line
	v, err = g.SetView(cmdPrompt, 3, yPos+maxHeight-2, width, yPos+maxHeight, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", cmdView, err)
		}
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

// Clear override to not clear the console output (this is triggered everytime new command is issued in update)
func (wc *WidgetConsole) Clear() {
	// wc.gview.Clear()
}

// Error print error message to the console output line (second line below prompt)
func (wc *WidgetConsole) Error(err error) {
	wc.gview.Autoscroll = true
	fmt.Fprintf(wc.gview, "\x1b[31;1merror: \x1b[0m%v\n\n", err.Error())
}

// Print message to the console output line
func (wc *WidgetConsole) Print(msg string) {
	wc.gview.Autoscroll = true
	fmt.Fprint(wc.gview, msg)
}

// Println prints message to the console output line and add new line at the end
func (wc *WidgetConsole) Println(msg string) {
	wc.gview.Autoscroll = true
	fmt.Fprintln(wc.gview, msg)
}

// Printf print formatted message to the console output line (second line below prompt)
func (wc *WidgetConsole) Printf(format string, a ...interface{}) {
	wc.gview.Autoscroll = true
	fmt.Fprintf(wc.gview, format, a...) // should there be ... ???
}

// Keybinds for specific widget
func (wc *WidgetConsole) Keybinds(g *gocui.Gui) {
	// setup Tab for autocompletion (because it's global key, so to work in console, overwrite)
	if err := g.SetKeybinding(cmdPrompt, gocui.KeyTab, gocui.ModNone, autoComplete); err != nil {
		log.Panicln(err)
	}
	// cancel key
	if err := g.SetKeybinding(cmdPrompt, gocui.KeyCtrlZ, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if v.Name() == cmdPrompt && wc.cancel != nil {
			wc.cancel()
		}
		return nil
	}); err != nil {
		log.Panicln(err)
	}
}

// ExecCmd execute command in the Console Widget
func (wc *WidgetConsole) ExecCmd(cmd string) {
	// add to history and update index
	wc.cmdHistory = append(wc.cmdHistory, cmd)
	wc.histIndex = len(wc.cmdHistory)
	if err := commandExecute(wc, cmd); err != nil {
		wc.Println(promptPS1 + cmd)
		wc.Error(err)
	} else {
		wc.Println(promptPS1 + cmd)
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
		} else {
			wc.Disconnect()
			wc.Enabled = false
			wc.Layout(g)
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
	var wc *WidgetConsole
	if wc = getConsoleWidget(); wc == nil {
		// if no console widget... wtf are we doing here??
		return
	}

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
			wc.ExecCmd(line)
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

	// these are for console output view (not for the actual command line)
	case key == gocui.KeyPgup:
		if wc := getConsoleWidget(); wc != nil {
			scrollView(wc.gview, -10)
		}

	case key == gocui.KeyPgdn:
		if wc := getConsoleWidget(); wc != nil {
			scrollView(wc.gview, 10)
		}

	default:
		v.EditWrite(ch)
	}

}

func autoComplete(g *gocui.Gui, v *gocui.View) error {
	// autocompletion
	if line, err := v.Line(0); err == nil {
		cmdParts := strings.Split(line, " ")
		clen := len(cmdParts)
		var final []string
		if clen == 1 {
			autocomp := cmdParts[0]
			for c := range cmdAuto {
				if strings.HasPrefix(c, autocomp) {
					final = append(final, c)
				}
			}
		} else if clen == 2 {
			autocomp := cmdParts[1]
			for _, c := range cmdAuto[cmdParts[0]] {
				if strings.HasPrefix(c, autocomp) {
					final = append(final, c)
				}
			}
		}
		// finish command or process output for console message
		if len(final) == 1 {
			v.Clear()
			var finalcmd string
			if clen == 1 {
				finalcmd = final[0] + " "
			} else if clen == 2 {
				finalcmd = cmdParts[0] + " " + final[0] + " "
			}
			fmt.Fprint(v, finalcmd)
			v.SetCursor(len(finalcmd), 0)
		} else if len(final) > 1 {
			wc := getConsoleWidget()
			wc.Println(strings.Join(final, " "))
		}
	}
	return nil
}
