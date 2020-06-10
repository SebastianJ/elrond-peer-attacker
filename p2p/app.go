package p2p

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	epa_libp2p "github.com/SebastianJ/elrond-peer-attacker/p2p/elrond/libp2p"
	sdkTransactions "github.com/SebastianJ/elrond-sdk/transactions"
)

func StartNodes() error {
	fmt.Println("Starting new node....")

	for i := 0; i <= Configuration.P2P.Peers; i++ {
		go StartNode()
	}

	return nil
}

func StartNode() error {
	messenger, err := createNode(*Configuration.P2P.ElrondConfig)
	if err != nil {
		return err
	}

	err = messenger.Bootstrap()
	if err != nil {
		return err
	}

	time.Sleep(time.Second * time.Duration(Configuration.P2P.ConnectionWait))

	subscribeToTopics(messenger)
	displayMessengerInfo(messenger)

	fmt.Printf("Sleeping %d seconds before proceeding to start sending messages\n", Configuration.P2P.ConnectionWait)

	nonce := Configuration.Account.Nonce

	for {
		for i := 0; i < 100000; i++ {
			broadcastMessage(messenger, nonce)
			nonce++
		}

		/*select {
		case <-time.After(time.Second * 10):
			subscribeToTopics(messenger)
		case <-time.After(time.Second * 35):
			displayMessengerInfo(messenger)
		default:
			broadcastMessage(messenger, nonce)
			nonce++
		}*/
	}
}

func subscribeToTopics(messenger p2p.Messenger) {
	for _, topic := range Configuration.P2P.Topics {
		messenger.CreateTopic(topic, true)
	}
}

func broadcastMessage(messenger p2p.Messenger, nonce uint64) { //, waitGroup *sync.WaitGroup) {
	//defer waitGroup.Done()
	bytes, err := generateTransaction(nonce)

	/*bytes := randomizeData()
	var err error = nil*/

	if err == nil {
		for _, topic := range Configuration.P2P.Topics {
			fmt.Printf("Sending message of %d bytes to topic/channel %s\n", len(bytes), topic)

			go messenger.BroadcastOnChannel(
				topic,
				topic,
				bytes,
			)
		}
	}
}

func randomizeData() []byte {
	randomNumber := rand.New(rand.NewSource(time.Now().UTC().UnixNano())).Intn(1000000)
	var message strings.Builder
	message.WriteString(Configuration.P2P.Data)
	message.WriteString(strconv.Itoa(randomNumber))
	bytes := []byte(message.String())

	return bytes
}

func generateTransaction(nonce uint64) ([]byte, error) {
	gasParams := Configuration.Account.GasParams

	tx, _, err := sdkTransactions.GenerateTransaction(
		Configuration.Account.Wallet,
		"erd1hlccprf7e89gfzq0r7z2gfypjr9dm7ya3sw4m36r0gaxllm507vsxqy9zk",
		0.0,
		false,
		int64(nonce),
		Configuration.P2P.Data,
		gasParams,
		Configuration.Account.Client,
	)

	signature, err := sdkTransactions.SignTransaction(Configuration.Account.Wallet, tx)
	if err != nil {
		return nil, err
	}

	tx.Signature = signature
	marshalizer := &marshal.GogoProtoMarshalizer{}

	txBuff, err := marshalizer.Marshal(tx)
	if err != nil {
		return nil, err
	}

	return txBuff, err
}

func generateAddress() ([]byte, error) {
	return hex.DecodeString("000000000000000000005fed9c659422cd8429ce92f8973bba2a9fb51e0eb3a1")
}

func createNode(p2pConfig config.P2PConfig) (p2p.Messenger, error) {
	arg := epa_libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.ListenAddrWithIp4AndTcp,
		P2pConfig:     p2pConfig,
	}

	return epa_libp2p.NewNetworkMessenger(arg)
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

	for _, topic := range Configuration.P2P.Topics {
		peers := messenger.ConnectedPeersOnTopic(topic)
		fmt.Printf("Connected peers on topic %s: %d\n\n", topic, len(peers))
	}
}
