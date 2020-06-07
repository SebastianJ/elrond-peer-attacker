package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	"github.com/SebastianJ/elrond-libp2p-attacker/utils"
	"github.com/urfave/cli"
)

var (
	seedNodeHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
VERSION:
   {{.Version}}
   {{end}}
`
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

	p2pConfigurationPath = "./config/p2p.toml"

	errNilSeed                     = errors.New("nil seed")
	errEmotySeed                   = errors.New("empty seed")
	errNilBuffer                   = errors.New("nil buffer")
	errEmptyBuffer                 = errors.New("empty buffer")
	errInvalidPort                 = errors.New("cannot start node on port < 0")
	errPeerDiscoveryShouldBeKadDht = errors.New("kad-dht peer discovery should have been enabled")
)

type seedRandReader struct {
	index int
	seed  []byte
}

// NewSeedRandReader will return a new instance of a seed-based reader
func NewSeedRandReader(seed []byte) *seedRandReader {
	return &seedRandReader{seed: seed, index: 0}
}

// Read to provided buffer pseudo-random generated bytes
func (srr *seedRandReader) Read(p []byte) (n int, err error) {
	if srr.seed == nil {
		return 0, errNilSeed
	}
	if len(srr.seed) == 0 {
		return 0, errEmotySeed
	}
	if p == nil {
		return 0, errNilBuffer
	}
	if len(p) == 0 {
		return 0, errEmptyBuffer
	}
	for i := 0; i < len(p); i++ {
		p[i] = srr.seed[srr.index]
		srr.index = (srr.index + 1) % len(srr.seed)
	}

	return len(p), nil
}

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = seedNodeHelpTemplate
	app.Name = "Eclipser CLI App"
	app.Usage = "This is the entry point for starting a new eclipser app - the app will launch a bunch of seed nodes that essentially don't do anything"
	app.Flags = []cli.Flag{count, p2pSeed, ipAddressFile, configurationPath}
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

	nodeCount := ctx.GlobalInt(count.Name)
	addressPath, _ := filepath.Abs(ctx.GlobalString(ipAddressFile.Name))
	addresses, _ := utils.ArrayFromFile(addressPath)

	err = startSeedNodes(p2pConfig, nodeCount, addresses)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("terminating at user's signal...")

	return nil
}

func startSeedNodes(p2pConfig *config.P2PConfig, nodeCount int, addresses []string) error {
	fmt.Println("Starting new seed node....")

	var address string

	if len(addresses) > 0 {
		address = utils.RandomElementFromArray(addresses)
	} else {
		address = ""
	}

	for i := 0; i <= nodeCount; i++ {
		go startSeedNode(p2pConfig, address)
	}

	return nil
}

func startSeedNode(p2pConfig *config.P2PConfig, address string) error {
	messenger, err := createNode(*p2pConfig)
	if err != nil {
		return err
	}

	err = messenger.Bootstrap()
	if err != nil {
		return err
	}

	displayMessengerInfo(messenger)
	for {
		select {
		case <-time.After(time.Second * 5):
			displayMessengerInfo(messenger)
		}
	}
}

func createNode(p2pConfig config.P2PConfig) (p2p.Messenger, error) {
	arg := libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.ListenAddrWithIp4AndTcp,
		P2pConfig:     p2pConfig,
	}

	return libp2p.NewNetworkMessenger(arg)
}

func displayMessengerInfo(messenger p2p.Messenger) {
	headerSeedAddresses := []string{"Seednode addresses:"}
	addresses := make([]*display.LineData, 0)

	for _, address := range messenger.Addresses() {
		addresses = append(addresses, display.NewLineData(false, []string{address}))
	}

	tbl, _ := display.CreateTableString(headerSeedAddresses, addresses)
	fmt.Println(tbl)

	mesConnectedAddrs := messenger.ConnectedAddresses()
	sort.Slice(mesConnectedAddrs, func(i, j int) bool {
		return strings.Compare(mesConnectedAddrs[i], mesConnectedAddrs[j]) < 0
	})

	headerConnectedAddresses := []string{
		fmt.Sprintf("Seednode is connected to %d peers:", len(mesConnectedAddrs))}
	connAddresses := make([]*display.LineData, len(mesConnectedAddrs))

	for idx, address := range mesConnectedAddrs {
		connAddresses[idx] = display.NewLineData(false, []string{address})
	}

	tbl2, _ := display.CreateTableString(headerConnectedAddresses, connAddresses)
	fmt.Println(tbl2)
}
