package monitor

import (
	"bufio"
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

func commandExecute(ctx context.Context, command string) (*wRecvConn, error) {
	cmdParts := strings.Split(strings.TrimSpace(command), " ")
	// prepare result wRecvConn
	result := &wRecvConn{
		outchan: make(chan string, 10),
		err:     make(chan error, 1),
		signal:  make(chan struct{}),
	}

	switch cmdParts[0] {
	case "exit":
		gui.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})
	case "help":
		return nil, errors.New("help: command not implemented yet")
	case "error":
		return nil, errors.New("command failed")
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
		var c *exec.Cmd
		if len(cmdParts) > 1 {
			c = exec.CommandContext(ctx, "code", cmdParts[1])
		} else {
			c = exec.CommandContext(ctx, "code", "--help")
		}
		stdouterr, err := c.CombinedOutput()
		// maybe this into goroutine??? just to not block accidentally???
		result.outchan <- string(stdouterr)
		close(result.outchan)
		if err != nil {
			return result, err
		}
		return result, nil
	default:
		// handle bash command execution
		c := exec.CommandContext(ctx, "sh", "-c", command)
		outPipe, err := c.StdoutPipe()
		if err != nil {
			return nil, err
		}
		c.Stderr = c.Stdout // combine stdout and stderr
		if err := c.Start(); err != nil {
			return nil, err
		}

		// setup output processing
		go func() {
			scan := bufio.NewScanner(outPipe)
			for scan.Scan() {
				select {
				case <-result.signal:
					return
				case result.outchan <- scan.Text():
				}
			}
			close(result.outchan)
		}()

		// setup wait function
		go func() {
			if err := c.Wait(); err != nil {
				select {
				case <-result.signal:
					return
				case result.err <- err:
				}
			}
			close(result.err)
		}()

		return result, nil
	}

	return nil, nil
}

// simple function for testing widgets
func cmdSyslog(ctx context.Context) (string, error) {
	// handle bash command execution
	if (time.Now().Second() % 30) < 10 {
		return "", errors.New("WTF??? Eroooooooooooooooorrr... ")
	}
	c := exec.CommandContext(ctx, "sh", "-c", "ls -l ~ && date")
	stdouterr, err := c.CombinedOutput()
	if err != nil {
		return string(stdouterr), err
	}
	return string(stdouterr), nil
}
