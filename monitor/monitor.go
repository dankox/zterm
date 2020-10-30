package monitor

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"sort"
	"strconv"
	"time"

	"github.com/awesome-gocui/gocui"
	"github.com/muesli/termenv"
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
	Server Server          `mapstructure:"server"`
	Theme  map[string]int  `mapstructure:"theme"`
	Views  map[string]View `mapstructure:"views"`
}

var (
	// default config with empty View map (so we don't have to do make)
	config = Config{
		Server{},
		map[string]int{},
		map[string]View{},
	}

	// widget/view parameters
	viewMaxSize  = 0
	viewLastPos  = 0
	viewFirstPos = -1
	widgets      []Widgeter
	gui          *gocui.Gui

	// TUI coloring
	// gFrameHighlight = gocui.ColorYellow
	gFrameHighlight = gocui.ColorDefault
	gFrameOk        = gocui.ColorCyan
	gFrameError     = gocui.ColorRed
	gFrameColor     = gocui.ColorDefault
	cConsole        = gocui.ColorCyan
	cConsoleStr     = strconv.Itoa(int(cConsole) - 1)
	cError          = gocui.ColorRed
	cErrorStr       = strconv.Itoa(int(cError) - 1)
	cHighlight      = gocui.ColorMagenta
	cHighlightStr   = strconv.Itoa(int(cHighlight) - 1)

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

	// setup UI
	g, err := gocui.NewGui(gocui.Output256, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	gui = g // save pointer for use outside

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

func colorText(text string, color string) string {
	// outputStr := "\033[38;5;"
	p := termenv.ColorProfile()
	return termenv.String(text).Foreground(p.Color(color)).String()
	// outputStr := "\033[3"
	// outbuf.Write(strconv.AppendUint(intbuf, uint64(a-1), 10))
	// attr := strings.Split(color, ",")
	// bold := false
	// col := attr[0]
	// if attr[0] == "bold" {
	// 	bold = true
	// } else if (len(attr) > 1 && attr[1] == "bold") {
	// 	bold = true
	// }
	// switch color {
	// 	case ""
	// }
	// outputStr += "m"
	// return outputStr
}
