package monitor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
)

// autocomplete map (command with subcommands/operands)
var cmdAuto = map[string][]string{
	"addview": {"test-1", "test-2"},
	"attach":  {"1-joblog", "2-syslog", "3-cmd"},
	"code":    {},
	"error":   {},
	"exit":    {},
	"help":    {},
	"remote":  {},
	"resize":  {"1-joblog", "2-syslog", "3-cmd"},

	"pwd":    {},
	"whoami": {},
	"which":  {},
	"cd":     {},
	"ls":     {"#list-dir"},
}

func commandExecute(wgm WidgetManager, command string) error {
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
		config.Views[vname] = 10
		viewMaxSize += 10
		viewOrder = append(viewOrder, vname)
		// prepare widgets
		widget := NewWidget(vname, len(viewOrder)-1, 10, "new view")
		widgets = append(widgets, widget)
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
		widget := getWidget(vname)
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
		viewMaxSize += widget.height - config.Views[vname]
		config.Views[vname] = widget.height
		return fmt.Errorf("view '%s' resized", vname)
	case "attach":
		if len(cmdParts) < 3 {
			return errors.New("missing arguments - usage: attach <view-name> <command>")
		}

		vname := cmdParts[1]
		widget := getWidget(vname)
		if widget == nil {
			return fmt.Errorf("attach: view '%s' doesn't exist", vname)
		}
		widget.StopFun()
		if cmdParts[2] == "remote" && len(cmdParts) > 3 {
			widget.Fun = func(w WidgetManager) error {
				return cmdSSH(w, strings.Join(cmdParts[3:], " "))
			}
		} else {
			widget.Fun = func(w WidgetManager) error {
				return cmdShell(w, strings.Join(cmdParts[2:], " "))
			}
		}
		widget.StartFun()
		return fmt.Errorf("command attached to view '%s'", vname)
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
func cmdSyslogShell(widget WidgetManager) error {
	// handle bash command execution
	return cmdSSH(widget, "zsyslog")
}

// simple function for testing widgets
func cmdTestShell(widget WidgetManager) error {
	// fake error
	if (time.Now().Second() % 30) < 10 {
		return errors.New("WTF??? Eroooooooooooooooorrr... ")
	}

	// handle bash command execution
	return cmdShell(widget, "./test.sh")
}
