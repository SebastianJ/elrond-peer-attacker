package p2p

import (
	"github.com/ElrondNetwork/elrond-go/config"
)

var Configuration Config

// Config - general config
type Config struct {
	BasePath       string `yaml:"-"`
	Concurrency    int    `yaml:"-"`
	Verbose        bool   `yaml:"-"`
	ElrondConfig   *config.P2PConfig
	MessageCount   int      `yaml:"-"`
	Bootnodes      []string `yaml:"-"`
	Peers          int      `yaml:"-"`
	Host           string   `yaml:"-"`
	Port           int      `yaml:"-"`
	Topics         []string `yaml:"-"`
	Protocol       string   `yaml:"-"`
	Rendezvous     string   `yaml:"-"`
	Data           string   `yaml:"-"`
	ConnectionWait int      `yaml:"-"`
	Shards         []string `yaml:"-"`
}
