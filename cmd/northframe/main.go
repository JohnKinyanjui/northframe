package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/JohnKinyanjui/northframe/internal/lsp"
)

func main() {
	if err := run(os.Args); err != nil {
		var commandErr commandLineError
		if errors.As(err, &commandErr) {
			os.Exit(commandErr.code)
		}
		fmt.Fprintln(os.Stderr, "northframe:", err)
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
  northframe create DIR   create an app; add --template=docs for a documentation site
  northframe generate     compile views, loaders, and routes/api handlers
  northframe run          compile, serve, and reload during development
  northframe build        produce one deployment binary
  northframe db generate  generate typed database code with sqlc
  northframe db create    create the next reversible Goose migration
  northframe db migrate   apply pending application database migrations
  northframe db rollback  roll back the latest application migration
  northframe db seed      run the application's database seeder
  northframe db status    show application database migration status
  northframe db verify    validate migration files without connecting
  northframe add PACKAGE  add and lock a browser JavaScript dependency
  northframe remove NAME  remove a browser JavaScript dependency
  northframe update       resolve and reinstall locked JavaScript dependencies
  northframe upgrade      safely refresh Northframe-managed generated files
  northframe deploy check compile and validate a production executable
  northframe deploy docker print or write a production Dockerfile
  northframe lsp          run the .north language server over stdio`)
}
