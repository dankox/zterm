package zterm

import (
	"fmt"

	"github.com/alecthomas/chroma/quick"
	"github.com/awesome-gocui/gocui"
)

// WidgetPipe structure for pipe which can be put between widget and command
type WidgetPipe struct {
	Widget
	pipedWidget Widgeter
}

// NewWidgetPipe creates a pipe widget wich is used as a pipe between real widget and output sent to it.
func NewWidgetPipe(w Widgeter) *WidgetPipe {
	return &WidgetPipe{Widget: Widget{name: "pipe", Enabled: true}, pipedWidget: w}
}

// Layout setup for floaty widget
func (wp *WidgetPipe) Layout(g *gocui.Gui) error {
	// do not display if disabled
	if !wp.Enabled {
		return nil
	}
	return nil
}

// Print append a text to the widget content
func (wp *WidgetPipe) Print(str string) {
	if wp.pipedWidget != nil {
		// fmt.Fprint(wp.pipedWidget.GetView(), str)
		// quick.Highlight(wp.pipedWidget.GetView(), str, mylexer, "terminal16m", "monokai")
		// lexers.Register()
		quick.Highlight(wp.pipedWidget.GetView(), str, "go", "terminal256", "monokai")
	}
}

// Error append an error text to the widget content
func (wp *WidgetPipe) Error(err error) {
	if wp.pipedWidget != nil {
		fmt.Fprintln(wp.pipedWidget.GetView(), colorText("error:", cErrorStr), err.Error())
	}
}
