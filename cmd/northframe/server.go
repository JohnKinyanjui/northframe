package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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
	// Fail before claiming the port: a missing application will not fix itself by watching.
	if err := validateRoutesDirectory(options.routes); err != nil {
		return err
	}

	temporary, err := os.MkdirTemp("", "northframe-run-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupts)

	supervisor, err := newDevelopmentSupervisor(*port)
	if err != nil {
		return err
	}
	defer supervisor.Close()

	var current *developmentChild
	defer func() {
		if current != nil {
			current.Stop()
		}
	}()
	buildNumber := 0
	for {
		baseline, err := watchSignature(*watchRoot, options.generated)
		if err != nil {
			return err
		}
		buildNumber++
		binary := filepath.Join(temporary, fmt.Sprintf("app-%d", buildNumber))
		if err := compileDevelopmentBuild(*options, binary); err != nil {
			fmt.Fprintln(os.Stderr, "northframe:", err)
			if waitErr := waitForChange(*watchRoot, options.generated, baseline, interrupts); waitErr != nil {
				if errors.Is(waitErr, errInterrupted) {
					return nil
				}
				return waitErr
			}
			continue
		}

		next, err := startDevelopmentChild(binary)
		if err != nil {
			return err
		}
		if err := next.WaitReady(); err != nil {
			next.Stop()
			return err
		}
		if err := supervisor.Switch(next.URL()); err != nil {
			next.Stop()
			return err
		}
		previous := current
		current = next
		if previous == nil {
			if err := supervisor.Start(); err != nil {
				next.Stop()
				current = nil
				return err
			}
			fmt.Printf("Northframe development server: http://localhost:%d\n", *port)
			fmt.Println("Watching project files. Press Ctrl+C to stop.")
		} else {
			supervisor.ReloadBrowsers()
			previous.Stop()
			fmt.Println("Updated without restarting the public development server.")
		}

		changed, err := waitForDevelopmentChange(*watchRoot, options.generated, baseline, current, interrupts)
		if err != nil {
			if errors.Is(err, errInterrupted) {
				return nil
			}
			return err
		}
		if changed {
			fmt.Println("Change detected; rebuilding…")
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
