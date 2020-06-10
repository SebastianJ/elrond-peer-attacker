package p2p

import (
	"fmt"
	"sync"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/SebastianJ/elrond-peer-attacker/p2p/custom"
)

var (
	host             string = "0.0.0.0"
	peers            int    = 100
	port             int    = 15000
	maximumPort      int    = 65000 // technically 65535 is the maximum, but use some extra buffer
	pageSize         int    = 100
	pages            int    = 1
	totalCount       int    = 0
	processed        int    = 0
	waitAfterConnect int    = 30
)

func StartCustomNodes(p2pConfig *config.P2PConfig, nodeCount int, addresses []string) error {
	fmt.Println("Starting new nodes....")

	//cleanUpDHT()

	hosts, _ := custom.ConnectPeers(port, nodeCount, Configuration.P2P.Shards, p2pConfig.KadDhtPeerDiscovery.InitialPeerList)

	var waitGroup sync.WaitGroup

	for _, host := range hosts {
		waitGroup.Add(1)
		host.PropagateMessages(&waitGroup)
	}

	waitGroup.Wait()

	return nil
}

/*func cleanUpDHT() {
	dhtPath := filepath.Join(Configuration.BasePath, ".dht*")
	files, err := filepath.Glob(dhtPath)
	if err == nil && len(files) > 0 {
		for _, file := range files {
			os.Remove(file)
		}
	}
}*/
