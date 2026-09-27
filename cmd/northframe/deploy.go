package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func deploy(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("deploy requires a subcommand: check or docker")
	}
	switch arguments[0] {
	case "check":
		return deployCheck(arguments[1:])
	case "docker":
		return deployDocker(arguments[1:])
	default:
		return fmt.Errorf("unknown deploy subcommand %q; expected check or docker", arguments[0])
	}
}

func deployCheck(arguments []string) error {
	flags := flag.NewFlagSet("deploy check", flag.ContinueOnError)
	target := flags.String("target", ".", "Go main package")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if _, err := generateProject(projectOptions{routes: "web/routes", generated: ".generated/routes", packageName: "routes"}); err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "northframe-deploy-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	output := filepath.Join(directory, "app")
	command := exec.Command("go", "build", "-trimpath", "-o", output, *target)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("deployment build failed: %w", err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() == 0 {
		return errors.New("deployment build did not produce an executable")
	}
	fmt.Printf("deployment check passed (%d byte executable)\n", info.Size())
	return nil
}

func deployDocker(arguments []string) error {
	flags := flag.NewFlagSet("deploy docker", flag.ContinueOnError)
	output := flags.String("output", "-", "Dockerfile output path, or - for stdout")
	target := flags.String("target", ".", "Go main package")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	contents := dockerfile(goDirective("go.mod"), *target)
	if *output == "-" {
		fmt.Print(contents)
		return nil
	}
	if err := atomicWriteFile(*output, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write Dockerfile: %w", err)
	}
	fmt.Println("wrote", *output)
	return nil
}

func goDirective(path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "1.27"
	}
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "go" {
			return fields[1]
		}
	}
	return "1.27"
}

func dockerfile(version, target string) string {
	var output bytes.Buffer
	fmt.Fprintf(&output, "FROM golang:%s-alpine AS build\nWORKDIR /src\nCOPY go.mod go.sum ./\nRUN go mod download\nCOPY . .\nRUN CGO_ENABLED=0 go build -trimpath -ldflags=\"-s -w\" -o /out/app %s\n\n", version, target)
	output.WriteString("FROM alpine:3.22\nRUN addgroup -S north && adduser -S -G north north\nWORKDIR /app\nCOPY --from=build /out/app /app/app\nUSER north\nENV PORT=8000\nEXPOSE 8000\nHEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD wget -q -O /dev/null http://127.0.0.1:8000/api/health || exit 1\nENTRYPOINT [\"/app/app\"]\n")
	return output.String()
}
