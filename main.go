package main

import (
	"github.com/jaffee/commandeer/cobrafy"
	"github.com/pm-connect/log-shipper/cmd"
)

func main() {
	err := cobrafy.Execute(cmd.NewRunCommand())

	if err != nil {
		panic(err)
	}
}
