package monitor

import (
	"errors"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
)

// autocomplete map (command with subcommands/operands)
var cmdAuto = map[string][]string{
	"addview": {"test-1", "test-2"},
	"code":    {},
	"error":   {},
	"exit":    {},
	"help":    {},

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
		config.Views[cmdParts[1]] = 10
		viewMaxSize += 10
		viewOrder = append(viewOrder, cmdParts[1])
		// prepare widgets
		widget := NewWidget(cmdParts[1], len(viewOrder)-1, 10, "new view")
		widgets = append(widgets, widget)
		widget.Keybinds(gui)
		// run layouts to sort the order (console on top)
		widget.Layout(gui)
		getConsoleWidget().Layout(gui)
	case "code":
		// handle vscode command execution
		if len(cmdParts) > 1 {
			return cmdShell(wgm, command)
		}
		return cmdShell(wgm, "code --help")
	default:
		// handle bash command execution
		return cmdShell(wgm, command)
	}

	return nil
}

// simple function for testing widgets
func cmdSyslogShell(widget WidgetManager) error {
	// fake error
	if (time.Now().Second() % 30) < 10 {
		return errors.New("WTF??? Eroooooooooooooooorrr... ")
	}

	// handle bash command execution
	return cmdShell(widget, "./test.sh")
}
