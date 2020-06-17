package p2p

import (
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/hashing/blake2b"
	"github.com/ElrondNetwork/elrond-go/marshal"
	sdkAPI "github.com/SebastianJ/elrond-sdk/api"
	"github.com/SebastianJ/elrond-sdk/transactions"
	sdkWallet "github.com/SebastianJ/elrond-sdk/wallet"
)

var Configuration Config

// Config - general config
type Config struct {
	BasePath       string
	Concurrency    int
	Verbose        bool
	P2P            P2PConfig
	Account        AccountConfig
	NumberOfShards uint32
}

// P2PConfig - p2p
type P2PConfig struct {
	ElrondConfig        *config.P2PConfig
	Method              string
	MessageCount        int
	Bootnodes           []string
	Peers               int
	Host                string
	Port                int
	Rotation            int
	Topics              []string
	Protocol            string
	Rendezvous          string
	Data                string
	ConnectionWait      int
	Log                 bool
	Shards              []string
	ShardIDs            []uint32
	TxAmount            float64
	TxReceivers         []string
	TxMarshalizer       *marshal.TxJsonMarshalizer
	InternalMarshalizer *marshal.GogoProtoMarshalizer
	Hasher              *blake2b.Blake2b
}

type AccountConfig struct {
	Wallets   []sdkWallet.Wallet
	GasParams transactions.GasParams
	Client    sdkAPI.Client
}
