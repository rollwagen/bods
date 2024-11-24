package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type BashSession struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	outputBuf  *bufio.Reader
	errorBuf   *bufio.Reader
	started    bool
	timedOut   bool
	timeout    time.Duration
	outputDely time.Duration
	sentinel   string
	doneChan   chan struct{}
	wg         sync.WaitGroup
}

func NewBashSession() *BashSession {
	return &BashSession{
		timeout:    120 * time.Second,
		outputDely: 200 * time.Millisecond,
		sentinel:   "<<exit>>",
		doneChan:   make(chan struct{}),
	}
}

func (bs *BashSession) Start() error {
	if bs.started {
		return nil
	}

	cmd := exec.Command("/bin/bash")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	bs.outputBuf = bufio.NewReader(stdout)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	bs.errorBuf = bufio.NewReader(stderr)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	bs.stdin = stdin // Store the stdin pipe

	if err := cmd.Start(); err != nil {
		return err
	}

	bs.cmd = cmd
	bs.started = true

	bs.wg.Add(1)
	go bs.handleSignals()

	return nil
}

func (bs *BashSession) Stop() error {
	if !bs.started {
		return nil
	}

	if bs.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM and wait for the process to exit
	err := bs.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}
	bs.wg.Wait()

	bs.started = false
	close(bs.doneChan)

	return nil
}

func (bs *BashSession) Run(command string) (string, string, error) {
	if !bs.started {
		return "", "", fmt.Errorf("session has not started")
	}

	if bs.cmd.Process == nil {
		return "", "", fmt.Errorf("bash has exited with returncode %d", bs.cmd.ProcessState.ExitCode())
	}

	if bs.timedOut {
		return "", "", fmt.Errorf("timed out: bash has not returned in %v seconds and must be restarted", bs.timeout.Seconds())
	}

	// Use the stored stdin instead of getting a new pipe
	_, err := io.WriteString(bs.stdin, command+"; echo '"+bs.sentinel+"'\n")
	if err != nil {
		return "", "", err
	}

	output, err := bs.readOutput()
	if err != nil {
		return "", "", err
	}

	errorOutput, _ := bs.readError()

	return output, errorOutput, nil
}

func (bs *BashSession) readOutput() (string, error) {
	var output []byte
	buf := make([]byte, 1024)
	timeout := time.After(bs.timeout)

	for {
		select {
		case <-timeout:
			bs.timedOut = true
			return "", fmt.Errorf("timed out: bash has not returned in %v seconds and must be restarted", bs.timeout.Seconds())
		default:
			n, err := bs.outputBuf.Read(buf)
			if n > 0 {
				output = append(output, buf[:n]...)
				if string(output[len(output)-len(bs.sentinel):]) == bs.sentinel {
					output = output[:len(output)-len(bs.sentinel)]
					return string(output), nil
				}
			}
			if err != nil {
				return "", err
			}
			time.Sleep(bs.outputDely)
		}
	}
}

func (bs *BashSession) readError() (string, error) {
	var errorOutput []byte
	buf := make([]byte, 1024)

	for {
		n, err := bs.errorBuf.Read(buf)
		if n > 0 {
			errorOutput = append(errorOutput, buf[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}

	return string(errorOutput), nil
}

func (bs *BashSession) handleSignals() {
	defer bs.wg.Done()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		_ = bs.Stop()
	case <-bs.doneChan:
	}
}

func mainBash() {
	session := NewBashSession()
	err := session.Start()
	if err != nil {
		fmt.Println("Error starting bash session:", err)
		return
	}
	defer session.Stop()

	output, errorOutput, err := session.Run("echo 'Hello, World!'")
	if err != nil {
		fmt.Println("Error running command:", err)
	} else {
		fmt.Println("Output:", output)
		fmt.Println("Error Output:", errorOutput)
	}
}

//
// - `BashSession` struct represents a session of a bash shell.
// - `NewBashSession` function creates a new `BashSession` instance with default values.
// - `Start` method starts a new bash process and sets up pipes for stdin, stdout, and stderr.
// - `Stop` method terminates the bash process and waits for it to exit.
// - `Run` method executes a command in the bash shell, reads the output and error output, and returns them as strings.
// - `readOutput` and `readError` helper methods are used to read output and error output from the bash process.
// - `handleSignals` goroutine handles SIGINT and SIGTERM signals and stops the bash session.
//
// The `main` function demonstrates the usage of the `BashSession` by creating a new session, running the command `echo 'Hello, World!'`, and printing the output and error output.
//
// Note: This implementation assumes a Unix-like environment and may require adjustments for other operating systems.
