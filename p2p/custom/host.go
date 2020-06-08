package custom

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	badger "github.com/ipfs/go-ds-badger"
	libp2p "github.com/libp2p/go-libp2p"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	libp2p_discovery "github.com/libp2p/go-libp2p-discovery"
	libp2p_host "github.com/libp2p/go-libp2p-host"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

const (
	waitInRetry       = 2 * time.Second
	connectionTimeout = 3 * time.Minute
	findPeerInterval  = 60 * time.Second
	connectionRetry   = 50
	maxMessageSize    = 2_145_728

	concurrency = 100
	protocolID  = "/erd/directsend/1.0.0"
	rendezvous  = "/erd/kad/1.0.0"
)

// Host - generalized structure for p2p communication
type Host struct {
	Host          libp2p_host.Host
	Topics        map[string]*libp2p_pubsub.Topic
	Subscriptions map[string]*libp2p_pubsub.Subscription
	Streams       map[string]network.Stream
	Peer          Peer
	PrivateKey    *libp2pCrypto.Secp256k1PrivateKey
	Discovery     *libp2p_discovery.RoutingDiscovery
	PubSub        *libp2p_pubsub.PubSub
	Dht           *kaddht.IpfsDHT
	Protocol      string
	AllTopics     []string
	Context       context.Context
	Lock          sync.Mutex
}

// Peer is the object for a p2p peer (node)
type Peer struct {
	IP     string         // IP address of the peer
	Port   string         // Port number of the peer
	Addrs  []ma.Multiaddr // MultiAddress of the peer
	PeerID libp2p_peer.ID // PeerID, the pubkey for communication
}

func newHost(peer *Peer, privKey *libp2pCrypto.Secp256k1PrivateKey, topics []string) (*Host, error) {
	ctx := context.Background()

	listenAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", peer.IP, peer.Port))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create listen multiaddr from ip %s and port %#v", peer.IP, peer.Port)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(listenAddr),
		libp2p.Identity(privKey),
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.DefaultTransports,
		//we need the disable relay option in order to save the node's bandwidth as much as possible
		//libp2p.DisableRelay(),
		libp2p.NATPortMap(),
	}

	p2pHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot initialize libp2p host")
	}

	// set the KValue to 50 for DHT
	// 50 is the size of every bucket in the DHT
	//kaddht.KValue = dhtMaximumRequests
	dataStorePath := fmt.Sprintf(".dht-%s-%s", peer.IP, peer.Port)
	dataStore, err := badger.NewDatastore(dataStorePath, nil)
	if err != nil {
		//fmt.Printf("cannot initialize DHT cache at %s - error: %s\n", dataStorePath, err.Error)
		return nil, errors.Wrapf(err, "Badger")
	}

	dht := kaddht.NewDHT(ctx, p2pHost, dataStore)
	if err := dht.Bootstrap(ctx); err != nil {
		//fmt.Printf("cannot bootstrap DHT - error: %s", err.Error())
		return nil, errors.Wrapf(err, "DHT bootstrap")
	}

	discovery := libp2p_discovery.NewRoutingDiscovery(dht)

	options := []libp2p_pubsub.Option{
		libp2p_pubsub.WithMessageSigning(true),
		libp2p_pubsub.WithPeerOutboundQueueSize(1000),
		libp2p_pubsub.WithDiscovery(discovery),
		//libp2p_pubsub.WithMessageAuthor(""),
		//libp2p_pubsub.WithMaxMessageSize(maxMessageSize),
	}

	//pubsub, err := libp2p_pubsub.NewFloodSub(ctx, p2pHost, options...)
	pubsub, err := libp2p_pubsub.NewGossipSub(ctx, p2pHost, options...)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot initialize libp2p pubsub")
	}

	peer.PeerID = p2pHost.ID()

	fmt.Printf("HostID: %s\n", p2pHost.ID().Pretty())

	host := &Host{
		Host:          p2pHost,
		Context:       ctx,
		Topics:        map[string]*libp2p_pubsub.Topic{},
		Subscriptions: map[string]*libp2p_pubsub.Subscription{},
		Peer:          *peer,
		PrivateKey:    privKey,
		PubSub:        pubsub,
		Dht:           dht,
		Discovery:     discovery,
		Protocol:      rendezvous,
		AllTopics:     topics,
	}

	return host, nil
}

func (host *Host) connectToBootnodes(bootnodes []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	bootNodeAddrs, err := StringsToAddrs(bootnodes)
	if err != nil {
		fmt.Printf("Can't parse bootnodes from %+v - error: %s\n", bootnodes, err)
	}

	var mutex = &sync.Mutex{}
	var wg sync.WaitGroup

	connected := false
	for _, peerAddr := range bootNodeAddrs {
		peerinfo, _ := peerstore.InfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < connectionRetry; i++ {
				if connected {
					break
				}

				if err := host.Host.Connect(ctx, *peerinfo); err != nil {
					fmt.Printf("Can't connect to bootnode %+v - error: %s\n", peerAddr, err)
					time.Sleep(waitInRetry)
				} else {
					fmt.Printf("Connected to bootnode %+v\n", peerAddr)
					// it is okay if any bootnode is connected
					mutex.Lock()
					connected = true
					mutex.Unlock()
					break
				}
			}
		}()
	}
	wg.Wait()

	if !connected {
		return fmt.Errorf("[FATAL] error connecting to bootnodes")
	}

	return nil
}

// Advertise - advertises the peer on all required topics
func (host *Host) Advertise() {
	libp2p_discovery.Advertise(host.Context, host.Discovery, rendezvous)

	peerChan, err := host.Discovery.FindPeers(host.Context, rendezvous)
	if err != nil {
		panic(err)
	}

	for peer := range peerChan {
		if peer.ID == host.Host.ID() {
			continue
		}
		fmt.Printf("Found peer: %s\n", peer)

		stream, err := host.Host.NewStream(host.Context, peer.ID, protocol.ID(protocolID))

		if err != nil {
			fmt.Printf("Connection failed: %s", err.Error())
			continue
		} else {
			bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		}

		fmt.Printf("Connected to: %s\n", peer)
	}
}

// ListPeers - displays information about connected peers
func (host *Host) ListPeers() {
	peerIDS := host.Host.Network().Peers()
	fmt.Printf("\n\nFound a total of %d peers\n\n", len(peerIDS))
	for _, peerID := range peerIDS {
		fmt.Printf("Found Peer: %s\n", peerID.Pretty())
	}
	fmt.Println("\n\n")
}

// ConnectToPeer - establishes a direct connection to another identified peer
func (host *Host) ConnectToPeer(peerID peer.ID) error {
	fmt.Printf("Connecting to Peer: %s, Protocol: %s\n", peerID.Pretty(), host.Protocol)

	_, err := host.Host.NewStream(host.Context, peerID, protocol.ID(host.Protocol))
	if err != nil {
		fmt.Printf("Connection failed: %s\n\n", err.Error())
		return err
	}

	fmt.Printf("Connected to peer %s using protocol %s\n\n", peerID.Pretty(), host.Protocol)

	return nil
}

// UpdateStatus - advertise, join topics and list peers
func (host *Host) UpdateStatus() {
	host.Advertise()
	host.JoinTopics()
	host.ListPeers()
}

// PropagateMessages - propagate messages
func (host *Host) PropagateMessages(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	//index := 0
	for {
		host.UpdateStatus()

		/*switch Configuration.P2P.Mode {
		case "txs", "transactions":
			SendTxs(host)
		case "msgs", "messages":
			host.AsyncSendMessagesToTopics()
		default:
			SendTxs(host)
		}
		index++

		if Configuration.P2P.MessageCount > 0 && index >= Configuration.P2P.MessageCount {
			host.Host.Close()
			break
		}*/
	}
}

// JoinTopics - joins the specified topics for a given peer
func (host *Host) JoinTopics() error {
	for _, topic := range host.AllTopics {
		//fmt.Printf("Joining topic %s\n", topic)
		topicHandle, err := host.getTopic(topic)
		if err != nil {
			fmt.Printf("Error joining topic %s - error: %s\n", topic, err.Error())
			return err
		}
		host.Topics[topic] = topicHandle

		/*subscription, err := topicHandle.Subscribe()
		if err != nil {
			fmt.Printf("Error subscribing to topic %s - error: %s\n", topic, err.Error())
			return err
		}
		host.Subscriptions[topic] = subscription*/
	}

	return nil
}

// AsyncSendMessagesToTopics - sends the configured message to all relevant topics
func (host *Host) AsyncSendMessagesToTopics() {
	var waitGroup sync.WaitGroup
	for topic, topicHandle := range host.Topics {
		for i := 0; i < concurrency; i++ {
			waitGroup.Add(1)
			go asyncSendMessageToTopic(topic, topicHandle, &waitGroup)
		}
	}
	waitGroup.Wait()
}

func asyncSendMessageToTopic(name string, handle *libp2p_pubsub.Topic, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	randomNumber := rand.New(rand.NewSource(time.Now().UTC().UnixNano())).Intn(1000000)
	var message strings.Builder
	message.WriteString("")
	message.WriteString(strconv.Itoa(randomNumber))
	p2pMessage := []byte(message.String())

	fmt.Printf("Sending message of %d byte(s) to topic %s ...\n", len(p2pMessage), name)
	if err := handle.Publish(context.Background(), p2pMessage); err != nil {
		fmt.Printf("Failed to send message to topic %s ...\n", name)
	}
}

// SendMessageToTopics - broadcasts a message to multiple topics
func (host *Host) SendMessageToTopics(topics []string, msg []byte) (err error) {
	for _, topic := range topics {
		topicHandle, e := host.getTopic(topic)
		if e != nil {
			err = e
			continue
		}
		e = topicHandle.Publish(context.Background(), msg)
		if e != nil {
			err = e
			continue
		}
	}
	return err
}

func (host *Host) getTopic(topic string) (*libp2p_pubsub.Topic, error) {
	host.Lock.Lock()
	defer host.Lock.Unlock()
	if t, ok := host.Topics[topic]; ok {
		return t, nil
	} else if t, err := host.PubSub.Join(topic); err != nil {
		return nil, errors.Wrapf(err, "cannot join pubsub topic %x", topic)
	} else {
		host.Topics[topic] = t
		return t, nil
	}
}

// StringsToAddrs convert a list of strings to a list of multiaddresses
func StringsToAddrs(addrStrings []string) (maddrs []ma.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := ma.NewMultiaddr(addrString)
		if err != nil {
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}
