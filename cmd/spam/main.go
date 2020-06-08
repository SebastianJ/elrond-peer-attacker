package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/SebastianJ/elrond-peer-attacker/p2p"
	"github.com/SebastianJ/elrond-peer-attacker/utils"
	"github.com/urfave/cli"
)

var (
	// count defines the number of seed nodes to launch
	count = cli.IntFlag{
		Name:  "count",
		Usage: "How many seed nodes to launch",
		Value: 100,
	}

	ipAddressFile = cli.StringFlag{
		Name:  "ip-address-file",
		Usage: "Which file to use for reading ip addresses to launch seed nodes on",
		Value: "../../data/ips.txt",
	}

	configurationPath = cli.StringFlag{
		Name:  "config",
		Usage: "Path to p2p.toml config file",
		Value: "./config/p2p.toml",
	}

	dataPath = cli.StringFlag{
		Name:  "data",
		Usage: "Path to data file",
		Value: "./data/data.txt",
	}

	p2pConfigurationPath = "./config/p2p.toml"
	txData               = ""
)

func main() {
	app := cli.NewApp()
	app.Name = "Eclipser CLI App"
	app.Usage = "This is the entry point for starting a new eclipser app - the app will launch a bunch of seed nodes that essentially don't do anything"
	app.Flags = []cli.Flag{count, ipAddressFile, configurationPath, dataPath}
	app.Version = "v0.0.1"
	app.Authors = []cli.Author{
		{
			Name:  "Sebastian Johnsson",
			Email: "",
		},
	}

	app.Action = func(c *cli.Context) error {
		return startApp(c)
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func startApp(ctx *cli.Context) error {
	fmt.Println("Starting app...")

	if ctx.IsSet(configurationPath.Name) {
		p2pConfigurationPath = ctx.GlobalString(configurationPath.Name)
	}

	p2pConfig, err := core.LoadP2PConfig(p2pConfigurationPath)
	if err != nil {
		return err
	}
	fmt.Printf("Initialized with p2p config from: %s\n", p2pConfigurationPath)

	p2p.Configuration = p2p.Config{}

	if ctx.IsSet(dataPath.Name) {
		fileData, err := utils.ReadFileToString(ctx.GlobalString(dataPath.Name))
		if err != nil {
			return err
		}

		p2p.Configuration.Data = fileData
	}

	p2p.Configuration.ElrondConfig = p2pConfig
	p2p.Configuration.Peers = ctx.GlobalInt(count.Name)
	p2p.Configuration.Topics = p2p.Topics
	p2p.Configuration.ConnectionWait = 30
	p2p.Configuration.MessageCount = 100000
	p2p.Configuration.Shards = []string{
		"META",
		"0",
		"1",
	}

	err = p2p.StartNodes()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("terminating at user's signal...")

	return nil
}
