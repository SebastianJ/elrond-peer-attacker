package p2p

import (
	"github.com/ElrondNetwork/elrond-go/config"
	sdkAPI "github.com/SebastianJ/elrond-sdk/api"
	"github.com/SebastianJ/elrond-sdk/transactions"
	sdkWallet "github.com/SebastianJ/elrond-sdk/wallet"
)

var Configuration Config

// Config - general config
type Config struct {
	BasePath    string
	Concurrency int
	Verbose     bool
	P2P         P2PConfig
	Account     AccountConfig
}

// P2PConfig - p2p
type P2PConfig struct {
	ElrondConfig   *config.P2PConfig
	MessageCount   int
	Bootnodes      []string
	Peers          int
	Host           string
	Port           int
	Topics         []string
	Protocol       string
	Rendezvous     string
	Data           string
	ConnectionWait int
	Shards         []string
	TxReceivers    []string
}

type AccountConfig struct {
	Wallet    sdkWallet.Wallet
	Nonce     uint64
	GasParams transactions.GasParams
	Client    sdkAPI.Client
}
