package custom

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	randFactory "github.com/ElrondNetwork/elrond-go/p2p/libp2p/rand/factory"
	"github.com/btcsuite/btcd/btcec"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
)

var (
	host             string = "0.0.0.0"
	port             int    = 15000
	maximumPort      int    = 65000 // technically 65535 is the maximum, but use some extra buffer
	pageSize         int    = 100
	pages            int    = 1
	totalCount       int    = 0
	processed        int    = 0
	waitAfterConnect int    = 30
)

// ConnectPeers - set up the peer connections for the required amount of peers
func ConnectPeers(portIndex int, peers int, topics []string, bootnodes []string) ([]*Host, int) {
	pages = calculatePageCount(peers, pageSize)
	hostsChannel := make(chan *Host, peers)

	var waitGroup sync.WaitGroup
	for page := 0; page < pages; page++ {
		for i := 0; i < pageSize; i++ {
			_, ok := validPosition(page, pageSize, i, peers)
			if ok {
				waitGroup.Add(1)

				if portIndex < maximumPort {
					portIndex++
				} else {
					portIndex = port
				}

				go ConnectPeer(portIndex, topics, bootnodes, hostsChannel, &waitGroup)
			}
		}

		waitGroup.Wait()
	}
	close(hostsChannel)

	fmt.Printf("Finished setting up connections for %d peers\n", peers)
	fmt.Printf("Waiting %d seconds to let peers connect to each other before proceeding\n", waitAfterConnect)
	time.Sleep(time.Second * time.Duration(waitAfterConnect))

	hosts := []*Host{}
	for host := range hostsChannel {
		host.Advertise()
		hosts = append(hosts, host)
	}

	return hosts, portIndex
}

// ConnectPeer - set up a peer and connect it to the supplied boot nodes
func ConnectPeer(port int, topics []string, bootnodes []string, hostsChannel chan<- *Host, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	fmt.Printf("Setting up new p2p peer using address %s and port %d\n\n", host, port)

	host, err := InitializePeer(host, port, topics)
	if err != nil {
		fmt.Printf("Failed to setup p2p peer, error: %s\n", err.Error())
		return
	}

	if err = host.connectToBootnodes(bootnodes); err != nil {
		fmt.Printf("Failed to connect to bootnodes, error: %s\n", err.Error())
		return
	}

	hostsChannel <- host
}

// InitializePeer - sets up a new p2p connection
func InitializePeer(address string, port int, topics []string) (*Host, error) {
	nodePrivKey, err := generatePrivateKey("")
	if err != nil {
		return nil, err
	}

	peer := Peer{
		IP:   address,
		Port: strconv.Itoa(port),
	}

	host, err := newHost(&peer, nodePrivKey, topics)
	if err != nil {
		return nil, err
	}

	return host, nil
}

func calculatePageCount(totalCount int, pageSize int) int {
	if totalCount > 0 {
		pageNumber := math.RoundToEven(float64(totalCount) / float64(pageSize))
		if math.Mod(float64(totalCount), float64(pageSize)) > 0 {
			return int(pageNumber) + 1
		}

		return int(pageNumber)
	} else {
		return 0
	}
}

func validPosition(page int, pageSize int, index int, totalCount int) (position int, ok bool) {
	position = ((page * pageSize) + index)
	ok = position <= (totalCount - 1)
	return position, ok
}

func generatePrivateKey(seed string) (*libp2pCrypto.Secp256k1PrivateKey, error) {
	randReader, err := randFactory.NewRandFactory(seed)
	if err != nil {
		return nil, err
	}

	prvKey, _ := ecdsa.GenerateKey(btcec.S256(), randReader)

	return (*libp2pCrypto.Secp256k1PrivateKey)(prvKey), nil
}
