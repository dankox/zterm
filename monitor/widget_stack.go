package monitor

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
	refresh   time.Duration
	highlight map[string]bool
}

var hiColor = "\x1b[35;2m"
var resAnsi = "\x1b[0m"

// NewWidgetStack creates a widget for stack GUI
func NewWidgetStack(name string, pos int, height int, body string) *WidgetStack {
	return &WidgetStack{Widget: Widget{name: name, body: body, width: 0, height: height, Enabled: true}, pos: pos,
		refresh: 5 * time.Second, stopFun: make(chan bool, 1)}
}

// NewHelpWidget creates a widget for GUI
func NewHelpWidget() *WidgetStack {
	return &WidgetStack{Widget: Widget{name: "help-window", width: 0, height: -1, Enabled: true,
		body: `
  Help for zMonitor tool:
    - CTRL+C or F10 to exit the tool
    - ESC to invoke console (can be used to type commands)
    - Tab to swap between windows/views
`}, pos: 0}
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
	for i, view := range viewOrder {
		if i < ws.pos {
			yPos += maxY * config.Views[view] / viewMaxSize
		} else {
			break
		}
	}
	// adjust height to maximum if it is last view
	if ws.pos+1 == len(viewOrder) && yPos+yHeight < maxY {
		yHeight = maxY - yPos - 1
	}

	// save for floaty ;)
	ws.x0 = 0
	ws.y0 = yPos
	ws.x1 = maxX - 1
	ws.y1 = yPos + yHeight
	// overlap, for first in stack, set to 0 (so it looks good :)
	var overlap byte = 1
	if ws.pos == 0 {
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
	if g.CurrentView() == nil && (len(viewOrder) == 0 || ws.name != "help-window") {
		g.SetCurrentView(ws.name)
	}

	// set title
	if g.CurrentView() == v {
		v.Title = fmt.Sprintf("[ %v ]", ws.name)
	} else {
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
							fmt.Fprintf(ws.gview, "%s%s%s\n", hiColor, line, resAnsi)
							written = true
							break
						}
						// highlight word only
						fmt.Fprintln(ws.gview, strings.ReplaceAll(line, sub, hiColor+sub+resAnsi))
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
		if wf, err := addSimplePopupWidget("refresh-popup", gocui.ColorYellow, ws.x0+1, ws.y1-4, ws.x1-2, 3,
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
