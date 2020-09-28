package monitor

import (
	"fmt"

	"github.com/jroimartin/gocui"
)

// WidgetFloaty structure for GUI
type WidgetFloaty struct {
	name     string
	body     string
	x, y     int
	width    int
	height   int
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
		// check if current view was pointing to this view before (just to be sure!)
		if g.CurrentView() != nil && g.CurrentView().Name() == wf.name {
			g.SetCurrentView(viewOrder[0])
		}
		return nil
	}
	// Enabled, display...
	maxX, maxY := g.Size()
	// compute correct position and width
	yPos := wf.y
	width := maxX - 1
	if wf.y < 0 {
		yPos = maxY + wf.y
	}
	if wf.width > 0 {
		width = wf.width
	}

	v, err := g.SetView(wf.name, wf.x, yPos, width, yPos+wf.height)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return fmt.Errorf("view %v: %v", wf.name, err)
		}
		fmt.Fprint(v, wf.body)
	}

	// set title
	v.Title = fmt.Sprintf("< %v >", wf.name)
	g.SetViewOnTop(wf.name)

	// set editing
	v.Editable = wf.Editable
	if wf.Editable {
		g.SetCurrentView(wf.name)
	}

	return nil
}

// GetName returns floaty widget name
func (wf *WidgetFloaty) GetName() string {
	return wf.name
}

// IsHidden checks if floaty widget is disabled
func (wf *WidgetFloaty) IsHidden() bool {
	return wf.Enabled == false
}
