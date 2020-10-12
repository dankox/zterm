package monitor

import (
	"fmt"
	"log"
	"sort"

	"github.com/awesome-gocui/gocui"
	"github.com/spf13/viper"
)

// Server configuration
type Server struct {
	Host string
	User string
}

// Config type defining configuration
type Config struct {
	Server Server
	Views  map[string]int
}

// WidgetManager cover Layout for GUI and some specifics for widgets
type WidgetManager interface {
	// Layout is for gocui.GUI
	Layout(*gocui.Gui) error
	Keybinds(*gocui.Gui)
	GetName() string
	GetView() *gocui.View
	IsHidden() bool
}

var (
	// default config with empty View map (so we don't have to do make)
	config = Config{
		Server{},
		map[string]int{},
	}

	viewOrder   []string
	viewMaxSize = 0
	widgets     []WidgetManager
	gui         *gocui.Gui

	// gFrameHighlight = gocui.ColorYellow
	gFrameHighlight = gocui.ColorDefault
	gFrameOk        = gocui.ColorCyan
	gFrameError     = gocui.ColorRed
	gFrameColor     = gocui.ColorDefault
)

// Main function of monitor package
//
// - setup GUI for TUI (terminal user interface)
//
// - set layout
//
// - set keybindings
//
// - run GUI.MainLoop
func Main() {
	// load config file (or arguments)
	viper.Unmarshal(&config)

	// count 100% size of all the views
	for k, v := range config.Views {
		viewOrder = append(viewOrder, k)
		viewMaxSize += v
	}
	sort.Strings(viewOrder)

	// setup UI
	g, err := gocui.NewGui(gocui.OutputNormal, false)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	gui = g // save pointer for use outside

	// prepare widgets
	widgets = setupManagers()
	// set layout manager function
	g.SetManagerFunc(handleLayouts)
	// prepare default highlight colors
	g.SelFrameColor = gFrameHighlight
	g.SelFgColor = gFrameHighlight

	// set keybinds (after layout manager)
	for _, w := range widgets {
		w.Keybinds(g)
	}
	keybindsGlobal(g)

	// main loop running
	if err := g.MainLoop(); err != nil && !gocui.IsQuit(err) {
		g.Cursor = true
		log.Panicln(err)
	}
	g.Cursor = true
}

// setupManagers prepare list of widgets where each of them manage its own layout and data
func setupManagers() []WidgetManager {
	managers := []WidgetManager{}

	// add help widget first
	managers = append(managers, NewHelpWidget())

	// add configured views
	for i, v := range viewOrder {
		widget := NewWidget(v, i, config.Views[v], fmt.Sprintf("Loading %v...", v))
		if widget.GetName() == "2-syslog" {
			widget.Fun = cmdSyslog
			widget.StartFun()
		}
		managers = append(managers, widget)
	}

	// add floaty widgets
	// managers = append(managers, NewWidgetFloaty("test-window", 0, -4, -1, 3, "Window"))

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

func getWidgetManager(name string) WidgetManager {
	for _, w := range widgets {
		if w.GetName() == name {
			return w
		}
	}
	return nil
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

func getWidget(name string) *Widget {
	for _, w := range widgets {
		if w.GetName() == name {
			if ww, ok := w.(*Widget); ok {
				return ww
			}
			return nil
		}
	}
	return nil
}
