package p2p

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go/heartbeat/data"
	"github.com/ElrondNetwork/elrond-go/p2p"
	sdkCrypto "github.com/SebastianJ/elrond-sdk/crypto"
)

var (
	heartbeatTopic   = "heartbeat"
	versionNumber    = "v1.0.131-0-gc00596aae/go1.13.5/linux-amd64"
	validatorNumbers = []int{1, 2, 3, 4, 5, 6}
)

// BulkSendHeartbeats - bulk send heartbeat messages
func BulkSendHeartbeats(messenger p2p.Messenger) {
	if !messenger.HasTopic(heartbeatTopic) {
		messenger.CreateTopic(heartbeatTopic, true)
	}

	for i := 0; i < Configuration.Concurrency; i++ {
		heartbeatData, err := GenerateHeartbeat(messenger)
		if err != nil {
			fmt.Printf("BulkSendHeartbeats - error: %s\n", err.Error())
			continue
		}
		fmt.Printf("Sending heartbeat message to topic %s of %d bytes\n", heartbeatTopic, len(heartbeatData))
		messenger.Broadcast(heartbeatTopic, heartbeatData)
	}
}

// GenerateHeartbeat - generate heartbeat message
func GenerateHeartbeat(messenger p2p.Messenger) ([]byte, error) {
	blsKey, err := sdkCrypto.GenerateBlsKey()
	if err != nil {
		return nil, err
	}

	heartbeat := &data.Heartbeat{
		//Payload:         []byte(fmt.Sprintf("%v", time.Now())),
		Payload:         randomizeData(),
		ShardID:         randomizeShardID(),
		VersionNumber:   string(randomizeData()),
		NodeDisplayName: string(randomizeData()),
		Identity:        string(randomizeData()),
		Pid:             messenger.ID().Bytes(),
	}

	heartbeat.Pubkey, err = blsKey.PrivateKey.GeneratePublic().ToByteArray()
	if err != nil {
		return nil, err
	}

	heartbeatBytes, err := Configuration.P2P.InternalMarshalizer.Marshal(heartbeat)
	if err != nil {
		return nil, err
	}

	signer := sdkCrypto.NewSigner(sdkCrypto.BLS12)
	heartbeat.Signature, err = signer.Sign(blsKey.PrivateKey, heartbeatBytes)
	if err != nil {
		return nil, err
	}

	buffToSend, err := Configuration.P2P.InternalMarshalizer.Marshal(heartbeat)
	if err != nil {
		return nil, err
	}

	return buffToSend, nil
}
