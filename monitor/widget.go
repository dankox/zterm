package monitor

import (
	"fmt"

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
	Enabled bool
}

// NewWidget creates a widget for GUI
func NewWidget(name string, pos int, height int, body string) *Widget {
	return &Widget{name: name, pos: pos, height: height, body: body, Enabled: true}
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
