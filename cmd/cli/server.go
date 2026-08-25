package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

func runserver(arguments []string) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	options := addProjectFlags(flags, true)
	port := flags.Int("port", 8000, "development server port")
	watchRoot := flags.String("watch", ".", "project directory to watch")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *port < 1 || *port > 65535 {
		return fmt.Errorf("invalid port %d", *port)
	}

	temporary, err := os.MkdirTemp("", "north-run-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	binary := filepath.Join(temporary, "app")
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupts)

	for {
		baseline, err := watchSignature(*watchRoot, options.generated)
		if err != nil {
			return err
		}
		if err := compileDevelopmentBuild(*options, binary); err != nil {
			fmt.Fprintln(os.Stderr, "north:", err)
			if waitErr := waitForChange(*watchRoot, options.generated, baseline, interrupts); waitErr != nil {
				if errors.Is(waitErr, errInterrupted) {
					return nil
				}
				return waitErr
			}
			continue
		}
		if err := serveUntilChange(binary, *port, *watchRoot, options.generated, baseline, interrupts); err != nil {
			if errors.Is(err, errInterrupted) {
				return nil
			}
			return err
		}
	}
}

func compileDevelopmentBuild(options projectOptions, binary string) error {
	if _, err := generateProject(options); err != nil {
		return fmt.Errorf("compile failed: %w", err)
	}
	if err := buildBinary(options.target, binary); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	return nil
}

func serveUntilChange(binary string, port int, root, generated, baseline string, interrupts <-chan os.Signal) error {
	command := exec.Command(binary)
	configureChildProcess(command)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	projectEnvironment, err := loadDotEnv(".env", os.Environ())
	if err != nil {
		return err
	}
	command.Env = setEnvironment(projectEnvironment, "PORT", strconv.Itoa(port))
	command.Env = setEnvironment(command.Env, "NORTHFRAME_ENV", "development")
	if err := command.Start(); err != nil {
		return err
	}
	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()
	fmt.Printf("North development server: http://localhost:%d\n", port)
	fmt.Println("Watching project files. Press Ctrl+C to stop.")

	ticker := time.NewTicker(450 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-interrupts:
			stopProcess(command, wait)
			return errInterrupted
		case processErr := <-wait:
			if processErr == nil || processWasInterrupted(processErr) {
				return errInterrupted
			}
			return fmt.Errorf("development server stopped: %w", processErr)
		case <-ticker.C:
			current, err := watchSignature(root, generated)
			if err != nil {
				stopProcess(command, wait)
				return err
			}
			if current != baseline {
				fmt.Println("Change detected; rebuilding…")
				stopProcess(command, wait)
				return nil
			}
		}
	}
}
