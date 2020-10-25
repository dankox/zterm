package monitor

import (
	"fmt"
	"log"
	"time"

	"github.com/awesome-gocui/gocui"
)

// WidgetStack structure for GUI (widgets which are stack on each other)
type WidgetStack struct {
	Widget
	pos     int
	stopFun chan bool
	Fun     func(Widgeter) error
	refresh time.Duration
}

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
	// set view position and dimension
	v, err := g.SetView(ws.name, 0, yPos, maxX-1, yPos+yHeight, 0)
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
				appendErrorMsgToView(ws.GetView(), err)
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
	if w := getWidgetStack(v.Name()); w != nil {
		w.StopFun()
		switch w.refresh {
		case 2 * time.Second:
			w.refresh = 5 * time.Second
		case 5 * time.Second:
			w.refresh = 10 * time.Second
		case 10 * time.Second:
			w.refresh = 2 * time.Second
		}
		w.StartFun()
		// add notification pop-up
		if wf, err := addSimplePopupWidget("refresh-popup", gocui.ColorYellow, w.x0+1, w.y1-4, w.x1-2, 3,
			fmt.Sprintf("refresh interval changed to %v", w.refresh)); err == nil {
			// with CtrlR keybind to refresh THIS view (widget, not widget-floaty)
			g.DeleteKeybinding(wf.name, gocui.KeyCtrlR, gocui.ModNone) // don't care about errors (just to not duplicate it)
			if err := g.SetKeybinding(wf.name, gocui.KeyCtrlR, gocui.ModNone,
				func(g *gocui.Gui, v *gocui.View) error {
					nv, err := g.View(w.GetName())
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
					nv, err := g.View(w.GetName())
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
