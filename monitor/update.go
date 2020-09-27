package monitor

import (
	"fmt"
	"math/rand"

	"github.com/jroimartin/gocui"
)

func updateLayout(g *gocui.Gui, v *gocui.View) error {

	// gocui.Update() can be called from goroutine to update content
	go g.Update(func(g *gocui.Gui) error {
		maxX, maxY := g.Size()
		delta := rand.Intn(10)
		nv, err := g.SetView(viewSyslog, maxX/2-(7+delta), maxY/2-delta, maxX/2+7+delta, maxY/2+2+delta)
		if err != nil {
			fmt.Printf("zMonitor error: %v", err)
			return err
		}
		nv.Wrap = true
		g.SetCurrentView(viewSyslog)
		g.SetViewOnTop(viewSyslog)
		nv.Clear()
		fmt.Fprintf(nv, "Hello random z/OS world!\nhost = %v\nuser = %v", config.Server.Host, config.Server.User)
		return nil
	})

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
