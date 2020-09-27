package monitor

import (
	"log"
	"math/rand"
	"time"

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
}

var config = Config{}

// go always executes init() at the startup, after all variable declarations
func init() {
	rand.Seed(time.Now().UnixNano())
}

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
