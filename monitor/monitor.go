package monitor

import (
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

var (
	config      = Config{}
	viewOrder   []string
	viewMaxSize = 0
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

	g.SetManagerFunc(layout)

	keybinds(g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
