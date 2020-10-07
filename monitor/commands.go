package monitor

import (
	"errors"
	"io/ioutil"
	"os/exec"
	"strings"

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

func commandExecute(command string) (string, string, error) {
	cmdParts := strings.Split(strings.TrimSpace(command), " ")
	switch cmdParts[0] {
	case "exit":
		gui.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})
	case "help":
		return "", "help: command not implemented yet!", nil
	case "error":
		return "", "", errors.New("command failed")
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
		var c *exec.Cmd
		if len(cmdParts) > 1 {
			c = exec.Command("code", cmdParts[1])
		} else {
			c = exec.Command("code", "--help")
		}
		stderr, err := c.StderrPipe()
		stdout, err := c.StdoutPipe()
		if err != nil {
			return "", "", err
		}
		if err := c.Start(); err != nil {
			return "", "", err
		}

		slurpErr, _ := ioutil.ReadAll(stderr)
		slurpOut, _ := ioutil.ReadAll(stdout)
		output := string(slurpOut)

		if err := c.Wait(); err != nil {
			return "", "", err
		}

		if len(slurpErr) > 0 {
			return "", "", errors.New(string(slurpErr))
		}
		return output, "", nil
	default:
		// handle bash command execution
		c := exec.Command("sh", "-c", command)
		stderr, err := c.StderrPipe()
		stdout, err := c.StdoutPipe()
		if err != nil {
			return "", "", err
		}
		if err := c.Start(); err != nil {
			return "", "", err
		}
		slurpErr, _ := ioutil.ReadAll(stderr)
		slurpOut, _ := ioutil.ReadAll(stdout)
		output := string(slurpOut)

		if err := c.Wait(); err != nil {
			return "", "", err
		}

		if len(slurpErr) > 0 {
			return "", "", errors.New(string(slurpErr))
		}
		return output, "", nil
	}

	return "", "", nil
}
