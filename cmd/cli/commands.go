package main

import (
	"flag"
	"fmt"
)

func generate(arguments []string) error {
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	options := addProjectFlags(flags, false)
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	count, err := generateProject(*options)
	if err != nil {
		return err
	}
	fmt.Printf("compiled %d route(s) into %s\n", count, options.generated)
	return nil
}

func build(arguments []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	options := addProjectFlags(flags, true)
	binary := flags.String("o", "app", "output binary")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if _, err := generateProject(*options); err != nil {
		return err
	}
	if err := buildBinary(options.target, *binary); err != nil {
		return err
	}
	fmt.Println("built", *binary)
	return nil
}
