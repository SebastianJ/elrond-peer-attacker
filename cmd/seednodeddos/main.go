package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/libp2p/go-libp2p"
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
	// count defines the number of seed nodes to launch
	count = cli.IntFlag{
		Name:  "count",
		Usage: "How many seed ping nodes to launch",
		Value: 100,
	}

	startPort = cli.IntFlag{
		Name:  "start-port",
		Usage: "Which node to use as a basis for starting the pingers",
		Value: 49152,
	}

	p2pConfigurationFile = "../../config/p2p.toml"
)

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = seedNodeHelpTemplate
	app.Name = "Eclipser CLI App"
	app.Usage = "This is the entry point for starting a new eclipser app - the app will launch a bunch of seed nodes that essentially don't do anything"
	app.Flags = []cli.Flag{count, startPort}
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
	p2pConfig, err := core.LoadP2PConfig(p2pConfigurationFile)
	if err != nil {
		fmt.Println("Failed to parse P2P Config - make sure the config file exists!")
		os.Exit(1)
	}

	seedNodeAddress := p2pConfig.KadDhtPeerDiscovery.InitialPeerList[0]

	fmt.Println(fmt.Sprintf("Using seed node address: %s", seedNodeAddress))

	pingerCount := ctx.GlobalInt(count.Name)
	pingerStartPort := ctx.GlobalInt(startPort.Name)

	for i := 0; i <= pingerCount; i++ {
		actualPort := pingerStartPort + i
		go ddosSeedNode(actualPort, seedNodeAddress)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("terminating at user's signal...")

	return nil
}

func ddosSeedNode(port int, seedNodeAddress string) {
	ctx := context.Background()

	node, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)),
		libp2p.Ping(false),
	)

	if err != nil {
		panic(err)
	}

	pingService := &ping.PingService{Host: node}
	node.SetStreamHandler(ping.ID, pingService.PingHandler)

	peerInfo := &peerstore.PeerInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}

	addrs, _ := peerstore.InfoToP2pAddrs(peerInfo)
	if err == nil {
		fmt.Println("libp2p node address:", addrs[0])

		addr, err := multiaddr.NewMultiaddr(seedNodeAddress)

		if err == nil {
			peer, err := peerstore.InfoFromP2pAddr(addr)

			if err == nil {
				err = node.Connect(ctx, *peer)

				if err == nil {
					ch := pingService.Ping(ctx, peer.ID)

					for {
						fmt.Println("sending ping message to", seedNodeAddress)
						res := <-ch
						fmt.Println("pinged", seedNodeAddress, "in", res.RTT)
					}
				} else {
					fmt.Println("Failed to connnect to ", seedNodeAddress)
				}
			}
		}
	}
}
