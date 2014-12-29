package main

import (
	"github.com/codegangsta/cli"
	"os"
)

var Version string

func main() {
	if Version == "" {
		Version = "debug"
	}

	app := cli.NewApp()
	app.Name = "benchmarker"
	app.Version = Version
	app.Usage = "For ISUCON4 FINAL"
	app.Author = ""
	app.Email = ""
	app.Commands = []cli.Command{
		{
			Name:   "remote",
			Usage:  "launch remote benchmark process",
			Flags:  remoteFlags,
			Action: RemoteAction,
		},
		{
			Name:   "bench",
			Usage:  "lanunch standalone benchmark process",
			Flags:  benchFlags,
			Action: BenchAction,
		},
	}
	if MasterAPIKey != "None" {
		app.Commands = append(
			app.Commands,
			cli.Command{
				Name:   "server",
				Usage:  "launch server process",
				Flags:  masterFlags,
				Action: MasterAction,
			},
		)
		app.Commands = append(
			app.Commands,
			cli.Command{
				Name:   "bb",
				Usage:  "launch server process",
				Flags:  bbFlags,
				Action: BBAction,
			},
		)
	}

	app.Run(os.Args)
}
