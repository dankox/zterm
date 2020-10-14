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
func cmdSyslog(ctx context.Context) (*RecvConn, error) {
	result := NewRecvConn()

	// handle bash command execution
	if (time.Now().Second() % 30) < 10 {
		return nil, errors.New("WTF??? Eroooooooooooooooorrr... ")
	}

	// c := exec.CommandContext(ctx, "sh", "-c", "ls -l ~ && date")
	c := exec.CommandContext(ctx, "sh", "-c", "./test.sh")
	outPipe, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	c.Stderr = c.Stdout // combine stdout and stderr
	if err := c.Start(); err != nil {
		return nil, err
	}

	// setup moderator
	go func() {
		<-result.sigEnd
		close(result.signal)
	}()

	// setup output processing
	go func() {
		defer close(result.outchan)

		scan := bufio.NewScanner(outPipe)
		for scan.Scan() {
			select {
			case <-result.signal:
				return
			case result.outchan <- scan.Text():
			}
		}
	}()

	// setup wait function
	go func() {
		defer close(result.err)

		if err := c.Wait(); err != nil {
			select {
			case <-result.signal:
				return
			case result.err <- err:
			}
		}
		result.Stop() // try to send sigEnd
	}()

	return result, nil
}
