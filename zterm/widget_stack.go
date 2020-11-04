package zterm

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
)

// WidgetStack structure for GUI (widgets which are stack on each other)
type WidgetStack struct {
	Widget
	pos       int
	stopFun   chan bool
	Fun       func(Widgeter) error
	funStr    string
	refresh   time.Duration
	highlight map[string]bool
}

// NewWidgetStack creates a widget for stack GUI
func NewWidgetStack(name string, pos int, height int, body string) *WidgetStack {
	return &WidgetStack{Widget: Widget{name: name, body: body, width: 0, height: height, Enabled: true}, pos: pos,
		refresh: 5 * time.Second, stopFun: make(chan bool, 1)}
}

// Layout setup for widget
func (ws *WidgetStack) Layout(g *gocui.Gui) error {
	// do not display if disabled
	if !ws.Enabled {
		g.DeleteView(ws.name) // if doesn't exist, don't care
		ws.gview = nil
		return nil
	}
	// Enabled, display...
	maxX, maxY := g.Size()
	yHeight := maxY - 1 // initial height is 100%
	if ws.height > 0 {
		yHeight = maxY * ws.height / viewMaxSize
	}
	yPos := 0
	for _, wl := range getSortedWidgetStack() {
		if wl.Position() < ws.pos {
			yPos += maxY * wl.height / viewMaxSize
		} else {
			break
		}
	}
	// adjust height to maximum if it is last view
	if ws.pos == viewLastPos && yPos+yHeight < maxY {
		yHeight = maxY - yPos - 1
	}

	// save for floaty ;)
	ws.x0 = 0
	ws.y0 = yPos
	ws.x1 = maxX - 1
	ws.y1 = yPos + yHeight
	// overlap, for first in stack, set to 0 (so it looks good :)
	var overlap byte = 1
	if ws.pos == viewFirstPos {
		overlap = 0
	}
	// set view position and dimension
	v, err := g.SetView(ws.name, 0, yPos, maxX-1, yPos+yHeight, overlap)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", ws.name, err)
		}
		fmt.Fprint(v, ws.body)
	}
	ws.gview = v // set pointer to GUI View
	v.FrameColor = cFrame
	if g.CurrentView() == nil {
		g.SetCurrentView(ws.name)
	}

	// set title
	if g.CurrentView() == v {
		v.TitleColor = cFrameSel
		v.Title = fmt.Sprintf("[ %v ]", ws.name)
	} else {
		v.TitleColor = cFrame
		v.Title = fmt.Sprintf("| %v |", ws.name)
	}
	v.Autoscroll = true
	return nil
}

// Keybinds for specific widget
func (ws *WidgetStack) Keybinds(g *gocui.Gui) {
	// special keybinds for the widgets
	// change refresh rate
	if err := g.SetKeybinding(ws.name, gocui.KeyCtrlR, gocui.ModNone, changeRefresh); err != nil {
		log.Panicln(err)
	}
	// cancel key
	if err := g.SetKeybinding(ws.name, gocui.KeyCtrlZ, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if v.Name() == ws.name {
			ws.Disconnect()
		}
		return nil
	}); err != nil {
		log.Panicln(err)
	}
}

// Position returns position in the stack of widgets
func (ws *WidgetStack) Position() int {
	return ws.pos
}

// Print append a text to the widget content.
// Printed line or word will be highlighted if such word exist in `highlight` map in WidgetStack.
func (ws *WidgetStack) Print(str string) {
	if ws.gview != nil {
		ws.gview.Autoscroll = true
		if ws.highlight != nil && len(ws.highlight) > 0 {
			// remove last new line
			if str[len(str)-1] == '\n' {
				str = str[:len(str)-1]
			}
			lines := strings.Split(str, "\n")
			for _, line := range lines {
				written := false
				for sub, hiLine := range ws.highlight {
					if strings.Contains(line, sub) {
						if hiLine {
							// highlight full line
							fmt.Fprintln(ws.gview, colorText(line, cHighlightStr))
							written = true
							break
						}
						// highlight word only
						fmt.Fprintln(ws.gview, strings.ReplaceAll(line, sub, colorText(sub, cHighlightStr)))
						written = true
						break
					}
				}
				// write normal (if not in highlight)
				if !written {
					fmt.Fprintln(ws.gview, line)
				}
			}
		} else {
			// write full text if no highlight map
			fmt.Fprint(ws.gview, str)
		}
	}
}

// SetupFun set function to run in interval in this widget
func (ws *WidgetStack) SetupFun(cmd string) {
	if len(cmd) == 0 {
		return
	}

	ws.funStr = strings.TrimSpace(cmd)
	if strings.HasPrefix(ws.funStr, "remote") {
		ws.Fun = func(w Widgeter) error {
			return cmdSSH(w, strings.TrimPrefix(ws.funStr, "remote"))
		}
	} else {
		ws.Fun = func(w Widgeter) error {
			return cmdShell(w, ws.funStr)
		}
	}
	ws.StartFun()
}

// GetFunString return command running in this widget
func (ws *WidgetStack) GetFunString() string {
	return ws.funStr
}

// StartFun starts a function for the view to update it's content.
// Function has to return string which is used for update
func (ws *WidgetStack) StartFun() {
	// check if function is set
	if ws.Fun == nil {
		return
	}

	// setup goroutine
	go func() {
		// setup action function
		action := func() error {
			if err := ws.Fun(ws); err != nil {
				appendErrorMsgToView(ws, err)
				return err
			}
			return nil
		}
		// run it for the first time
		acterr := action()

		for {
			// To make it possible to kill the Fun, we need to listen to 2 different channels
			// one for stopFun and one for timeout, which would start the action again
			sleepTime := make(chan struct{})
			// sleeping goroutine
			go func() {
				<-time.After(ws.refresh)
				close(sleepTime)
			}()
			if acterr == nil {
				select {
				case <-ws.stopFun:
					ws.Disconnect() // disconnect content channel
					// w.conn.Stop()
					return
				case <-ws.conn.IsEnd():
				}
			}
			select {
			case <-ws.stopFun:
				ws.Disconnect()
				return
			case <-sleepTime:
			}
			acterr = action()
		}
	}()
}

// StopFun stops function running to update widget
func (ws *WidgetStack) StopFun() {
	if ws.Fun == nil {
		return
	}
	select {
	case ws.stopFun <- true:
	default:
		// channel is full, screw another write...
	}
}

// Change refresh rate of the widget content (Fun stuff)
func changeRefresh(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}
	if ws := getWidgetStack(v.Name()); ws != nil {
		ws.StopFun()
		switch ws.refresh {
		case 2 * time.Second:
			ws.refresh = 5 * time.Second
		case 5 * time.Second:
			ws.refresh = 10 * time.Second
		case 10 * time.Second:
			ws.refresh = 2 * time.Second
		}
		ws.StartFun()
		// add notification pop-up
		if wf, err := addSimplePopupWidget("refresh-popup", cPopup, ws.x0+1, ws.y1-4, ws.x1-2, 3,
			fmt.Sprintf("refresh interval changed to %v", ws.refresh)); err == nil {
			// with CtrlR keybind to refresh THIS view (widget, not widget-floaty)
			g.DeleteKeybinding(wf.name, gocui.KeyCtrlR, gocui.ModNone) // don't care about errors (just to not duplicate it)
			if err := g.SetKeybinding(wf.name, gocui.KeyCtrlR, gocui.ModNone,
				func(g *gocui.Gui, v *gocui.View) error {
					nv, err := g.View(ws.GetName())
					if err != nil {
						return err
					}
					return changeRefresh(g, nv)
				}); err != nil {
				log.Panicln(err)
			}
			// with KeyTab keybind to change to NEXT view directly (widget, not widget-floaty)
			g.DeleteKeybinding(wf.name, gocui.KeyTab, gocui.ModNone) // don't care about errors (just to not duplicate it)
			if err := g.SetKeybinding(wf.name, gocui.KeyTab, gocui.ModNone,
				func(g *gocui.Gui, v *gocui.View) error {
					nv, err := g.View(ws.GetName())
					if err != nil {
						return err
					}
					// close this floaty
					closeFloatyWidget(g, v)
					// change ;)
					changeView(g, nv)
					return nil
				}); err != nil {
				log.Panicln(err)
			}

		}
	}
	return nil
}
