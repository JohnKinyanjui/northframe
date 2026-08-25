package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func buildBinary(target, output string) error {
	command := exec.Command("go", "build", "-o", output, target)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return nil
}

func stopProcess(command *exec.Cmd, wait <-chan error) {
	if command.Process == nil {
		return
	}
	_ = signalProcessGroup(command.Process.Pid, os.Interrupt)
	select {
	case <-wait:
		return
	case <-time.After(3 * time.Second):
		_ = signalProcessGroup(command.Process.Pid, os.Kill)
		<-wait
	}
}

func configureChildProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalProcessGroup(pid int, signal os.Signal) error {
	systemSignal, ok := signal.(syscall.Signal)
	if !ok {
		return fmt.Errorf("unsupported process signal %T", signal)
	}
	return syscall.Kill(-pid, systemSignal)
}

func processWasInterrupted(err error) bool {
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return false
	}
	status, ok := exitError.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return false
	}
	return status.Signal() == syscall.SIGINT || status.Signal() == syscall.SIGTERM
}

func runtimeVersion() string {
	command := exec.Command("go", "version")
	output, err := command.Output()
	if err != nil {
		return "Go unavailable"
	}
	return strings.TrimSpace(string(output))
}
