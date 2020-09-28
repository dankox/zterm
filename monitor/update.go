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

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
