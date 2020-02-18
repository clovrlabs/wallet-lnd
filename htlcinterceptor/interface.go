package htlcinterceptor

import (
	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/lnwire"
)

// ForwardHtlcHandler is a method that is invoked from the switch for every
// incoming htlc that is intended to be forwarded.
// The return value indicates if this handler will take control of this forward
// and resolve it later or let the switch execute its default behavior.
type ForwardHtlcHandler func(inKey channeldb.CircuitKey,
	req lnwire.UpdateAddHTLC) bool

// ForwardResolver is the resolver interface that is intended to be called from
// outside the switch to resolve an existing forward.
type ForwardResolver interface {
	// Resume notifies the intention to resume an existing hold forward. This
	// basically means the caller wants to resume with the default behavior for
	// this htlc which usually means forward it.
	Resume() error

	// Settle notifies the intention to settle an existing hold
	// forward with a given preimage.
	Settle(lntypes.Preimage) error

	// Fails notifies the intention to fail an existing hold forward
	Fail() error
}

// Interceptor is an interface that allow intercepting htlc forwards and decide
// on the next action. It enables the htlc switch to delegate its default
// behavior to a strategy implemented outside the daemon.
type Interceptor interface {
	// InterceptForwardHtlc is called for each forward. It is provided with a
	// ForwardResolver interface that declares three options for resolving
	// a hold forward.
	// The boolean return value denotes the following:
	//   true - means this interceptor is handling the forward and the switch
	//     should wait for a later resolve action.
	//   false - means this controller is not handling the forward and the
	//     switch should keep with its default behavior.
	//
	// In case this Interceptor handled the forward it can later resolve it
	// by calling one of the three nethods in the given ForwardResolver:
	//   Resume - executes the deault behavior (probably just forward it).
	//   Settle - settles the htlc for a given preimage.
	//   Fail - fails the htlc.
	InterceptForwardHtlc(channeldb.CircuitKey,
		lnwire.UpdateAddHTLC, ForwardResolver) bool
}
