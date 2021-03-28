package zterm

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/awesome-gocui/gocui"
	"github.com/melbahja/goph"
	"github.com/spf13/viper"
)

// Server configuration
type Server struct {
	Host string
	User string
}

// View configuration
type View struct {
	Position int      `mapstructure:"position"`
	Size     int      `mapstructure:"size"`
	Job      string   `mapstructure:"job,omitempty"`
	HiLine   []string `mapstructure:"hiline,omitempty"`
	HiWord   []string `mapstructure:"hiword,omitempty"`
}

// Config type defining configuration
type Config struct {
	Server `mapstructure:"server"`
	Theme  `mapstructure:"theme"`
	Views  map[string]View `mapstructure:"views"`
}

var (
	// default config with empty View map (so we don't have to do make)
	config = Config{
		Server{},
		Theme{},
		map[string]View{},
	}

	// widget/view parameters
	viewMaxSize  = 0
	viewLastPos  = 0
	viewFirstPos = -1
	widgets      []Widgeter
	gui          *gocui.Gui

	// ssh connection
	sshConn *goph.Client

	// ErrSuspend error cause gocui environment to suspend
	ErrSuspend = errors.New("suspend")

	suspendChan chan struct{}
	resumeChan  chan struct{}
)

// Main function of zterm package
//
// - setup GUI for TUI (terminal user interface)
//
// - set layout
//
// - set keybindings
//
// - run GUI.MainLoop
func Main(remote bool) {
	var err error

	// load config file (or arguments)
	viper.Unmarshal(&config)

	// load theme from config
	LoadTheme()

	if remote {
		// setup ssh configuration
		sshConn, err = sshNewConnect(config.Server.Host, 22, config.Server.User)
	}

	// For Windows 7 or other non-compatible stuff
	// This is how in `tcell/v2` we can bypass scrolling problem
	if config.ColorSpace == "basic" {
		os.Setenv("TCELL_TRUECOLOR", "disable")
	} else if config.ColorSpace == "truecolor" {
		os.Setenv("COLORTERM", "truecolor")
	}
	// setup UI
	g, err := gocui.NewGui(gocui.OutputTrue, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	gui = g // save pointer for use outside
	g.FgColor = cFgColor
	g.BgColor = cBgColor

	// prepare widgets
	widgets = setupManagers()
	g.SetManagerFunc(handleLayouts)

	hasWidgets := false
	// set keybinds (after layout manager)
	for _, w := range widgets {
		w.Keybinds(g)
		if w.GetName() != cmdView {
			hasWidgets = true
		}
	}
	keybindsGlobal(g)

	// if no widget stack, show help
	if !hasWidgets {
		// this needs to be run at the end, because it handles all keybinds and layouts and stuff
		PopupHelpWidget()
	}

	// prepare suspend channel
	suspendChan = make(chan struct{})

	// main loop running
	for {
		err = g.MainLoop()
		// check what type of exit we've got from mainloop
		if err != nil && err.Error() == ErrSuspend.Error() {
			// suspend request
			g.Cursor = true
			g.Close()

			// prepare resume channel
			resumeChan = make(chan struct{})
			close(suspendChan)

			// wait for resume
			<-resumeChan
			suspendChan = make(chan struct{}) // recreate suspend channel
			// recreate gocui
			g, err = gocui.NewGui(gocui.OutputTrue, true)
			if err != nil {
				log.Panicln(err)
			}
			gui = g
			g.SetManagerFunc(handleLayouts)
			// set keybinds (after layout manager)
			for _, w := range widgets {
				w.Keybinds(g)
				if w.GetName() != cmdView {
					hasWidgets = true
				}
			}
			keybindsGlobal(g)

		} else if errors.Is(err, gocui.ErrQuit) {
			// quit request
			g.Cursor = true
			break
		} else if err != nil {
			log.Panicln(err)
		}
	}
}

// setupManagers prepare list of widgets where each of them manage its own layout and data
func setupManagers() []Widgeter {
	managers := []Widgeter{}

	// add configured views
	for vname, v := range config.Views {
		viewMaxSize += v.Size // setup view maximum size
		// setup first view position (if nothing set before)
		if viewFirstPos < 0 {
			viewFirstPos = v.Position
		}
		widget := NewWidgetStack(vname, v.Position, v.Size, fmt.Sprintf("Loading %v...\n", vname))
		// check if last position
		if viewLastPos < v.Position {
			viewLastPos = v.Position
		}
		// check if first position
		if viewFirstPos > v.Position {
			viewFirstPos = v.Position
		}
		// setup job for view ;)
		widget.SetupFun(v.Job)
		// setup highlight
		widget.highlight = make(map[string]bool)
		for _, hi := range v.HiWord {
			widget.highlight[hi] = false
		}
		for _, hi := range v.HiLine {
			widget.highlight[hi] = true
		}

		// add to manager list
		managers = append(managers, widget)
	}
	sortWidgetManager(managers)

	// add console widget
	managers = append(managers, NewWidgetConsole())
	return managers
}

// Handle layouts of all the widgets (called by managerFunc)
func handleLayouts(g *gocui.Gui) error {
	for _, w := range widgets {
		if err := w.Layout(g); err != nil {
			return err
		}
	}
	// handle cursor visibility (for editable only)
	if v := g.CurrentView(); v != nil {
		if v.Editable {
			g.Cursor = true
		} else {
			g.Cursor = false
		}
	}
	return nil
}

func getWidgetManager(name string) Widgeter {
	for _, w := range widgets {
		if w.GetName() == name {
			return w
		}
	}
	return nil
}

// Sort WidgetStacks in Widgeter list
func sortWidgetManager(managers []Widgeter) {
	sort.Slice(managers, func(i, j int) bool {
		return managers[i].Position() < managers[j].Position()
	})
}

func getWidgetView(name string) *gocui.View {
	for _, w := range widgets {
		if w.GetName() == name {
			return w.GetView()
		}
	}
	return nil
}

func getConsoleWidget() *WidgetConsole {
	for _, w := range widgets {
		if w.GetName() == cmdView {
			if wc, ok := w.(*WidgetConsole); ok {
				return wc
			}
			return nil
		}
	}
	return nil
}

func getWidgetStack(name string) *WidgetStack {
	for _, w := range widgets {
		if w.GetName() == name {
			if ws, ok := w.(*WidgetStack); ok {
				return ws
			}
			return nil
		}
	}
	return nil
}

func getSortedWidgetStack() (wlist []*WidgetStack) {
	for _, w := range widgets {
		if ws, ok := w.(*WidgetStack); ok {
			wlist = append(wlist, ws)
		}
	}
	sort.Slice(wlist, func(i, j int) bool {
		return wlist[i].pos < wlist[j].pos
	})
	return
}
