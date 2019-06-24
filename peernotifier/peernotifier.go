package peernotifier

import (
	"sync/atomic"

	"github.com/lightningnetwork/lnd/subscribe"
)

// PeerNotifier is a subsystem which connected and disconnected peer
// events pipe through. It takes subscriptions for its events, and whenever
// it receives a new event it notifies its subscribers over the proper channel.
type PeerNotifier struct {
	started uint32
	stopped uint32

	ntfnServer *subscribe.Server
}

// PeerConnectionChangedEvent represents a new event where a peers connection status
// is changed.
type PeerConnectionChangedEvent struct {
	// Channel is the channel that has become open.
	PeerAddress   string
	PeerConnected bool
	PeerPubKey    string
}

// New creates a new channel notifier. The ChannelNotifier gets channel
// events from peers and from the chain arbitrator, and dispatches them to
// its clients.
func New() *PeerNotifier {
	return &PeerNotifier{
		ntfnServer: subscribe.NewServer(),
	}
}

// Start starts the ChannelNotifier and all goroutines it needs to carry out its task.
func (c *PeerNotifier) Start() error {
	if !atomic.CompareAndSwapUint32(&c.started, 0, 1) {
		return nil
	}

	log.Tracef("PeerNotifier %v starting", c)

	if err := c.ntfnServer.Start(); err != nil {
		return err
	}

	return nil
}

// Stop signals the notifier for a graceful shutdown.
func (c *PeerNotifier) Stop() error {
	if !atomic.CompareAndSwapUint32(&c.stopped, 0, 1) {
		return nil
	}

	return c.ntfnServer.Stop()
}

// SubscribePeerEvents returns a subscribe.Client that will receive updates
// any time the Server is made aware of a new event.
func (c *PeerNotifier) SubscribePeerEvents() (*subscribe.Client, error) {
	return c.ntfnServer.Subscribe()
}

// NotifyPeerConnectionChangedEvent notifies that a peers connection has changed.
func (c *PeerNotifier) NotifyPeerConnectionChangedEvent(pubKey string,
	address string, connected bool) {

	// Send the open event to all channel event subscribers.
	event := PeerConnectionChangedEvent{PeerAddress: address, PeerPubKey: pubKey,
		PeerConnected: connected}
	if err := c.ntfnServer.SendUpdate(event); err != nil {
		log.Warnf("Unable to send peer connection changed update: %v", err)
	}
}
