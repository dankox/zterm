package monitor

import (
	"fmt"
	"log"
	"time"

	"github.com/awesome-gocui/gocui"
)

// Widget structure for GUI
type Widget struct {
	name    string
	body    string
	pos     int
	height  int
	width   int
	x, y    int // for floaty widgets
	gview   *gocui.View
	stopFun chan bool
	refresh time.Duration
	Fun     func() (string, error)
	Enabled bool
}

// NewWidget creates a widget for GUI
func NewWidget(name string, pos int, height int, body string) *Widget {
	return &Widget{name: name, pos: pos, height: height, body: body, Enabled: true,
		refresh: 5 * time.Second, stopFun: make(chan bool, 1)}
}

// NewHelpWidget creates a widget for GUI
func NewHelpWidget() *Widget {
	return &Widget{name: "help-window", pos: 0, height: -1, Enabled: true,
		body: `
  Help for zMonitor tool:
    - CTRL+C or F10 to exit the tool
    - ESC to invoke console (can be used to type commands)
    - Tab to swap between windows/views
`}
}

// Layout setup for widget
func (w *Widget) Layout(g *gocui.Gui) error {
	// do not display if disabled
	if !w.Enabled {
		g.DeleteView(w.name) // if doesn't exist, don't care
		w.gview = nil
		return nil
	}
	// Enabled, display...
	maxX, maxY := g.Size()
	yHeight := maxY - 1 // initial height is 100%
	if w.height > 0 {
		yHeight = maxY * w.height / viewMaxSize
	}
	yPos := 0
	for i, view := range viewOrder {
		if i < w.pos {
			yPos += maxY * config.Views[view] / viewMaxSize
		} else {
			break
		}
	}
	// adjust height to maximum if it is last view
	if w.pos+1 == len(viewOrder) && yPos+yHeight < maxY {
		yHeight = maxY - yPos - 1
	}

	// set view position and dimension
	v, err := g.SetView(w.name, 0, yPos, maxX-1, yPos+yHeight, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", w.name, err)
		}
		fmt.Fprint(v, w.body)
	}
	w.gview = v // set pointer to GUI View
	if g.CurrentView() == nil && (len(viewOrder) == 0 || w.name != "help-window") {
		g.SetCurrentView(w.name)
	}

	// set title
	if g.CurrentView() == v {
		v.Title = fmt.Sprintf("[ %v ]", w.name)
	} else {
		v.Title = fmt.Sprintf("| %v |", w.name)
	}
	v.Autoscroll = true
	return nil
}

// Keybinds for specific widget
func (w *Widget) Keybinds(g *gocui.Gui) {
	// special keybinds for the widgets
	// change refresh rate
	if err := g.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, changeRefresh); err != nil {
		log.Panicln(err)
	}
}

// GetName returns widget name
func (w *Widget) GetName() string {
	return w.name
}

// GetView returns widget GUI View
func (w *Widget) GetView() *gocui.View {
	return w.gview
}

// IsHidden checks if widget is disabled
func (w *Widget) IsHidden() bool {
	return w.Enabled == false
}

// StartFun starts a function for the view to update it's content.
// Function has to return string which is used for update
func (w *Widget) StartFun() {
	// check if function is set
	if w.Fun == nil {
		return
	}

	// setup goroutine
	go func() {
		// setup action function
		action := func() {
			output, err := w.Fun()
			gui.Update(func(g *gocui.Gui) error {
				if err != nil {
					fmt.Fprintf(w.gview, "\nerror: %v\n", err.Error())
					return nil
				}
				w.gview.Clear() // clear or add???
				fmt.Fprint(w.gview, output)
				return nil
			})
		}
		// run it for the first time
		action()

		for {
			// To make it possible to kill the Fun, we need to listen to 2 different channels
			// one for stopFun and one for timeout, which would start the action again
			sleepTime := make(chan struct{})
			// sleeping goroutine
			go func() {
				<-time.After(w.refresh)
				close(sleepTime)
			}()
			// check sleep or kill
			select {
			case <-w.stopFun:
				return
			case <-sleepTime:
				// it's after sleep, run action
				action()
			}
		}
	}()
}

// StopFun stops function running to update widget
func (w *Widget) StopFun() {
	select {
	case w.stopFun <- true:
	default:
		// channel is full, screw another write...
	}
}

// Change refresh rate of the widget content (Fun stuff)
func changeRefresh(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}
	if w := getWidget(v.Name()); w != nil {
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
	}
	return nil
}
