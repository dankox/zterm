package monitor

import (
	"fmt"
	"log"
	"sort"

	"github.com/jroimartin/gocui"
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
	GetName() string
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
	cmdView     = "console"
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
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	// prepare widgets
	widgets = setupManagers()
	// convert for GUI library
	managers := make([]gocui.Manager, len(widgets))
	for i, w := range widgets {
		managers[i] = w
	}
	// set layout managers
	g.SetManager(managers...)

	keybinds(g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

// setupManagers prepare list of widgets where each of them manage its own layout and data
func setupManagers() []WidgetManager {
	managers := []WidgetManager{}

	// add help widget first
	managers = append(managers, NewHelpWidget())

	// add configured views
	for i, v := range viewOrder {
		widget := NewWidget(v, i, config.Views[v], fmt.Sprintf("Loading %v...", v))
		managers = append(managers, widget)
	}

	// add floaty widgets
	managers = append(managers, NewWidgetFloaty(cmdView, 0, -4, -1, 3, ">> "))
	return managers
}
