package monitor

import (
	"fmt"

	"github.com/jroimartin/gocui"
)

var viewSyslog = "syslog"
var viewCmdline = "cmdline"

// Layout function is always called at the end of MainLoop(), so be careful to not update layout outside of it
//  - consume events (like keypress, mouse, resize) and user events (from gocui.Update())
//  - flush () -> calls layout function
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if _, err := g.View(viewSyslog); err == gocui.ErrUnknownView {
		if v, err := g.SetView(viewSyslog, maxX/2-7, maxY/2, maxX/2+7, maxY/2+2); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			fmt.Fprintln(v, "Hello world!")
		}
	}
	return nil
}
