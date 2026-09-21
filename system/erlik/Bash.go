package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

type BashShell struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
}

func NewBashShell() (*BashShell) {
	cmd := exec.Command("bash")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil
	}

	shell := &BashShell{
		cmd:   cmd,
		stdin: stdin,
	}

	if err := cmd.Start(); err != nil {
		return nil
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println("OUT:", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println("ERR:", scanner.Text())
		}
	}()

	return shell
}

func (b *BashShell) Run(command string) error {
	_, err := io.WriteString(b.stdin, command+"\n")
	return err
}

func (b *BashShell) Close() error {
	b.stdin.Close()
	return b.cmd.Wait()
}