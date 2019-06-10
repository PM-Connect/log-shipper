package main

import (
	"fmt"
	"github.com/jaffee/commandeer/cobrafy"
	"github.com/pm-connect/log-shipper/cmd"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	log.SetLevel(log.DebugLevel)

	args := os.Args[1:]

	var err error

	if len(args) < 1 {
		fmt.Print("No command specified.")
		return
	}

	switch args[0] {
	case "run":
		err = cobrafy.Execute(cmd.NewRunCommand())
	case "source":
		switch args[1] {
		case "nomad":
			err = cobrafy.Execute(cmd.NewNomadSourceCommand())
		}
	}

	if err != nil {
		panic(err)
	}
}
