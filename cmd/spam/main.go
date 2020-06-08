package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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
	// p2pSeed defines a flag to be used as a seed when generating P2P credentials. Useful for seed nodes.
	p2pSeed = cli.StringFlag{
		Name:  "p2p-seed",
		Usage: "P2P seed will be used when generating credentials for p2p component. Can be any string.",
		Value: "seed",
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
	app.Flags = []cli.Flag{count, p2pSeed, ipAddressFile, configurationPath, dataPath}
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

	if ctx.IsSet(p2pSeed.Name) {
		p2pConfig.Node.Seed = ctx.GlobalString(p2pSeed.Name)
	}

	if ctx.IsSet(dataPath.Name) {
		fileData, err := utils.ReadFileToString(ctx.GlobalString(dataPath.Name))
		if err != nil {
			return err
		}
		txData = fileData
	}

	nodeCount := ctx.GlobalInt(count.Name)
	addressPath, _ := filepath.Abs(ctx.GlobalString(ipAddressFile.Name))
	addresses, _ := utils.ArrayFromFile(addressPath)

	err = p2p.StartNodes(p2pConfig, nodeCount, addresses)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("terminating at user's signal...")

	return nil
}
