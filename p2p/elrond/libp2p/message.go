package libp2p

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/p2p/message"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// NewMessage returns a new instance of a Message object
func NewMessage(msg *pubsub.Message) (*message.Message, error) {
	newMsg := &message.Message{
		FromField:      msg.From,
		DataField:      msg.Data,
		SeqNoField:     msg.Seqno,
		TopicsField:    msg.TopicIDs,
		SignatureField: msg.Signature,
		KeyField:       msg.Key,
	}

	id, err := peer.IDFromBytes(newMsg.From())
	if err != nil {
		return nil, err
	}

	newMsg.PeerField = core.PeerID(id)
	return newMsg, nil
}
