package monitor

import (
	"fmt"

	"github.com/awesome-gocui/gocui"
)

// Widget structure for GUI
type Widget struct {
	name    string
	body    string
	x0, y0  int // coordinates top-left
	x1, y1  int // coordinates bottom-right
	height  int
	width   int
	gview   *gocui.View
	conn    *RecvConn
	Enabled bool
}

// Widgeter cover Layout for GUI and some specifics for widgets
type Widgeter interface {
	// Layout is for gocui.GUI
	Layout(*gocui.Gui) error
	Keybinds(*gocui.Gui)
	GetName() string
	GetView() *gocui.View
	IsHidden() bool
	Connect(conn *RecvConn)
	Disconnect()
	CleanPrint(str string)
	Print(str string)
	Error(err error)
}

// NewWidget creates a widget for GUI
func NewWidget(name string, x0 int, y0 int, x1 int, y1 int, body string) *Widget {
	return &Widget{name: name, x0: x0, y0: y0, x1: x1, y1: y1, body: body, Enabled: true}
}

// Layout setup for widget
func (w *Widget) Layout(g *gocui.Gui) error {
	// do not display if disabled
	if !w.Enabled {
		g.DeleteView(w.name) // if doesn't exist, don't care
		w.gview = nil
		return nil
	}
	// set view position and dimension
	v, err := g.SetView(w.name, w.x0, w.y0, w.x1, w.y1, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return fmt.Errorf("view %v: %v", w.name, err)
		}
		fmt.Fprint(v, w.body)
	}
	w.gview = v // set pointer to GUI View
	v.Title = fmt.Sprintf("= %v =", w.name)
	v.Autoscroll = true
	return nil
}

// Keybinds for specific widget
func (w *Widget) Keybinds(g *gocui.Gui) {
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

// Connect content producing channel
func (w *Widget) Connect(conn *RecvConn) {
	if w.conn != nil {
		w.conn.Stop()
	}
	w.conn = conn
}

// Disconnect content producing channel
func (w *Widget) Disconnect() {
	if w.conn != nil {
		w.conn.Stop()
	}
}

// CleanPrint clear and print new text into the widget
func (w *Widget) CleanPrint(str string) {
	if w.gview != nil {
		w.gview.Clear()
		w.gview.SetOrigin(0, 0)
		if len(str) > 0 {
			w.gview.Autoscroll = true
			fmt.Fprint(w.gview, str)
		}
	}
}

// Print append a text to the widget content
func (w *Widget) Print(str string) {
	if w.gview != nil {
		w.gview.Autoscroll = true
		fmt.Fprint(w.gview, str)
	}
}

// Error append an error text to the widget content
func (w *Widget) Error(err error) {
	if w.gview != nil {
		w.gview.Autoscroll = true
		fmt.Fprintf(w.gview, "\x1b[31;1merror: \x1b[0m%v\n", err.Error())
		// g.SelFrameColor = gFrameError // ??
		// g.SelFgColor = gFrameError
	}
}
