package monitor

import (
	"fmt"

	"github.com/jroimartin/gocui"
)

// Layout function is always called at the end of MainLoop(), so be careful to not update layout outside of it
//  - consume events (like keypress, mouse, resize) and user events (from gocui.Update())
//  - flush () -> calls layout function
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	yPos := 0
	// setup layout with all the views in correct order
	for _, view := range viewOrder {
		yHeight := maxY * config.Views[view] / viewMaxSize

		if yHeight <= 1 {
			// for small height, remove the view from layout
			g.DeleteView(view) // don't care for ErrUnknownView error

		} else {
			// update the view dimension
			if v, err := g.SetView(view, 0, yPos, maxX-1, yPos+yHeight); err != nil {
				if err != gocui.ErrUnknownView {
					return err
				}
				// view created/updated
				v.Title = view
				fmt.Fprintf(v, "Loading view %v ...", view)
			}
			yPos += yHeight
		}
	}
	return nil
}
