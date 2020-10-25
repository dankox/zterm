package monitor

import (
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
func textToView(w Widgeter, outstr string) {
	if w != nil {
		gui.UpdateAsync(func(g *gocui.Gui) error {
			w.CleanPrint(outstr)
			return nil
		})
	}
}

// Append text to the View. This will preserve previously added content
func appendTextToView(w Widgeter, outstr string) {
	if w != nil {
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
			case out, ok := <-conn.outchan:
				output += out + "\n"
				if !ok {
					if len(output) > 0 {
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
				if len(output) > 0 {
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
