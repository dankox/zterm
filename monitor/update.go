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
		next = viewOrder[0]
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

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
