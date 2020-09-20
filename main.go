package main

import (
	"fmt"
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

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, updateLayout); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

// Layout function is always called at the end of MainLoop(), so be careful to not update layout outside of it
//  - consume events (like keypress, mouse, resize) and user events (from gocui.Update())
//  - flush () -> calls layout function
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if _, err := g.View("zMonitor"); err == gocui.ErrUnknownView {
		if v, err := g.SetView("zMonitor", maxX/2-7, maxY/2, maxX/2+7, maxY/2+2); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			fmt.Fprintln(v, "Hello world!")
		}
	}
	return nil
}

func updateLayout(g *gocui.Gui, v *gocui.View) error {

	// gocui.Update() can be called from goroutine to update content
	go g.Update(func(g *gocui.Gui) error {
		name := "zMonitor"
		maxX, maxY := g.Size()
		delta := rand.Intn(10)
		nv, err := g.SetView(name, maxX/2-(7+delta), maxY/2-delta, maxX/2+7+delta, maxY/2+2+delta)
		if err != nil {
			fmt.Printf("zMonitor error: %v", err)
			return err
		}
		nv.Wrap = true
		g.SetCurrentView(name)
		g.SetViewOnTop(name)
		nv.Clear()
		fmt.Fprintf(nv, "Hello random z/OS world!\nhost = %v\nuser = %v", config.Server.Host, config.Server.User)
		return nil
	})

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
