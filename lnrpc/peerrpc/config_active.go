// +build peerrpc

package peerrpc

import (
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/lightningnetwork/lnd/peernotifier"
)

// Config is the primary configuration struct for the peer notifier RPC server.
// It contains all the items required for the server to carry out its duties.
// The fields with struct tags are meant to be parsed as normal configuration
// options, while if able to be populated, the latter fields MUST also be
// specified.
type Config struct {
	// PeerNotifierMacPath is the path for the peer notifier macaroon. If
	// unspecified then we assume that the macaroon will be found under the
	// network directory, named DefaultPeerNotifierMacFilename.
	PeerNotifierMacPath string `long:"peermacaroonpath" description:"Path to the peer notifier macaroon"`

	// NetworkDir is the main network directory wherein the peer notifier
	// RPC server will find the macaroon named
	// DefaultPeerNotifierMacFilename.
	NetworkDir string

	// MacService is the main macaroon service that we'll use to handle
	// authentication for the chain notifier RPC server.
	MacService *macaroons.Service

	// PeerNotifier is the peer notifier instance that backs the peer
	// notifier RPC server. The job of the peer notifier RPC server is
	// simply to proxy valid requests to the active peer notifier instance.
	PeerNotifier *peernotifier.PeerNotifier
}
