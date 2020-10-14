package monitor

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/awesome-gocui/gocui"
)

// WidgetFloaty structure for GUI
type WidgetFloaty struct {
	name     string
	body     string
	x, y     int
	width    int
	height   int
	gview    *gocui.View
	cancel   context.CancelFunc
	ctx      context.Context
	conn     *RecvConn
	Enabled  bool
	Editable bool
}

var pageScroll = 10

// NewWidgetFloaty creates a widget for GUI which doesn't contribute to the layout.
// This type of widget is displayed on top over the layout.
func NewWidgetFloaty(name string, x, y int, width int, height int, body string) *WidgetFloaty {
	return &WidgetFloaty{name: name, x: x, y: y, height: height, width: width, body: body}
}

// Layout setup for floaty widget
func (wf *WidgetFloaty) Layout(g *gocui.Gui) error {
	// do not display if disabled
	if !wf.Enabled {
		g.DeleteView(wf.name) // if doesn't exist, don't care
		wf.gview = nil
		// check if current view was pointing to this view before (just to be sure!)
		if g.CurrentView() != nil && g.CurrentView().Name() == wf.name {
			setDefaultView(g)
		}
		return nil
	}
	// Enabled, display...
	maxX, maxY := g.Size()
	// compute correct position and width
	yPos := wf.y
	xPos := wf.x
	width := maxX - 1
	if wf.y < 0 {
		yPos = maxY + wf.y
		// push from bottom if go out of display
		if (yPos + wf.height) > maxY {
			yPos = maxY - wf.height
		}
	} else if wf.y == 0 {
		yPos = (maxY - wf.height) / 2
		if yPos < 0 {
			yPos = 0
		}
	}
	if wf.width > 0 {
		width = wf.width
	}
	if wf.x == 0 {
		xPos = (maxX - width) / 2
		if xPos < 0 {
			xPos = 0
		}
	}

	v, err := g.SetView(wf.name, xPos, yPos, xPos+width, yPos+wf.height, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", wf.name, err)
		}
		fmt.Fprint(v, wf.body)
		// Autoscroll done manualy (because of later code, to get correct origin)
		_, vy := v.Size()
		v.SetOrigin(0, v.LinesHeight()-vy)
	}
	wf.gview = v // set pointer to GUI View

	// get position & height
	lh := v.LinesHeight()
	_, vy := v.Size()
	_, oy := v.Origin()
	// set title
	v.Title = fmt.Sprintf("< %v - (%v-%v/%v) >", wf.name, oy, oy+vy, lh)
	// v.Wrap = true // set wrapping for long lines
	g.SetViewOnTop(wf.name)

	// set current view for keys and stuff...
	g.SetCurrentView(wf.name)
	g.Highlight = true // highlight the popup

	return nil
}

// Keybinds for specific widget
func (wf *WidgetFloaty) Keybinds(g *gocui.Gui) {
	// Esc close the widget
	if err := g.SetKeybinding(wf.name, gocui.KeyEsc, gocui.ModNone, closeFloatyWidget); err != nil {
		log.Panicln(err)
	}
	// Scrolling
	if err := g.SetKeybinding(wf.name, gocui.KeyPgup, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, -pageScroll)
			return nil
		}); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(wf.name, gocui.KeyPgdn, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, pageScroll)
			return nil
		}); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(wf.name, gocui.KeyHome, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, -v.LinesHeight())
			vx, _ := v.Origin()
			sideScrollView(v, -vx)
			return nil
		}); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(wf.name, gocui.KeyEnd, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, v.LinesHeight())
			vx, _ := v.Origin()
			sideScrollView(v, -vx)
			return nil
		}); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(wf.name, gocui.KeyArrowUp, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, -1)
			return nil
		}); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(wf.name, gocui.KeyArrowDown, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, 1)
			return nil
		}); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(wf.name, gocui.KeyArrowLeft, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			sideScrollView(v, -1)
			return nil
		}); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(wf.name, gocui.KeyArrowRight, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			sideScrollView(v, 1)
			return nil
		}); err != nil {
		log.Panicln(err)
	}
}

// GetName returns floaty widget name
func (wf *WidgetFloaty) GetName() string {
	return wf.name
}

// GetView returns floaty widget GUI View
func (wf *WidgetFloaty) GetView() *gocui.View {
	return wf.gview
}

// IsHidden checks if floaty widget is disabled
func (wf *WidgetFloaty) IsHidden() bool {
	return wf.Enabled == false
}

// WithContext create a context for WidgetFloaty to handle situation when widget is closed
func (wf *WidgetFloaty) WithContext(ctx context.Context) context.Context {
	// cancel previous context if it was set
	wf.CancelCtx()
	// create new context with cancel function
	wf.ctx, wf.cancel = context.WithCancel(ctx)
	return wf.ctx
}

// CancelCtx cancel context
func (wf *WidgetFloaty) CancelCtx() {
	if wf.cancel != nil {
		wf.cancel()
		// nil for garbage collector
		wf.cancel = nil
		wf.ctx = nil
	}
}

// DoneCtx returns channel that's closed when work is done or context is canceled
func (wf *WidgetFloaty) DoneCtx() <-chan struct{} {
	if wf.ctx != nil {
		return wf.ctx.Done()
	}
	return nil
}

func addSimplePopupWidget(name string, color gocui.Attribute, x int, y int, width int, height int, body string) (*WidgetFloaty, error) {
	if color != 0 {
		// set color for the frame
		gui.SelFrameColor = color
		gui.SelFgColor = color
	}
	// Enabled, display...
	maxX, maxY := gui.Size()
	// if width, height is zero, set to max
	if width == 0 {
		width = maxX - 1 // - 10
	}
	if height == 0 {
		height = maxY // - 5 - 10
	} else if height < 0 {
		height = maxY - 5 - 10
	}

	var widget *WidgetFloaty
	// check if exists
	for _, w := range widgets {
		if w.GetName() == name {
			if wf, ok := w.(*WidgetFloaty); ok {
				widget = wf
				break
			} else {
				return nil, errors.New("Widget already exists, but it's not a popup widget")
			}
		}
	}

	// setup widget
	if widget == nil {
		// if it didn't exist, create one
		widget = NewWidgetFloaty(name, x, y, width, height, body)
		widgets = append(widgets, widget)
	} else {
		// otherwise just update size, position and content
		widget.width = width
		widget.height = height
		widget.x = x
		widget.y = y
		// this shouldn't be nil, as it already exists
		if widget.gview != nil {
			widget.gview.Clear()
			fmt.Fprint(widget.gview, body)
		}
	}
	widget.Enabled = true
	widget.Keybinds(gui)
	err := widget.Layout(gui)
	return widget, err
}

func addAsyncPopupWidget(name string, color gocui.Attribute, conn *RecvConn, cncl context.CancelFunc) (*WidgetFloaty, error) {
	// compute correct position and width
	maxX, maxY := gui.Size()
	width := maxX - 1 - 10
	height := maxY - 5 - 10
	// add popup to the center of screen
	widget, err := addSimplePopupWidget(name, color, 0, 0, width, height, "")
	if err != nil {
		return nil, err
	}
	widget.cancel = cncl
	widget.conn = conn
	// run layouts to sort the order (console on top)
	getConsoleWidget().Layout(gui)
	err = widget.Layout(gui)
	connectWidgetOuput(widget, conn)
	return widget, err
}

func closeFloatyWidget(g *gocui.Gui, v *gocui.View) error {
	for i, w := range widgets {
		if w.GetName() == v.Name() {
			if wf, ok := w.(*WidgetFloaty); ok {
				wf.CancelCtx()     // cancel context which was running
				wf.Enabled = false // disable widget and delete the view (set previous view as current)
				wf.Layout(g)
				widgets = append(widgets[:i], widgets[i+1:]...) // remove from widgets list
				if getConsoleWidget().Enabled {
					g.SetCurrentView(cmdPrompt)
				}
				// return highlight colors to the default
				g.SelFrameColor = gFrameHighlight
				g.SelFgColor = gFrameHighlight
			} else {
				panic("Not a WidgetFloaty to close! Something went wrong!")
			}
		}
	}

	return nil
}

func scrollView(v *gocui.View, dy int) error {
	if v != nil {
		v.Autoscroll = false
		ox, oy := v.Origin()
		lh := v.LinesHeight()
		v.Subtitle = ""
		// verify to not scroll out
		if oy+dy < 0 {
			dy = -oy
			v.Subtitle = "[ TOP ]"
		} else if oy+dy >= (lh - 5) {
			dy = lh - oy - 5 // scroll at the bottom to display last 5 lines
			v.Subtitle = "[ BOTTOM ]"
		}
		if err := v.SetOrigin(ox, oy+dy); err != nil {
			return err
		}
	}
	return nil
}

func sideScrollView(v *gocui.View, dx int) error {
	if v != nil {
		v.Wrap = false
		ox, oy := v.Origin()
		// verify to not scroll out
		if ox+dx < 0 {
			dx = -ox
		}
		if err := v.SetOrigin(ox+dx, oy); err != nil {
			return err
		}
	}
	return nil
}
