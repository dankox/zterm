package zterm

import (
	"errors"
	"fmt"
	"log"

	"github.com/awesome-gocui/gocui"
)

// WidgetFloaty structure for GUI
type WidgetFloaty struct {
	Widget
	Editable bool
}

var pageScroll = 10

// NewWidgetFloaty creates a widget for GUI which doesn't contribute to the layout.
// This type of widget is displayed on top over the layout.
func NewWidgetFloaty(name string, x, y int, width int, height int, body string) *WidgetFloaty {
	return &WidgetFloaty{Widget: Widget{name: name, body: body, x0: x, y0: y, width: width, height: height, Enabled: true}}
}

// PopupHelpWidget creates a pop help widget for GUI
func PopupHelpWidget() *WidgetFloaty {
	wf, err := addSimplePopupWidget("help-window", gocui.ColorYellow, 0, 0, 0, 10, `
Help for zMonitor tool:
	- CTRL+C or F10 to exit the tool
	- ESC to invoke console (can be used to type commands)
	- Tab to swap between windows/views`)
	if err != nil {
		return nil
	}
	return wf
}

// Layout setup for floaty widget
func (wf *WidgetFloaty) Layout(g *gocui.Gui) error {
	// do not display if disabled
	if !wf.Enabled {
		g.DeleteKeybindings(wf.name)
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
	yPos := wf.y0
	xPos := wf.x0
	width := maxX - 1
	if wf.y0 < 0 {
		yPos = maxY + wf.y0
		// push from bottom if go out of display
		if (yPos + wf.height) > maxY {
			yPos = maxY - wf.height
		}
	} else if wf.y0 == 0 {
		yPos = (maxY - wf.height) / 2
		if yPos < 0 {
			yPos = 0
		}
	}
	if wf.width > 0 {
		width = wf.width
	}
	if wf.x0 == 0 {
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
	v.FrameColor = wf.FrameColor
	v.TitleColor = wf.TitleColor

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

func addSimplePopupWidget(name string, color gocui.Attribute, x int, y int, width int, height int, body string) (*WidgetFloaty, error) {
	// Enabled, display...
	maxX, maxY := gui.Size()
	// if width, height is zero, set to max
	if width == 0 {
		width = maxX - 1 - x // - 10
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
		widget.x0 = x
		widget.y0 = y
		// this shouldn't be nil, as it already exists
		if widget.gview != nil {
			widget.gview.Clear()
			fmt.Fprint(widget.gview, body)
		}
	}
	widget.Enabled = true
	widget.Keybinds(gui)
	err := widget.Layout(gui)
	if color != 0 {
		// set color for the frame
		widget.FrameColor = color
		widget.TitleColor = color
	}
	return widget, err
}

func closeFloatyWidget(g *gocui.Gui, v *gocui.View) error {
	for i, w := range widgets {
		if w.GetName() == v.Name() {
			if wf, ok := w.(*WidgetFloaty); ok {
				wf.Disconnect()    // disconnect content channel
				wf.Enabled = false // disable widget and delete the view (set previous view as current)
				wf.Layout(g)
				widgets = append(widgets[:i], widgets[i+1:]...) // remove from widgets list
				if getConsoleWidget().Enabled {
					g.SetCurrentView(cmdPrompt)
				}
			} else {
				panic("Not a WidgetFloaty to close! Something went wrong!")
			}
		}
	}

	return nil
}
