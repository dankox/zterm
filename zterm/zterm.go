package zterm

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
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
	sshConn *ssh.Client
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
	// load config file (or arguments)
	viper.Unmarshal(&config)

	// load theme from config
	LoadTheme()

	if remote {
		// setup ssh configuration
		sshConn = initSSHConnection()
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

// initSSHConnection loads a public key and setup ssh config for connection
func initSSHConnection() *ssh.Client {
	usr, err := user.Current()
	if err != nil {
		return nil
	}
	keyfile := usr.HomeDir + "/.ssh/id_rsa"

	var signer ssh.Signer
	key, err := ioutil.ReadFile(keyfile)
	if err == nil {
		signer, err = ssh.ParsePrivateKey(key)
	}

	tries := 2
	for {
		conn, err := sshConnect(signer)
		if err == nil {
			return conn
		} else if signer != nil || tries < 1 {
			fmt.Println(err)
			return nil
		}
		if _, ok := err.(*net.OpError); ok {
			fmt.Println("cannot connect to remote server")
			return nil
		}
		fmt.Println(err)
		tries--
	}
}

// sshConnect tries to connect to ssh server with provided authkey/signer,
// or request password if signer is nil.
func sshConnect(signer ssh.Signer) (*ssh.Client, error) {
	auth := []ssh.AuthMethod{}
	if signer == nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Password: ")
		pass, _, _ := reader.ReadLine()
		auth = []ssh.AuthMethod{ssh.Password(string(pass))}
	} else {
		auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}
	sshConfig := &ssh.ClientConfig{
		User:            config.Server.User,
		Auth:            auth,
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	hostport := fmt.Sprintf("%s:%d", config.Server.Host, 22)
	conn, err := ssh.Dial("tcp", hostport, sshConfig)
	return conn, err
}
