package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/pelletier/go-toml"
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

func main() {

	// example how the content looks like (need double quotes for string)
	// [server]
	// host = "localhost"
	// user = "username"
	doc, err := ioutil.ReadFile(".zmonitor")
	if err == nil {
		toml.Unmarshal(doc, &config)
	}

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
