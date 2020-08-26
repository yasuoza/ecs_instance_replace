package main

import (
	"log"
	"os"

	"github.com/mitchellh/cli"
	instance_replace "github.com/yasuoza/ecs_instance_replace"
	"github.com/yasuoza/ecs_instance_replace/cmd/ecs_instance_replace/command"
)

var app *instance_replace.App
var Version = "current"

func main() {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stdout,
	}

	c := cli.NewCLI("ecs_instance_replace", Version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"replace": func() (cli.Command, error) {
			return &command.ReplaceCommand{UI: ui}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
