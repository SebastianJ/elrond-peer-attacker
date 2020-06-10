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
	"github.com/SebastianJ/elrond-sdk/api"
	"github.com/SebastianJ/elrond-sdk/transactions"
	sdkWallet "github.com/SebastianJ/elrond-sdk/wallet"
	"github.com/urfave/cli"
)

var (
	// count defines the number of seed nodes to launch
	count = cli.IntFlag{
		Name:  "count",
		Usage: "How many nodes to launch",
		Value: 100,
	}

	ipAddressFile = cli.StringFlag{
		Name:  "ip-address-file",
		Usage: "Which file to use for reading ip addresses to launch seed nodes on",
		Value: "./data/ips.txt",
	}

	receiversFile = cli.StringFlag{
		Name:  "receivers",
		Usage: "Which file to use for reading receiver addresses to send txs to",
		Value: "./data/receivers.txt",
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

	walletPath = cli.StringFlag{
		Name:  "wallet",
		Usage: "Wallet PEM file to use for sending transactions",
		Value: "./keys/initialBalancesSk.pem",
	}

	economicsConfigurationPath = cli.StringFlag{
		Name:  "economics-config",
		Usage: "Path to economics.toml config file",
		Value: "./config/economics.toml",
	}

	p2pConfigurationPath  = "./config/p2p.toml"
	econConfigurationPath = "./config/economics.toml"
	txData                = ""
)

func main() {
	app := cli.NewApp()
	app.Name = "Eclipser CLI App"
	app.Usage = "This is the entry point for starting a new eclipser app - the app will launch a bunch of seed nodes that essentially don't do anything"
	app.Flags = []cli.Flag{count, ipAddressFile, receiversFile, configurationPath, economicsConfigurationPath, dataPath, walletPath}
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

	if err := setupP2PConfig(ctx); err != nil {
		return err
	}

	if err := setupAccountConfig(ctx); err != nil {
		return err
	}

	p2p.Configuration.Concurrency = 10000

	if err := p2p.StartNodes(); err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("terminating at user's signal...")

	return nil
}

func setupP2PConfig(ctx *cli.Context) error {
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

		p2p.Configuration.P2P.Data = fileData
	}

	p2p.Configuration.P2P.ElrondConfig = p2pConfig
	p2p.Configuration.P2P.Peers = ctx.GlobalInt(count.Name)
	p2p.Configuration.P2P.Topics = p2p.Topics
	p2p.Configuration.P2P.ConnectionWait = 60
	p2p.Configuration.P2P.MessageCount = 100000
	p2p.Configuration.P2P.Shards = []string{
		"META",
		"0",
		"1",
	}

	addressPath, err := filepath.Abs(ctx.GlobalString(receiversFile.Name))
	if err != nil {
		return err
	}

	receivers, err := utils.ArrayFromFile(addressPath)
	if err != nil {
		return err
	}

	if len(receivers) == 0 {
		receivers = []string{"erd1mp543xj384uzehwzp360wy2y86q22svdm022lwxaryg8cqmxqwvszjnrf7"}
	}

	p2p.Configuration.P2P.TxReceivers = receivers

	return nil
}

func setupAccountConfig(ctx *cli.Context) error {
	path := ctx.GlobalString(walletPath.Name)

	wallet, err := sdkWallet.Decrypt(path)
	if err != nil {
		return err
	}

	client := api.Client{
		Host:                 "https://wallet-api.elrond.com",
		ForceAPINonceLookups: true,
	}
	client.Initialize()

	if ctx.IsSet(economicsConfigurationPath.Name) {
		econConfigurationPath = ctx.GlobalString(economicsConfigurationPath.Name)
	}

	defaultGasParams, err := transactions.ParseGasSettings(econConfigurationPath)
	if err != nil {
		return err
	}

	account, err := client.GetAccount(wallet.Address)
	if err != nil {
		return err
	}
	nonce := uint64(account.Nonce)

	p2p.Configuration.Account = p2p.AccountConfig{
		Wallet:    wallet,
		Nonce:     nonce,
		GasParams: defaultGasParams,
		Client:    client,
	}

	return nil
}
