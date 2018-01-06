package scp

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"os"
	"path"

	shellquote "github.com/kballard/go-shellquote"
)

type Client struct {
	Host         string
	ClientConfig *ssh.ClientConfig
	Session      *ssh.Session
}

// Connects to the remote SSH server, returns error if it couldn't establish a session to the SSH server
func (a *Client) Connect() error {
	client, err := ssh.Dial("tcp", a.Host, a.ClientConfig)
	if err != nil {
		return err
	}

	a.Session, err = client.NewSession()
	if err != nil {
		return err
	}
	return nil
}

//Copies the contents of an os.File to a remote location, it will get the length of the file by looking it up from the filesystem
func (a *Client) CopyFromFile(file os.File, remotePath string, permissions string) error {
	stat, _ := file.Stat()
	return a.Copy(&file, remotePath, permissions, stat.Size())
}

// Copies the contents of an io.Reader to a remote location, the length is determined by reading the io.Reader until EOF
// if the file length in know in advance please use "Copy" instead
func (a *Client) CopyFile(fileReader io.Reader, remotePath string, permissions string) error {
	contents_bytes, _ := ioutil.ReadAll(fileReader)
	bytes_reader := bytes.NewReader(contents_bytes)

	return a.Copy(bytes_reader, remotePath, permissions, int64(len(contents_bytes)))
}

// Copies the contents of an io.Reader to a remote location
func (a *Client) Copy(r io.Reader, remotePath string, permissions string, size int64) error {
	filename := path.Base(remotePath)
	directory := path.Dir(remotePath)

	w, err := a.Session.StdinPipe()

	if err != nil {
		return err
	}

	cmd := shellquote.Join("scp", "-t", directory)

	fmt.Printf("cmd: %s\n", cmd)

	if err := a.Session.Start(cmd); err != nil {
		w.Close()
		return err
	}

	errors := make(chan error)

	go func() {
		errors <- a.Session.Wait()
	}()

	fmt.Fprintln(w, "C"+permissions, size, filename)
	io.Copy(w, r)
	fmt.Fprintln(w, "\x00")
	w.Close()

	return <- errors
}
