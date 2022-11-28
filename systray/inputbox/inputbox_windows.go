package inputbox

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

type Local struct{}

func (b *Local) StartProcess(cmd string, args ...string) (Waiter, io.Writer, io.Reader, io.Reader, error) {
	command := exec.Command(cmd, args...)
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW

	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	err = command.Start()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return command, stdin, stdout, stderr, nil
}

const newline = "\r\n"

type Shell interface {
	Execute(cmd string) (string, string, error)
	Exit()
}

type shell struct {
	handle Waiter
	stdin  io.Writer
	stdout io.Reader
	stderr io.Reader
}

type Waiter interface {
	Wait() error
}

type Starter interface {
	StartProcess(cmd string, args ...string) (Waiter, io.Writer, io.Reader, io.Reader, error)
}

func psNew(backend Starter) (Shell, error) {
	handle, stdin, stdout, stderr, err := backend.StartProcess("powershell.exe", "-NoExit", "-Command", "-")
	if err != nil {
		return nil, err
	}

	return &shell{handle, stdin, stdout, stderr}, nil
}

func (s *shell) Execute(cmd string) (string, string, error) {
	if s.handle == nil {
		return "", "", errors.New("nil handle")
	}

	outBoundary := createBoundary()
	errBoundary := createBoundary()

	// wrap the command in special markers so we know when to stop reading from the pipes
	full := fmt.Sprintf("%s; echo '%s'; [Console]::Error.WriteLine('%s')%s", cmd, outBoundary, errBoundary, newline)

	_, err := s.stdin.Write([]byte(full))
	if err != nil {
		return "", "", err
	}

	// read stdout and stderr
	sout := ""
	serr := ""

	waiter := &sync.WaitGroup{}
	waiter.Add(2)

	go streamReader(s.stdout, outBoundary, &sout, waiter)
	go streamReader(s.stderr, errBoundary, &serr, waiter)

	waiter.Wait()

	if len(serr) > 0 {
		return sout, serr, errors.New(serr)
	}

	return sout, serr, nil
}

func (s *shell) Exit() {
	s.stdin.Write([]byte("exit" + newline))

	// if it's possible to close stdin, do so (some backends, like the local one,
	// do support it)
	closer, ok := s.stdin.(io.Closer)
	if ok {
		closer.Close()
	}

	s.handle.Wait()

	s.handle = nil
	s.stdin = nil
	s.stdout = nil
	s.stderr = nil
}

func streamReader(stream io.Reader, boundary string, buffer *string, signal *sync.WaitGroup) error {
	// read all output until we have found our boundary token
	output := ""
	bufsize := 64
	marker := boundary + newline

	for {
		buf := make([]byte, bufsize)
		read, err := stream.Read(buf)
		if err != nil {
			return err
		}

		output = output + string(buf[:read])

		if strings.HasSuffix(output, marker) {
			break
		}
	}

	*buffer = strings.TrimSuffix(output, marker)
	signal.Done()

	return nil
}

func CreateRandomString(bytes int) string {
	c := bytes
	b := make([]byte, c)

	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}
func createBoundary() string {
	return "$gorilla" + CreateRandomString(12) + "$"
}

// InputBox displays a dialog box, returning the entered value and a bool for success
func InputBox(title, message, defaultAnswer string) (string, bool) {
	shell, err := psNew(&Local{})
	if err != nil {
		panic(err)
	}
	defer shell.Exit()

	out, _, err := shell.Execute(`
		[void][Reflection.Assembly]::LoadWithPartialName('Microsoft.VisualBasic')
		$title = '` + title + `'
		$msg = '` + message + `'
		$default = '` + defaultAnswer + `'
		$answer = [Microsoft.VisualBasic.Interaction]::InputBox($msg, $title, $default)
		Write-Output $answer
		`)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
