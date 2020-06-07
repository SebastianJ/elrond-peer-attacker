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
	"github.com/ElrondNetwork/elrond-go/node"
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

	dataPath = cli.StringFlag{
		Name:  "data",
		Usage: "Path to data file",
		Value: "./data/data.txt",
	}

	p2pConfigurationPath = "./config/p2p.toml"

	shards = []string{
		"META",
		"0",
		"1",
	}

	topics = []string{
		"transactions_META",
		"transactions_0_META",
		"transactions_0",
		"transactions_0_1",
		"transactions_1",
		"transactions_1_META",
		"unsignedTransactions_META",
		"unsignedTransactions_0_META",
		"unsignedTransactions_0",
		"unsignedTransactions_0_1",
		"unsignedTransactions_1",
		"unsignedTransactions_1_META",
		"rewardsTransactions_0_META",
		"rewardsTransactions_1_META",
		"shardBlocks_0_META",
		"shardBlocks_1_META",
		"txBlockBodies_ALL",
		"validatorTrieNodes_META",
		"accountTrieNodes_META",
		"accountTrieNodes_0_META",
		"accountTrieNodes_1_META",
		"consensus_0",
		"consensus_1",
		"consensus_meta",
		"heartbeat",
	}

	txData = ""

	waitTime = 30

	errNilSeed                     = errors.New("nil seed")
	errEmotySeed                   = errors.New("empty seed")
	errNilBuffer                   = errors.New("nil buffer")
	errEmptyBuffer                 = errors.New("empty buffer")
	errInvalidPort                 = errors.New("cannot start node on port < 0")
	errPeerDiscoveryShouldBeKadDht = errors.New("kad-dht peer discovery should have been enabled")
)

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = seedNodeHelpTemplate
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
		fmt.Printf("FileData: %s\n\n", fileData)
		txData = fileData
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

	for _, topic := range topics {
		messenger.CreateTopic(topic, true)
	}

	go displayMessengerInfo(messenger)

	fmt.Printf("Sleeping %d seconds before proceeding to start sending messages\n", waitTime)
	time.Sleep(time.Duration(waitTime))

	for {
		go broadcastMessage(messenger)

		select {
		case <-time.After(time.Second * 5):
			go displayMessengerInfo(messenger)
		}
	}
}

func broadcastMessage(messenger p2p.Messenger) {
	for _, topic := range topics {
		fmt.Printf("Sending message of %d bytes to topic/channel %s\n", len(txData), topic)
		go messenger.BroadcastOnChannelBlocking(
			node.SendTransactionsPipe,
			topic,
			[]byte(txData),
		)
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

	headerConnectedAddresses := []string{fmt.Sprintf("Node is connected to %d peers:", len(mesConnectedAddrs))}
	connAddresses := make([]*display.LineData, len(mesConnectedAddrs))

	for idx, address := range mesConnectedAddrs {
		connAddresses[idx] = display.NewLineData(false, []string{address})
	}

	tbl2, _ := display.CreateTableString(headerConnectedAddresses, connAddresses)
	fmt.Println(tbl2)

	for _, topic := range topics {
		peers := messenger.ConnectedPeersOnTopic(topic)
		fmt.Printf("Connected peers on topic %s: %d\n\n", topic, len(peers))
	}
}

func generateTopics() []string {
	var topics []string
	baseTopics := []string{
		"transactions",
		"unsignedTransactions",
		"rewardsTransactions",
		"shardBlocks",
		"txBlockBodies",
		"peerChangeBlockBodies",
		"metachainBlocks",
		"accountTrieNodes",
		"validatorTrieNodes",
	}

	for _, baseTopic := range baseTopics {
		if baseTopic == "txBlockBodies" {
			topics = append(topics, fmt.Sprintf("%s_ALL", baseTopic))
		} else {
			for _, shard := range shards {
				shard = strings.ToUpper(shard)
				topics = append(topics, fmt.Sprintf("%s_%s", baseTopic, shard))

				for _, innerShard := range shards {
					innerShard = strings.ToUpper(innerShard)
					if innerShard != shard {
						topics = append(topics, fmt.Sprintf("%s_%s_%s", baseTopic, shard, innerShard))
					}
				}
			}
		}
	}

	return topics
}
