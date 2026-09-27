package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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

func runtimeVersion() string {
	command := exec.Command("go", "version")
	output, err := command.Output()
	if err != nil {
		return "Go unavailable"
	}
	return strings.TrimSpace(string(output))
}
