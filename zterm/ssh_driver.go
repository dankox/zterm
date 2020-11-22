package zterm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/crypto/ssh/terminal"
)

func sshVerifyHost(host string, remote net.Addr, key ssh.PublicKey) error {
	// hostFound: is host in known hosts file.
	// err: error if key not in known hosts file OR host in known hosts file but key changed!
	hostFound, err := goph.CheckKnownHost(host, remote, key, "")

	// Host in known hosts but key mismatch!
	// Maybe because of MAN IN THE MIDDLE ATTACK!
	if hostFound && err != nil {
		return err
	}

	// handshake because public key already exists.
	if hostFound && err == nil {
		return nil
	}

	// Ask user to check if he trust the host public key.
	if askIsHostTrusted(host, key) == false {
		// Make sure to return error on non trusted keys.
		return errors.New("you typed no, aborted")
	}

	// Add the new host to known hosts file.
	err = sshAddKnownHost(host, remote, key)
	if err != nil {
		fmt.Printf("add to knownhost failed: %v\n", err)
	}
	return err
}

func sshNewConnect(host string, port uint, username string) (*goph.Client, error) {
	var err error

	// if agent || goph.HasAgent() {
	// 	auth, err = goph.UseAgent()
	// } else if pass {

	var auth goph.Auth
	var keyfile string
	pass := false

	usr, err := user.Current()
	if err == nil {
		keyfile = usr.HomeDir + "/.ssh/id_rsa"
	}

	if len(keyfile) > 0 {
		auth, err = goph.Key(keyfile, "")
		if err != nil {
			auth = nil // remove auth
			if _, ok := err.(*ssh.PassphraseMissingError); ok {
				auth, err = goph.Key(keyfile, askPass("Enter Private Key Passphrase: "))
				if err != nil {
					return nil, fmt.Errorf("key/passphrase error: %v", err)
				}
			}
		}
	}

	if auth == nil {
		pass = true
		auth = goph.Password(askPass("Enter SSH Password: "))
	}

	config := goph.Config{
		User:     username,
		Addr:     host,
		Port:     port,
		Auth:     auth,
		Callback: sshVerifyHost,
	}

	for i := 0; i < 3; i++ {
		client, err := goph.NewConn(&config)

		// if ok, return client
		if err == nil {
			return client, nil
		}

		// if cannot connect return right away
		if _, ok := err.(*net.OpError); ok {
			return nil, fmt.Errorf("cannot connect to remote server")
		}

		// if key used (not password) return
		if !pass {
			return nil, fmt.Errorf("ssh error: %v", err)
		}

		fmt.Printf("ssh error: %v\n", err)
		// for password, ask for new one
		config.Auth = goph.Password(askPass("Enter SSH Password: "))
	}

	return nil, errors.New("password failure")
}

// sshAddKnownHost add a a host to known hosts file.
func sshAddKnownHost(host string, remote net.Addr, key ssh.PublicKey) (err error) {
	path, err := goph.DefaultKnownHostsPath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(knownhosts.Line([]string{host, knownhosts.Normalize(remote.String())}, key) + "\n")
	return err
}

func askPass(msg string) string {
	fmt.Print(msg)
	pass, err := terminal.ReadPassword(0)
	if err != nil {
		panic(err)
	}

	fmt.Println("")
	return strings.TrimSpace(string(pass))
}

func askIsHostTrusted(host string, key ssh.PublicKey) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Unknown Host: %s \nFingerprint: %s \n", host, ssh.FingerprintSHA256(key))
	fmt.Print("Would you like to add it? type yes or no: ")

	a, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	return strings.ToLower(strings.TrimSpace(a)) == "yes"
}

func sshCopy(r io.Reader, remotePath string, permissions string, size int64) error {
	filename := path.Base(remotePath)
	directory := path.Dir(remotePath)

	session, err := sshConn.NewSession()
	if err != nil {
		return fmt.Errorf("cannot open new session: %v", err)
	}

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintln(w, "C"+permissions, size, filename)
		io.Copy(w, r)
		fmt.Fprintln(w, "\x00")
	}()

	return session.Run("/usr/bin/scp -t " + directory)
}

func sshCopyTo(r io.Reader, remotePath string) error {
	session, err := sshConn.NewSession()
	if err != nil {
		return fmt.Errorf("cannot open new session: %v", err)
	}

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		io.Copy(w, r)
	}()

	return session.Run("dd of=" + remotePath)
}

func sshCopyFrom(w io.WriteCloser, remotePath string) error {
	session, err := sshConn.NewSession()
	if err != nil {
		return fmt.Errorf("cannot open new session: %v", err)
	}

	go func() {
		r, _ := session.StdoutPipe()
		defer w.Close()
		io.Copy(w, r)
	}()

	return session.Run("dd if=" + remotePath)
}

// func getSftp(client *goph.Client) *sftp.Client {
// 	sftpc, err := client.NewSftp()
// 	if err != nil {
// 		panic(err)
// 	}
// 	return sftpc
// }

// func playWithSSHJustForTestingThisProgram(client *goph.Client) {

// 	fmt.Println("Welcome To Goph :D")
// 	fmt.Printf("Connected to %s\n", client.Config.Addr)
// 	fmt.Println("Type your shell command and enter.")
// 	fmt.Println("To download file from remote type: download remote/path local/path")
// 	fmt.Println("To upload file to remote type: upload local/path remote/path")
// 	fmt.Println("To create a remote dir type: mkdirall /path/to/remote/newdir")
// 	fmt.Println("To exit type: exit")

// 	scanner := bufio.NewScanner(os.Stdin)

// 	fmt.Print("> ")

// 	var (
// 		out   []byte
// 		err   error
// 		cmd   string
// 		parts []string
// 	)

// loop:
// 	for scanner.Scan() {

// 		err = nil
// 		cmd = scanner.Text()
// 		parts = strings.Split(cmd, " ")

// 		if len(parts) < 1 {
// 			continue
// 		}

// 		switch parts[0] {

// 		case "exit":
// 			fmt.Println("goph bye!")
// 			break loop

// 		case "download":

// 			if len(parts) != 3 {
// 				fmt.Println("please type valid download command!")
// 				continue loop
// 			}

// 			err = client.Download(parts[1], parts[2])

// 			fmt.Println("download err: ", err)
// 			break

// 		case "upload":

// 			if len(parts) != 3 {
// 				fmt.Println("please type valid upload command!")
// 				continue loop
// 			}

// 			err = client.Upload(parts[1], parts[2])

// 			fmt.Println("upload err: ", err)
// 			break

// 		case "mkdirall":

// 			if len(parts) != 2 {
// 				fmt.Println("please type valid mkdirall command!")
// 				continue loop
// 			}

// 			ftp := getSftp(client)

// 			err = ftp.MkdirAll(parts[1])
// 			fmt.Printf("mkdirall err(%v) you can check via: stat %s\n", err, parts[1])

// 		default:

// 			command, err := client.Command(parts[0], parts[1:]...)
// 			if err != nil {
// 				panic(err)
// 			}
// 			out, err = command.CombinedOutput()
// 			fmt.Println(string(out), err)
// 		}

// 		fmt.Print("> ")
// 	}
// }
