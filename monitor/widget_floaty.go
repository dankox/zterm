package monitor

import (
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
	Enabled  bool
	Editable bool
}

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
	}
	wf.gview = v // set pointer to GUI View

	// set title
	v.Title = fmt.Sprintf("< %v >", wf.name)
	g.SetViewOnTop(wf.name)

	// set current view for keys and stuff...
	g.SetCurrentView(wf.name)
	v.Autoscroll = true

	return nil
}

// Keybinds for specific widget
func (wf *WidgetFloaty) Keybinds(g *gocui.Gui) {
	// Esc close the widget
	if err := g.SetKeybinding(wf.name, gocui.KeyEsc, gocui.ModNone, closeFloatyWidget); err != nil {
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

func addPopupWidget(name string, body string) {
	// Enabled, display...
	maxX, maxY := gui.Size()
	// compute correct position and width
	width := maxX - 1 - 10
	height := maxY - 5 - 10

	// prepare widgets
	widget := NewWidgetFloaty(name, 0, 0, width, height, body)
	widget.Enabled = true
	widgets = append(widgets, widget)
	widget.Keybinds(gui)
	// run layouts to sort the order (console on top)
	getConsoleWidget().Layout(gui)
	widget.Layout(gui)
}

func closeFloatyWidget(g *gocui.Gui, v *gocui.View) error {
	for i, w := range widgets {
		if w.GetName() == v.Name() {
			if wf, ok := w.(*WidgetFloaty); ok {
				wf.Enabled = false
				wf.Layout(g)                                    // delete the view and set previous view as current
				widgets = append(widgets[:i], widgets[i+1:]...) // remove from widgets
				if getConsoleWidget().Enabled {
					g.Cursor = true
					g.SetCurrentView(cmdPrompt)
				}
			} else {
				panic("Not a WidgetFloaty to close! Something went wrong!")
			}
		}
	}

	return nil
}
