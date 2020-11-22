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
	"path/filepath"
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
	fmt.Print("Do you want to add it? [y/n]: ")

	a, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	return strings.ToLower(strings.TrimSpace(a)) == "yes" || strings.ToLower(strings.TrimSpace(a)) == "y"
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

// sshCopyTo copies local file to remote path
//
// remote path can be absolute or relative path, or dataset name (starting with //)
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

	// if dataset pattern
	if isDsn(remotePath) {
		err = session.Run("cat > ~/.zterm/" + dsnNormalize(remotePath) + " && cp ~/.zterm/" + dsnNormalize(remotePath) + " " + dsnNormalize(remotePath))
		if err != nil {
			// return err
			return fmt.Errorf("dd/cp: %v", err)
		}
		return nil
	}

	// for regular files
	return session.Run("cat > " + remotePath)
}

// sshCopyFrom copies remote path to local file
//
// remote path can be absolute or relative path, or dataset name (starting with //)
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

	// if dataset pattern
	if isDsn(remotePath) {
		err = session.Run("cp " + dsnNormalize(remotePath) + " ~/.zterm/" + dsnPathBase(remotePath) + " && cat ~/.zterm/" + dsnPathBase(remotePath))
		if err != nil {
			// return err
			return fmt.Errorf("cp/dd: '%v' -> %v", "cp "+dsnNormalize(remotePath)+" ~/.zterm/"+dsnPathBase(remotePath)+" && cat ~/.zterm/"+dsnPathBase(remotePath), err)
		}
		return nil
	}

	// for regular files
	return session.Run("cat " + remotePath)
}

// isDsn check if string is valid dataset name or not.
// Valid name starts with // and can be included in double quotes, like: "//dsn.name", //dsn.name or //'dsn.name'
func isDsn(str string) bool {
	str = strings.Trim(str, "\"")
	if strings.HasPrefix(str, "//") {
		return true
	}
	return false
}

// dsnNormalize normalize dataset name to uss version - "//'dsn.name'"
//
// Dataset names are converted to the uss version. If the name is like dsn.name or //'dsn.name' or even full uss version.
func dsnNormalize(dsn string) string {
	dsn = strings.Trim(dsn, "\"")
	if strings.HasPrefix(dsn, "//") {
		return "\"" + dsn + "\""
	}
	return "\"//'" + dsn + "'\""
}

// dsnPathBase returns base name of dataset or path.
//
// - dataset name, it is simplified to last qualifier or member name
//
// - path (not dataset name), base name is returned from the path
func dsnPathBase(dsn string) string {
	if !isDsn(dsn) {
		return filepath.Base(dsn)
	}
	dsn = strings.Trim(strings.TrimLeft(strings.Trim(dsn, "\""), "//"), "'")
	dsnParts := strings.Split(dsn, ".")
	lastPart := strings.Split(dsnParts[len(dsnParts)-1], "(")
	if len(lastPart) > 1 {
		return strings.TrimRight(lastPart[1], ")")
	}
	return lastPart[0]
}
