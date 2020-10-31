package zterm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
	"github.com/spf13/viper"
)

// autocomplete map (command with subcommands/operands)
var cmdAuto = map[string][]string{
	"addview": {"joblog", "syslog", "messages"},
	"attach":  {"joblog", "syslog", "messages"},
	"code":    {},
	"error":   {},
	"exit":    {},
	"help":    {},
	"remote":  {},
	"resize":  {"joblog", "syslog", "messages"},
	"view":    {"joblog", "syslog", "messages"},
	"savecfg": {},

	"pwd":    {},
	"whoami": {},
	"which":  {},
	"cd":     {},
	"ls":     {"#list-dir"},
}

func commandExecute(wgm Widgeter, command string) error {
	cmdParts := strings.Split(strings.TrimSpace(command), " ")

	switch cmdParts[0] {
	case "exit":
		gui.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})
	case "help":
		return errors.New("help: command not implemented yet")
	case "error":
		return errors.New("command failed")
	case "addview":
		if len(cmdParts) == 1 {
			return errors.New("adview: requires view name to add it")
		}

		vname := cmdParts[1]
		vmap, ok := config.Views[vname]
		if ok {
			return fmt.Errorf("view '%s' already exist", vname)
		}
		vmap = View{}
		vmap.Size = 10
		vmap.Position = viewLastPos + 1
		viewLastPos = vmap.Position
		if viewFirstPos < 0 {
			viewFirstPos = vmap.Position
		}
		config.Views[vname] = vmap
		viewMaxSize += vmap.Size
		widget := NewWidgetStack(vname, vmap.Position, vmap.Size, "new view")
		widgets = append(widgets, widget)
		sortWidgetManager(widgets)
		widget.Keybinds(gui)
		// run layouts to sort the order (console on top)
		widget.Layout(gui)
		getConsoleWidget().Layout(gui)
		return fmt.Errorf("view '%s' added", vname)
	case "resize":
		if len(cmdParts) == 1 {
			return errors.New("resize: requires view name to resize it")
		}

		vname := cmdParts[1]
		widget := getWidgetStack(vname)
		if widget == nil {
			return fmt.Errorf("resize: view '%s' doesn't exist", vname)
		}
		newsize := 1
		if len(cmdParts) > 2 {
			if is, err := strconv.Atoi(cmdParts[2]); err == nil {
				newsize = is
			}
		}

		// resize and adjust maxsize
		widget.height += newsize
		vmap := config.Views[vname]
		viewMaxSize += widget.height - vmap.Size
		vmap.Size = widget.height
		config.Views[vname] = vmap // is this necessary ???
		return fmt.Errorf("view '%s' resized", vname)
	case "view":
		if len(cmdParts) < 3 {
			return errors.New(`missing arguments 
usage: view <view-name> <config>

config options: 
 hi-word   <word>    - highlight word
 hi-line   <word>    - highlight line which contains word
 hi-remove <word>    - remove highlight for specific word
 refresh   <number>  - set refresh interval to number`)
		}

		vname := cmdParts[1]
		vconf := cmdParts[2]
		widget := getWidgetStack(vname)
		if widget == nil {
			return fmt.Errorf("view: view '%s' doesn't exist", vname)
		}
		switch vconf {
		case "hi-word":
			fallthrough
		case "hi-line":
			if len(cmdParts) < 4 {
				return fmt.Errorf("view: view %s needs a <word> parameter", vconf)
			}
			if widget.highlight == nil {
				widget.highlight = make(map[string]bool)
			}
			if vconf == "hi-word" {
				widget.highlight[cmdParts[3]] = false
			} else {
				widget.highlight[cmdParts[3]] = true
			}
		case "hi-remove":
			if len(cmdParts) < 4 {
				return fmt.Errorf("view: view %s needs a <word> parameter", vconf)
			}
			if widget.highlight != nil {
				delete(widget.highlight, cmdParts[3])
			}
		default:
			return fmt.Errorf("view: config option %s not implemented", vconf)
		}
		return fmt.Errorf("view %s configured", vname)
	case "attach":
		if len(cmdParts) < 3 {
			return errors.New("missing arguments - usage: attach <view-name> <command>")
		}

		vname := cmdParts[1]
		widget := getWidgetStack(vname)
		if widget == nil {
			return fmt.Errorf("attach: view '%s' doesn't exist", vname)
		}
		widget.StopFun()
		widget.SetupFun(strings.Join(cmdParts[2:], " "))
		widget.StartFun()
		return fmt.Errorf("command attached to view '%s'", vname)
	case "savecfg":
		// update view configuration in viper
		for k, v := range config.Views {
			if ws := getWidgetStack(k); ws != nil {
				// highlights
				v.HiLine = []string{}
				v.HiWord = []string{}
				for hi, isline := range ws.highlight {
					if isline {
						v.HiLine = append(v.HiLine, hi)
					} else {
						v.HiWord = append(v.HiWord, hi)
					}
				}
				// job
				v.Job = ws.GetFunString()
			}
			viper.Set("views."+k, v)
		}
		cfgfile := viper.ConfigFileUsed()
		data, _ := ioutil.ReadFile(cfgfile) // save for error
		if err := viper.WriteConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				if err := viper.WriteConfigAs(".zterm.yml"); err != nil {
					return fmt.Errorf("%v", err)
				}
			} else {
				return fmt.Errorf("%v\noriginal config file: \n%v", err, string(data))
			}
		}
		return fmt.Errorf("config file %v updated", cfgfile)
	case "code":
		// handle vscode command execution
		if len(cmdParts) > 1 {
			return cmdShell(wgm, command)
		}
		return cmdShell(wgm, "code --help")
	case "vim":
		// handle vim command execution
		if len(cmdParts) > 1 {
			return cmdVim(wgm, strings.Join(cmdParts[1:], " "))
		}
		return cmdShell(wgm, "vim --help")
	case "remote":
		if len(cmdParts) > 1 {
			return cmdSSH(wgm, strings.Join(cmdParts[1:], " "))
		}
		return errors.New("remote: requires command to run on remote server")
	default:
		// handle bash command execution
		return cmdShell(wgm, command)
	}

	return nil
}

// simple function for testing widgets
func cmdSyslogShell(widget Widgeter) error {
	// handle bash command execution
	return cmdSSH(widget, "zsyslog")
}

// simple function for testing widgets
func cmdTestShell(widget Widgeter) error {
	// fake error
	if (time.Now().Second() % 30) < 10 {
		return errors.New("WTF??? Eroooooooooooooooorrr... ")
	}

	// handle bash command execution
	return cmdShell(widget, "./test.sh")
}
