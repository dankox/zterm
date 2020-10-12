package monitor

import (
	"fmt"

	"github.com/awesome-gocui/gocui"
)

// wRecvConn struct contains channels for output and error string and error
type wRecvConn struct {
	outchan chan string
	err     chan error
	signal  chan struct{}
}

func changeView(g *gocui.Gui, v *gocui.View) error {
	curr, next := "", ""
	if v != nil {
		curr = v.Name()
	}

	// find next view (from the current)
	if curr == "" {
		// set default only when there is some
		if len(viewOrder) > 0 {
			next = viewOrder[0]
		} else {
			next = widgets[0].GetName()
		}
	} else {
		for i, k := range viewOrder {
			if k == curr {
				if (i + 1) == len(viewOrder) {
					next = viewOrder[0]
				} else {
					next = viewOrder[i+1]
				}
				break
			}
		}
	}
	if next != "" {
		g.SetCurrentView(next)
	} else {
		setDefaultView(g)
	}
	return nil
}

// SetDefaultView to either first one in view list or help view (if none)
func setDefaultView(g *gocui.Gui) {
	if len(viewOrder) > 0 {
		// set to first view in regular layout
		g.SetCurrentView(viewOrder[0])
	} else {
		// set it on Help, if no other view is there
		g.SetCurrentView(widgets[0].GetName())
	}
}

// Put text into the View. This will delete the previous content
func textToView(v *gocui.View, outstr string) {
	if v != nil {
		gui.UpdateAsync(func(g *gocui.Gui) error {
			v.Clear()
			if len(outstr) > 0 {
				v.Autoscroll = true
				fmt.Fprintln(v, outstr)
			}
			return nil
		})
	}
}

// Append text to the View. This will preserve previously added content
func appendTextToView(v *gocui.View, outstr string) {
	if v != nil {
		gui.UpdateAsync(func(g *gocui.Gui) error {
			v.Autoscroll = true
			fmt.Fprintln(v, outstr)
			return nil
		})
	}
}

// Append text to the View. This will preserve previously added content
func appendErrorToView(v *gocui.View, err error) {
	if v != nil {
		gui.UpdateAsync(func(g *gocui.Gui) error {
			v.Autoscroll = true
			fmt.Fprintf(v, "error: %v\n", err.Error())
			g.SelFrameColor = gFrameError
			g.SelFgColor = gFrameError
			return nil
		})
	}
}

// Connect widget view to receive content from channels
func connectWidgetOuput(w WidgetManager, conn *wRecvConn) {
	go func() {
		textToView(w.GetView(), "") // clear the view content
		for out := range conn.outchan {
			// if view is nil, it will just dump to nowhere (so it won't block origin goroutine)
			appendTextToView(w.GetView(), out)
		}
		for err := range conn.err {
			appendErrorToView(w.GetView(), err)
		}
	}()
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
