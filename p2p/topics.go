package p2p

import (
	"fmt"
	"strings"
)

var (
	Topics = []string{
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
		/*"shardBlocks_0_META",
		"shardBlocks_1_META",
		"txBlockBodies_ALL",
		"validatorTrieNodes_META",
		"accountTrieNodes_META",
		"accountTrieNodes_0_META",
		"accountTrieNodes_1_META",
		"consensus_0",
		"consensus_1",
		"consensus_meta",
		"heartbeat",*/
	}
)

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
			for _, shard := range Configuration.Shards {
				shard = strings.ToUpper(shard)
				topics = append(topics, fmt.Sprintf("%s_%s", baseTopic, shard))

				for _, innerShard := range Configuration.Shards {
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
