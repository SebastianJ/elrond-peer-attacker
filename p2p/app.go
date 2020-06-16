package p2p

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	epa_libp2p "github.com/SebastianJ/elrond-peer-attacker/p2p/elrond/libp2p"
	"github.com/SebastianJ/elrond-peer-attacker/utils"
	sdkTransactions "github.com/SebastianJ/elrond-sdk/transactions"
	sdkWallet "github.com/SebastianJ/elrond-sdk/wallet"
)

var (
	transactionTopic     = "transactions"
	sendTransactionsPipe = "send transactions pipe"
)

func StartNodes() error {
	fmt.Println("Starting new node....")

	for _, wallet := range Configuration.Account.Wallets {
		go StartNode(wallet)
	}

	return nil
}

func StartNode(wallet sdkWallet.Wallet) error {
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

	for {
		account, err := Configuration.Account.Client.GetAccount(wallet.Address)
		if err != nil {
			fmt.Printf("Failed to retrieve account data - error: %s", err)
			continue
		}
		nonce := uint64(account.Nonce)
		txs := []sdkTransactions.Transaction{}

		for i := 0; i < Configuration.Concurrency; i++ {
			receiver := randomizeReceiverAddress()
			tx, err := generateTransaction(wallet, receiver, nonce)
			if err != nil {
				fmt.Printf("Error occurred while generating transaction - error: %s\n", err.Error())
				continue
			}
			txs = append(txs, tx)
			nonce++
		}

		BulkSendTransactions(messenger, wallet, txs)

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

	return nil
}

func subscribeToTopics(messenger p2p.Messenger) {
	for _, topic := range Configuration.P2P.Topics {
		messenger.CreateTopic(topic, true)
	}
}

// BulkSendTransactions - sends the provided transactions as a bulk, optimizing transfer between nodes
func BulkSendTransactions(messenger p2p.Messenger, wallet sdkWallet.Wallet, txs []sdkTransactions.Transaction) error {
	if len(txs) == 0 {
		return errors.New("No txs to process")
	}

	senderShardID := sdkTransactions.CalculateShardForAddress(wallet.AddressBytes, Configuration.NumberOfShards)

	transactionsByShards := make(map[uint32][][]byte)

	for _, tx := range txs {
		receiverShardID := sdkTransactions.CalculateShardForAddress(tx.Transaction.RcvAddr, Configuration.NumberOfShards)

		marshalizedTx, err := Configuration.P2P.InternalMarshalizer.Marshal(tx.Transaction)
		if err != nil {
			fmt.Printf("BulkSendTransactions: marshalizer error - %s\n", err.Error())
			continue
		}

		transactionsByShards[receiverShardID] = append(transactionsByShards[receiverShardID], marshalizedTx)
	}

	numOfSentTxs := uint64(0)
	for receiverShardID, txsForShard := range transactionsByShards {
		err := BulkSendTransactionsFromShard(messenger, txsForShard, senderShardID, receiverShardID)
		if err != nil {
			fmt.Printf("sendBulkTransactionsFromShard - error: %s\n", err.Error())
		} else {
			numOfSentTxs += uint64(len(txsForShard))
		}
	}

	return nil
}

// BulkSendTransactionsFromShard - bulk sends the transactions for a given shard
func BulkSendTransactionsFromShard(messenger p2p.Messenger, transactions [][]byte, senderShardID uint32, receiverShardID uint32) error {
	dataPacker, err := partitioning.NewSimpleDataPacker(Configuration.P2P.InternalMarshalizer)
	if err != nil {
		return err
	}

	topic := generateChannelID(transactionTopic, senderShardID, receiverShardID)
	messenger.CreateTopic(topic, true)

	packets, err := dataPacker.PackDataInChunks(transactions, core.MaxBulkTransactionSize)
	if err != nil {
		return err
	}

	for _, buff := range packets {
		go func(bufferToSend []byte) {
			fmt.Printf("BulkSendTransactionsFromShard - topic: %s, size: %d bytes\n", topic, len(bufferToSend))

			err = messenger.BroadcastOnChannelBlocking(
				node.SendTransactionsPipe,
				topic,
				bufferToSend,
			)
			if err != nil {
				fmt.Printf("BroadcastOnChannelBlocking - error: %s\n", err.Error())
			}
		}(buff)
	}

	return nil
}

func generateChannelID(channel string, senderShardID uint32, receiverShardID uint32) string {
	if receiverShardID == core.MetachainShardId {
		return fmt.Sprintf("%s_%d_META", channel, senderShardID)
	}

	if receiverShardID == senderShardID {
		return fmt.Sprintf("%s_%d", channel, senderShardID)
	}

	if senderShardID > receiverShardID {
		return fmt.Sprintf("%s_%d_%d", channel, receiverShardID, senderShardID)
	}

	return fmt.Sprintf("%s_%d_%d", channel, senderShardID, receiverShardID)
}

func randomizeReceiverAddress() string {
	return utils.RandomElementFromArray(Configuration.P2P.TxReceivers)
}

func generateTransaction(wallet sdkWallet.Wallet, receiver string, nonce uint64) (sdkTransactions.Transaction, error) {
	gasParams := Configuration.Account.GasParams

	tx, err := sdkTransactions.GenerateAndSignTransaction(
		wallet,
		receiver,
		0.0,
		false,
		int64(nonce),
		Configuration.P2P.Data,
		gasParams,
		Configuration.Account.Client,
	)
	if err != nil {
		return sdkTransactions.Transaction{}, err
	}

	fmt.Printf("Generated transaction - receiver: %s, nonce: %d, tx hash: %s\n", receiver, nonce, tx.TxHash)

	return tx, nil
}

func broadcastMessage(messenger p2p.Messenger, wallet sdkWallet.Wallet, receiver string, nonce uint64) { //, waitGroup *sync.WaitGroup) {
	//defer waitGroup.Done()
	tx, err := generateTransaction(wallet, receiver, nonce)
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
