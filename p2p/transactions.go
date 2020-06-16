package p2p

import (
	"errors"
	"fmt"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/SebastianJ/elrond-peer-attacker/utils"
	sdkTransactions "github.com/SebastianJ/elrond-sdk/transactions"
	sdkWallet "github.com/SebastianJ/elrond-sdk/wallet"
)

var (
	transactionTopic = "transactions"
)

// GenerateAndBulkSendTransactions - generates and sends transactions in bulk
func GenerateAndBulkSendTransactions(messenger p2p.Messenger, wallet sdkWallet.Wallet) error {
	account, err := Configuration.Account.Client.GetAccount(wallet.Address)
	if err != nil {
		fmt.Printf("Failed to retrieve account data - error: %s", err)
		return err
	}
	nonce := uint64(account.Nonce)
	txs := []sdkTransactions.Transaction{}

	for i := 0; i < Configuration.Concurrency; i++ {
		receiver := randomizeReceiverAddress()
		tx, err := generateTransaction(wallet, receiver, nonce)
		if err != nil {
			fmt.Printf("Error occurred while generating transaction - error: %s\n", err.Error())
			return err
		}
		txs = append(txs, tx)
		nonce++
	}

	BulkSendTransactions(messenger, wallet, txs)

	return nil
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

	topic := generateTopic(transactionTopic, senderShardID, receiverShardID)
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
