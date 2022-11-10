package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"nysa-network/pkg/notifyer"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

type service struct {
	cfg *Config

	notify *notifyer.Client
}

func (s *service) parseConfig(c *cli.Context) error {
	data, err := os.ReadFile(c.String("config"))
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &s.cfg); err != nil {
		return err
	}
	return nil
}

func main() {
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	s := service{}

	globalFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Value:   "config.yml",
			Usage:   "Application config file `PATH`",
		},
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "start",
				Usage:  "start cosmos-notifyer server",
				Action: s.Start,
				Flags:  globalFlags,
				Before: s.parseConfig,
			},
		},
	}
	app.Flags = globalFlags

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
