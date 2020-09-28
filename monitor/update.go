package monitor

import (
	"fmt"

	"github.com/jroimartin/gocui"
)

func updateLayout(g *gocui.Gui, v *gocui.View) error {

	// gocui.Update() can be called from goroutine to update content
	go g.Update(func(g *gocui.Gui) error {
		nv := g.CurrentView()
		if nv == nil {
			return nil
		}
		nv.Wrap = true
		nv.Clear()
		fmt.Fprintf(nv, "Hello random z/OS world!\nhost = %v\nuser = %v\n", config.Server.Host, config.Server.User)
		fmt.Fprintf(nv, "views = %v\n", config.Views)
		return nil
	})

	return nil
}

func changeView(g *gocui.Gui, v *gocui.View) error {
	next := ""
	if v != nil {
		next = v.Name()
	}

	// find next view (from the current)
	if next == "" {
		// set default only when there is some
		if len(viewOrder) > 0 {
			next = viewOrder[0]
		} else {
			next = widgets[0].GetName()
		}
	} else {
		for i, k := range viewOrder {
			if k == next {
				if (i + 1) == len(viewOrder) {
					next = viewOrder[0]
				} else {
					next = viewOrder[i+1]
				}
				break
			}
		}
	}
	g.SetCurrentView(next)
	return nil
}

func showConsole(g *gocui.Gui, v *gocui.View) error {
	for _, w := range widgets {
		if w.GetName() == cmdView {
			// console should be floaty widget
			wf, ok := w.(*WidgetFloaty)
			if ok {
				if w.IsHidden() {
					wf.Editable = true
					wf.Enabled = true
				} else {
					wf.Editable = false
					wf.Enabled = false
					// unset current view
					if g.CurrentView() != nil && g.CurrentView().Name() == wf.name {
						if len(viewOrder) > 0 {
							// set to first view in regular layout
							g.SetCurrentView(viewOrder[0])
						} else {
							// set it on Help, if no other view is there
							g.SetCurrentView(widgets[0].GetName())
						}
					}
				}
			} else {
				panic("WTF did you do? How did you setup console???")
			}
			return nil
		}
	}
	return gocui.ErrUnknownView
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
