package p2p

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	epa_libp2p "github.com/SebastianJ/elrond-peer-attacker/p2p/elrond/libp2p"
)

func StartNodes() error {
	fmt.Println("Starting new node....")

	for i := 0; i <= Configuration.Peers; i++ {
		go StartNode()
	}

	return nil
}

func StartNode() error {
	messenger, err := createNode(*Configuration.ElrondConfig)
	if err != nil {
		return err
	}

	err = messenger.Bootstrap()
	if err != nil {
		return err
	}

	time.Sleep(time.Second * time.Duration(Configuration.ConnectionWait))

	subscribeToTopics(messenger)
	displayMessengerInfo(messenger)

	fmt.Printf("Sleeping %d seconds before proceeding to start sending messages\n", Configuration.ConnectionWait)

	for {
		performWork(messenger)
		/*select {
		case <-time.After(time.Second * 5):
			//go displayMessengerInfo(messenger)
		}*/
	}
}

func subscribeToTopics(messenger p2p.Messenger) {
	for _, topic := range Configuration.Topics {
		messenger.CreateTopic(topic, true)
	}
}

func performWork(messenger p2p.Messenger) {
	subscribeToTopics(messenger)

	var waitGroup sync.WaitGroup

	for i := 0; i <= Configuration.MessageCount; i++ {
		waitGroup.Add(1)
		go broadcastMessage(messenger, &waitGroup)
	}

	waitGroup.Wait()
}

func broadcastMessage(messenger p2p.Messenger, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	bytes, err := generateTransaction()
	/*bytes := []byte(Configuration.Data)
	var err error = nil*/

	if err == nil {
		for _, topic := range Configuration.Topics {
			fmt.Printf("Sending message of %d bytes to topic/channel %s\n", len(bytes), topic)

			messenger.BroadcastOnChannel(
				//node.SendTransactionsPipe,
				topic,
				topic,
				bytes,
			)
		}
	}
}

func generateTransaction() ([]byte, error) {
	hexSender, err := generateAddress()
	if err != nil {
		return nil, err
	}

	hexReceiver, err := generateAddress()
	if err != nil {
		return nil, err
	}

	tx := transaction.Transaction{
		Nonce:    1,
		SndAddr:  hexSender,
		RcvAddr:  hexReceiver,
		Value:    new(big.Int).SetInt64(1000000000),
		Data:     []byte(Configuration.Data),
		GasPrice: 200000000000,
		GasLimit: 50000,
	}

	txBuff, err := json.Marshal(&tx)
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

	for _, topic := range Configuration.Topics {
		peers := messenger.ConnectedPeersOnTopic(topic)
		fmt.Printf("Connected peers on topic %s: %d\n\n", topic, len(peers))
	}
}
