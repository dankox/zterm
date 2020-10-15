package monitor

import (
	"bufio"
	"context"
	"os/exec"
)

// Execute shell command and process output in Widget
func cmdShell(widget WidgetManager, command string) error {
	// setup widget context
	ctx, cancel := context.WithCancel(context.Background())

	// handle bash command execution
	c := exec.CommandContext(ctx, "sh", "-c", command)
	outPipe, err := c.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}
	c.Stderr = c.Stdout // combine stdout and stderr
	if err := c.Start(); err != nil {
		cancel()
		return err
	}

	// prepare communication channel RecvConn
	comch := NewRecvConn()

	// setup moderator
	// go func() {
	// 	<-comch.sigEnd
	// 	close(comch.signal)
	// 	widget.CancelCtx() // ??
	// }()

	// setup output processing
	go func() {
		defer cancel()
		defer close(comch.err)
		defer close(comch.outchan)

		scan := bufio.NewScanner(outPipe)
		// read output
		for scan.Scan() {
			select {
			case <-comch.signal:
				// killing signal
				return
			case comch.outchan <- scan.Text():
			}
		}
		// wait end
		if err := c.Wait(); err != nil {
			select {
			case <-comch.signal:
				// moderator is already stopped (he is the only one closing this)
				return
			case comch.err <- err:
			}
		}
	}()

	connectWidgetOuput(widget, comch)

	return nil
}
