package monitor

import (
	"context"
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
	x0, y0  int // for floaty widgets
	x1, y1  int // for floaty widgets
	gview   *gocui.View
	stopFun chan bool
	cancel  context.CancelFunc
	ctx     context.Context
	refresh time.Duration
	Fun     func(context.Context) (*RecvConn, error)
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

	// save for floaty ;)
	w.x0 = 0
	w.y0 = yPos
	w.x1 = maxX - 1
	w.y1 = yPos + yHeight
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
	if err := g.SetKeybinding(w.name, gocui.KeyCtrlR, gocui.ModNone, changeRefresh); err != nil {
		log.Panicln(err)
	}
	// cancel key
	if err := g.SetKeybinding(w.name, gocui.KeyCtrlZ, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if v.Name() == w.name && w.cancel != nil {
			w.cancel()
		}
		return nil
	}); err != nil {
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

// WithContext create a context for Widget to handle situation when widget is closed
func (w *Widget) WithContext(ctx context.Context) context.Context {
	// cancel previous context if it was set
	w.CancelCtx()
	// create new context with cancel function
	w.ctx, w.cancel = context.WithCancel(ctx)
	return w.ctx
}

// CancelCtx cancel context
func (w *Widget) CancelCtx() {
	if w.cancel != nil {
		w.cancel()
		// nil for garbage collector
		w.cancel = nil
		w.ctx = nil
	}
}

// DoneCtx returns channel that's closed when work is done or context is canceled
func (w *Widget) DoneCtx() <-chan struct{} {
	if w.ctx != nil {
		return w.ctx.Done()
	}
	return nil
}

// StartFun starts a function for the view to update it's content.
// Function has to return string which is used for update
func (w *Widget) StartFun() {
	// check if function is set
	if w.Fun == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel // setup cancel

	// setup goroutine
	go func() {
		// setup action function
		action := func() *RecvConn {
			if wconn, err := w.Fun(ctx); err != nil {
				appendErrorMsgToView(w.GetView(), err)
			} else {
				connectWidgetOuput(w, wconn)
				return wconn
			}
			return nil
		}
		// run it for the first time
		conn := action()

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
			if conn != nil {
				// if connection provided, first wait for end signal (or stopFun)
				select {
				case <-conn.IsEnd():
					// if finished, check sleeptime or stopfun
					select {
					case <-sleepTime:
						conn = action()
					case <-w.stopFun:
						return
					}
				case <-w.stopFun:
					conn.Stop()
					return
				}
			} else {
				select {
				case <-w.stopFun:
					return
				case <-sleepTime:
					// it's after sleep, action finished (no connection opened)
					conn = action()
				}
			}
		}
	}()
}

// StopFun stops function running to update widget
func (w *Widget) StopFun() {
	if w.Fun == nil {
		return
	}
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
