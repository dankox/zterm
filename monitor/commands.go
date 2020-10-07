package monitor

import (
	"context"
	"errors"
	"os/exec"
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

// command timeout... if running for longer, it will be killed (to not get stuck)
var cmdTimeout = 3 * time.Second

func commandExecute(command string) (string, error) {
	cmdParts := strings.Split(strings.TrimSpace(command), " ")
	switch cmdParts[0] {
	case "exit":
		gui.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})
	case "help":
		return "help: command not implemented yet!", nil
	case "error":
		return "", errors.New("command failed")
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
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		var c *exec.Cmd
		if len(cmdParts) > 1 {
			c = exec.CommandContext(ctx, "code", cmdParts[1])
		} else {
			c = exec.CommandContext(ctx, "code", "--help")
		}
		stdouterr, err := c.CombinedOutput()
		// if err := c.Run(); err != nil {
		if err != nil {
			return string(stdouterr), err
		}
		return string(stdouterr), nil
	default:
		// handle bash command execution
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()
		c := exec.CommandContext(ctx, "sh", "-c", command)
		stdouterr, err := c.CombinedOutput()
		// if err := c.Run(); err != nil {
		if err != nil {
			return string(stdouterr), err
		}
		return string(stdouterr), nil
	}

	return "", nil
}
