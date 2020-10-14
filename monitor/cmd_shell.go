package monitor

import (
	"bufio"
	"context"
	"os/exec"
)

// Execute shell command and process output in Widget
func cmdShell(widget WidgetManager, command string) error {
	// setup widget context
	ctx := widget.WithContext(context.Background())

	// handle bash command execution
	c := exec.CommandContext(ctx, "sh", "-c", command)
	outPipe, err := c.StdoutPipe()
	if err != nil {
		widget.CancelCtx() // stop context
		return err
	}
	c.Stderr = c.Stdout // combine stdout and stderr
	if err := c.Start(); err != nil {
		widget.CancelCtx() // stop context
		return err
	}

	// prepare communication channel RecvConn
	comch := NewRecvConn()

	// setup moderator
	go func() {
		<-comch.sigEnd
		close(comch.signal)
	}()

	// setup output processing
	go func() {
		defer close(comch.outchan)

		scan := bufio.NewScanner(outPipe)
		for scan.Scan() {
			select {
			case <-comch.signal:
				return
			case comch.outchan <- scan.Text():
			}
		}
	}()

	// setup wait function
	go func() {
		defer widget.CancelCtx()
		defer close(comch.err)

		if err := c.Wait(); err != nil {
			select {
			case <-comch.signal:
				// moderator is already stopped (he is the only one closing this)
				return
			case comch.err <- err:
			}
		}
		comch.Stop() // try to send sigEnd
	}()

	connectWidgetOuput(widget, comch)

	return nil
}
