package zterm

import (
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
)

// RecvConn struct contains channels for output and error string and error
type RecvConn struct {
	outchan chan string
	err     chan error
	signal  chan struct{}
	sigEnd  chan bool
}

// NewRecvConn create new connection to receive output from command/function
func NewRecvConn() *RecvConn {
	return &RecvConn{
		outchan: make(chan string, 10),
		err:     make(chan error, 1),
		signal:  make(chan struct{}),
		sigEnd:  make(chan bool, 1),
	}
}

// Stop command/function origin by sending ending signal.
//
// *it can be called multiple times, it doesn't block*
func (conn *RecvConn) Stop() {
	select {
	case <-conn.signal:
	default:
		close(conn.signal)
	}
}

// IsEnd return ending signal. Check if it's closed to confirm if it is end.
func (conn *RecvConn) IsEnd() <-chan struct{} {
	return conn.signal
}

// WaitEnd blocks the processing until the connected command/function ends.
func (conn *RecvConn) WaitEnd() {
	<-conn.signal
	return
}

func (conn *RecvConn) send() {
}

func changeView(g *gocui.Gui, v *gocui.View) error {
	curr, next := "", ""
	if v != nil {
		curr = v.Name()
	}

	// find next view in WidgetStack (from current)
	next = nextWidgetStack(curr)
	if next != "" {
		g.SetCurrentView(next)
	} else {
		setDefaultView(g)
	}
	return nil
}

func nextWidgetStack(name string) (next string) {
	wslist := getSortedWidgetStack()
	nextidx := 0
	for i, ws := range wslist {
		if ws.GetName() == name {
			nextidx = i + 1
			break
		}
	}
	if nextidx >= len(wslist) && len(wslist) > 0 {
		next = wslist[0].GetName()
	} else if len(wslist) > 0 {
		next = wslist[nextidx].GetName()
	}
	return
}

// SetDefaultView to first one in WidgetStack list (if none, do not set).
func setDefaultView(g *gocui.Gui) {
	if wslist := getSortedWidgetStack(); len(wslist) > 0 {
		g.SetCurrentView(wslist[0].GetName())
	}
}

// Put text into the View. This will delete the previous content
func textToView(w Widgeter, outstr string) {
	if w != nil && !w.IsHidden() {
		gui.UpdateAsync(func(g *gocui.Gui) error {
			w.Clear()
			if len(outstr) > 0 {
				w.Print(outstr)
			}
			return nil
		})
	}
}

// Append text to the View. This will preserve previously added content
func appendTextToView(w Widgeter, outstr string) {
	if w != nil && !w.IsHidden() {
		gui.UpdateAsync(func(g *gocui.Gui) error {
			w.Print(outstr)
			return nil
		})
	}
}

// Append highlighter error message to the View. This will preserve previously added content
func appendErrorMsgToView(w Widgeter, err error) {
	if w != nil {
		gui.UpdateAsync(func(g *gocui.Gui) error {
			w.Error(err)
			return nil
		})
	}
}

// Connect widget view to receive content from channels
func connectWidgetOuput(w Widgeter, conn *RecvConn) {
	if conn == nil {
		return
	}

	// connect widget for communication
	w.Connect(conn)

	go func() {
		output := ""
		first := true
	renderloop:
		for {
			select {
			case <-suspendChan:
				// wait for resume
				<-resumeChan
			case out, ok := <-conn.outchan:
				output += out + "\n"
				if !ok {
					if len(strings.TrimSpace(output)) > 0 {
						if first {
							textToView(w, output)
							first = false
						} else {
							appendTextToView(w, output)
						}
					}
					// don't need to clean output, just break out
					break renderloop
				}
			case <-time.After(16 * time.Millisecond):
				// display in FPS ~60hz
				if len(strings.TrimSpace(output)) > 0 {
					// display only when there is some text
					if first {
						textToView(w, output)
						first = false
					} else {
						appendTextToView(w, output)
					}
					output = ""
				}
			}

		}
		// add to renderloop???
		for err := range conn.err {
			appendErrorMsgToView(w, err)
		}
		conn.Stop()
	}()
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func scrollView(v *gocui.View, dy int) error {
	if v != nil {
		v.Autoscroll = false
		ox, oy := v.Origin()
		lh := v.LinesHeight()
		v.Subtitle = ""
		// verify to not scroll out
		if oy+dy < 0 {
			dy = -oy
			v.Subtitle = "[ TOP ]"
		} else if oy+dy >= (lh - 5) {
			dy = lh - oy - 5 // scroll at the bottom to display last 5 lines
			v.Subtitle = "[ BOTTOM ]"
		}
		if err := v.SetOrigin(ox, oy+dy); err != nil {
			return err
		}
	}
	return nil
}

func sideScrollView(v *gocui.View, dx int) error {
	if v != nil {
		v.Wrap = false
		ox, oy := v.Origin()
		// verify to not scroll out
		if ox+dx < 0 {
			dx = -ox
		}
		if err := v.SetOrigin(ox+dx, oy); err != nil {
			return err
		}
	}
	return nil
}
