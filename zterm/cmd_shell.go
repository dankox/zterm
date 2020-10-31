package zterm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

// Execute shell command and process output in Widget
func cmdShell(widget Widgeter, command string) error {
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

// Execute vim command and use full terminal
func cmdVim(widget Widgeter, file string) error {
	// handle bash command execution
	c := exec.Command("sh", "-c", "vim "+file)
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	if err := c.Run(); err != nil {
		return err
	}

	return nil
}

// func run(ctx context.Context) error {
func cmdSSH(widget Widgeter, cmd string) error {
	if sshConn == nil {
		return errors.New("SSH connection not created! Adjust your configuration")
	}

	session, err := sshConn.NewSession()
	if err != nil {
		return fmt.Errorf("cannot open new session: %v", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	// start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("session shell: %s", err)
	}

	pipeR, pipeW := io.Pipe()
	session.Stdout = pipeW
	session.Stderr = pipeW

	// send command
	if _, err = fmt.Fprintf(stdin, "%s\n", cmd); err != nil {
		fmt.Println(err)
		return err
	}
	stdin.Close() // just one command

	// prepare communication channel RecvConn
	comch := NewRecvConn()

	// monitor for cancel and close session if done
	go func() {
		<-comch.signal
		pipeW.Close()
		session.Close() // TODO: maybe instead of close, call Signal??
		// session.Signal(ssh.SIGINT) // or maybe ssh.SIGTERM??
	}()

	// read both stdout/stderr in from one reader
	go func() {
		defer close(comch.outchan)

		scan := bufio.NewScanner(pipeR)
		// read output
		for scan.Scan() {
			select {
			case <-comch.signal:
				// killing signal
				return
			case comch.outchan <- scan.Text():
			}
		}
	}()

	// wait for end
	go func() {
		defer close(comch.err)
		defer session.Close()
		defer pipeW.Close() // pipe might not be closed and scanner would wait, therefore close

		// wait end
		if err := session.Wait(); err != nil {
			efmt := fmt.Errorf("ssh: %v", err.Error())
			// convert to ssh error if possible
			if e, ok := err.(*ssh.ExitError); ok && e != nil {
				efmt = fmt.Errorf("ssh: %v", e.ExitStatus())
			}
			// return
			select {
			case <-comch.signal:
				// skip passing error (it's already killed)
			case comch.err <- efmt:
			}
		}
	}()

	connectWidgetOuput(widget, comch)

	return nil
}
