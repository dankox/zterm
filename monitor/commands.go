package monitor

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/jroimartin/gocui"
)

var cmdList = []string{"ls", "pwd", "cd", "whoami", "which", "find", "grep",
	"addview", "exit"}

func commandExecute(command string) (string, string, error) {
	cmdParts := strings.Split(command, " ")
	switch cmdParts[0] {
	case "exit":
		gui.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})
	case "help":
		return "command not implemented yet!", "help", nil
	case "error":
		return "", "", errors.New("command failed")
	case "addview":
		// TODO: too fancy... and probably too much shits... try to fix it in better way
		config.Views[cmdParts[1]] = 10
		viewMaxSize += 10
		viewOrder = append(viewOrder, cmdParts[1])
		// prepare widgets
		widgets = setupManagers()
		// convert for GUI library
		managers := make([]gocui.Manager, len(widgets))
		for i, w := range widgets {
			managers[i] = w
		}
		// set layout managers (deletes everything: keys, views, etc.)
		gui.SetManager(managers...)

		// set keybinds (after layout manager)
		for _, w := range widgets {
			w.Keybinds(gui)
		}
		keybindsGlobal(gui)

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

		if cw := getConsoleWidget(); cw != nil {
			// if v, err := gui.View("1-joblog"); err == nil {
			// 	fmt.Fprint(v, cw.lastView)
			// }
			if v, err := gui.View(cw.lastView); err == nil {
				fmt.Fprint(v, output)
				output = ""
			}
		}

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
