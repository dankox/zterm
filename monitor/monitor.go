package monitor

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"sort"
	"time"

	"github.com/awesome-gocui/gocui"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
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
	// default config with empty View map (so we don't have to do make)
	config = Config{
		Server{},
		map[string]int{},
	}

	// widget/view parameters
	viewOrder   []string
	viewMaxSize = 0
	widgets     []Widgeter
	gui         *gocui.Gui

	// TUI coloring
	// gFrameHighlight = gocui.ColorYellow
	gFrameHighlight = gocui.ColorDefault
	gFrameOk        = gocui.ColorCyan
	gFrameError     = gocui.ColorRed
	gFrameColor     = gocui.ColorDefault

	// ssh parameters
	sshConfig *ssh.ClientConfig
	sshConn   *ssh.Client
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

	// fmt.Println("starting ssh", config.Server.Host)
	// setup ssh configuration
	sshConfig = setupSSHConfig()
	if sshConfig != nil {
		// fmt.Printf("config ssh: %v", sshConfig)
		hostport := fmt.Sprintf("%s:%d", config.Server.Host, 22)
		// fmt.Printf("ssh host: %v", hostport)
		conn, err := ssh.Dial("tcp", hostport, sshConfig)
		if err != nil {
			fmt.Printf("cannot connect %v: %v\n", hostport, err)
		} else {
			sshConn = conn
			// fmt.Println("connected to ssh as", config.Server.User)
			defer sshConn.Close()
		}
	}

	// count 100% size of all the views
	for k, v := range config.Views {
		viewOrder = append(viewOrder, k)
		viewMaxSize += v
	}
	sort.Strings(viewOrder)

	// setup UI
	g, err := gocui.NewGui(gocui.OutputNormal, true)
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
func setupManagers() []Widgeter {
	managers := []Widgeter{}

	// add help widget first
	managers = append(managers, NewHelpWidget())

	// add configured views
	for i, v := range viewOrder {
		widget := NewWidgetStack(v, i, config.Views[v], fmt.Sprintf("Loading %v...", v))
		if widget.GetName() == "1-joblog" {
			widget.Fun = cmdTestShell
			widget.StartFun()
		} else if widget.GetName() == "2-syslog" {
			widget.Fun = cmdSyslogShell
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

func getWidgetManager(name string) Widgeter {
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

// setupSSHConfig loads a public key and setup ssh config for connection
func setupSSHConfig() *ssh.ClientConfig {
	usr, err := user.Current()
	if err != nil {
		return nil
	}
	keyfile := usr.HomeDir + "/.ssh/id_rsa"
	fmt.Printf("read file %v\n", keyfile)

	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	authkey := ssh.PublicKeys(signer)
	config := &ssh.ClientConfig{
		User:            config.Server.User,
		Auth:            []ssh.AuthMethod{authkey},
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return config
}
