package cmd

import (
	"fmt"
	"github.com/pm-connect/log-shipper/source/nomad"
)

// RunCommand contains the config and methods for the Run command.
type NomadSourceCommand struct {
	NomadAddr string `help:"The nomad server address."`
}

// NewRunCommand created a new instance of the RunCommand ready to use.
func NewNomadSourceCommand() *NomadSourceCommand {
	return &NomadSourceCommand{}
}

// Run starts the command.
func (c *NomadSourceCommand) Run() error {
	source := nomad.Source{}

	details, err := source.Start()

	if err != nil {
		panic(err)
	}

	fmt.Printf("Running: %s:%d\n", details.Host, details.Port)

	select {}

	return nil
}
