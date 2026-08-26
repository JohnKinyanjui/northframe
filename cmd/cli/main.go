package main

import (
	"errors"
	"fmt"
	"os"

	"northframe.dev/northframe/internal/lsp"
)

func main() {
	if err := run(os.Args); err != nil {
		var commandErr commandLineError
		if errors.As(err, &commandErr) {
			os.Exit(commandErr.code)
		}
		fmt.Fprintln(os.Stderr, "north:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) < 2 {
		usage()
		return commandLineError{code: 2}
	}
	switch arguments[1] {
	case "create":
		return createProject(arguments[2:])
	case "generate":
		return generate(arguments[2:])
	case "build":
		return build(arguments[2:])
	case "run", "runserver", "dev":
		return runserver(arguments[2:])
	case "db":
		return database(arguments[2:])
	case "add":
		return addDependency(arguments[2:])
	case "remove":
		return removeDependency(arguments[2:])
	case "update":
		return updateDependencies(arguments[2:])
	case "upgrade":
		return upgrade(arguments[2:])
	case "deploy":
		return deploy(arguments[2:])
	case "lsp":
		return lsp.Run(os.Stdin, os.Stdout)
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return commandLineError{code: 2}
	}
}

type commandLineError struct{ code int }

func (err commandLineError) Error() string { return fmt.Sprintf("invalid command (exit %d)", err.code) }

func usage() {
	fmt.Fprintln(os.Stderr, `Northframe — native Go SSR with structured server views

usage:
  north create DIR   create a Northframe application (use . for this directory)
  north generate     compile views, loaders, and routes/api handlers
  north run          compile, serve, and reload during development
  north build        produce one deployment binary
  north db generate  generate typed database code with sqlc
  north db create    create the next reversible Goose migration
  north db migrate   apply pending application database migrations
  north db rollback  roll back the latest application migration
  north db seed      run the application's database seeder
  north db status    show application database migration status
  north db verify    validate migration files without connecting
  north add PACKAGE  add and lock a browser JavaScript dependency
  north remove NAME  remove a browser JavaScript dependency
  north update       resolve and reinstall locked JavaScript dependencies
  north upgrade      safely refresh Northframe-managed generated files
  north deploy check compile and validate a production executable
  north deploy docker print or write a production Dockerfile
  north lsp          run the .north language server over stdio`)
}
