package htlcinterceptor

import (
	"errors"
	"sync"

	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/lnwire"
)

var (
	//ErrFwdNotExists is an error returned when the caller tries to resolve
	// a forward that doesn't exist anymore.
	ErrFwdNotExists = errors.New("forward does not exist")
)

// Middleware is a middleware for other sub systems, including the RPC layer,
// to be able to intercept htlc forwards. It holds a list of handlers that are
// invoked sequentially per forward to figure out if one of them is interested
// to handle the forward outside the switch. In addition it exposed API for
// resolving existing hold forward.
type Middleware struct {
	sync.RWMutex
	// ForwardHtlcHandler is the callback that is called for each forward of
	// incoming htlc. It should return true if it is interested in handling
	// the forwardHtlc.
	fwdHandlers  map[uint64]ForwardHtlcHandler
	fwdHandlerID uint64

	// holdForwards is a map of current hold forwards and their corresponding
	// ForwardResolver.
	holdForwards map[channeldb.CircuitKey]ForwardResolver
}

// NewMiddleware initializes a Middleware.
func NewMiddleware() *Middleware {
	return &Middleware{
		fwdHandlers:  make(map[uint64]ForwardHtlcHandler),
		holdForwards: make(map[channeldb.CircuitKey]ForwardResolver),
	}
}

// AddForwardHtlcHandler adds a ForwardHtlcHandler.
func (c *Middleware) AddForwardHtlcHandler(
	handler ForwardHtlcHandler) uint64 {

	c.Lock()
	defer c.Unlock()
	c.fwdHandlerID++
	c.fwdHandlers[c.fwdHandlerID] = handler

	// Return the id so that a caller can call RemoveForwardHtlcHandler.
	return c.fwdHandlerID
}

// RemoveForwardHtlcHandler removes a ForwardHtlcHandler.
func (c *Middleware) RemoveForwardHtlcHandler(id uint64) {
	c.Lock()
	delete(c.fwdHandlers, id)
	c.Unlock()
}

// InterceptForwardHtlc is called before each forward and query its registered
// handlers to see if one of them is interested in handling (hold) this
// forward. In case one of them returns true the forward will be marked as hold
// and signals the switch to wait for a later resolve action which should be
// done later by calling one of:
// * ResumeHoldForward
// * SettleHoldForward
// * FailHoldForward.
func (c *Middleware) InterceptForwardHtlc(
	circuitKey channeldb.CircuitKey,
	htlc lnwire.UpdateAddHTLC, resolver ForwardResolver) bool {

	c.Lock()
	defer c.Unlock()
	for _, actionResolver := range c.fwdHandlers {
		if actionResolver(circuitKey, htlc) {
			c.holdForwards[circuitKey] = resolver
			return true
		}
	}

	// None of the resolvers handled the forward.
	return false
}

// ResumeHoldForward notifies the intention to resume an existing hold forward.
// This basically means the caller wants to resume with the default behavior
// for this htlc which usually means forward it. If an existing hold forward
// doesn't exist it returns an error.
func (c *Middleware) ResumeHoldForward(
	circuitKey channeldb.CircuitKey) error {

	return c.resolveHoldForward(circuitKey,
		func(resolver ForwardResolver) error {
			return resolver.Resume()
		})
}

// SettleHoldForward notifies the intention to settle an existing hold forward
// with a given preimage. If an existing hold forward doesn't exist it returns
// an error.
func (c *Middleware) SettleHoldForward(
	circuitKey channeldb.CircuitKey, preimage lntypes.Preimage) error {

	return c.resolveHoldForward(circuitKey,
		func(resolver ForwardResolver) error {
			return resolver.Settle(preimage)
		})
}

// FailHoldForward notifies the intention to fail an existing hold forward. If
// an existing hold forward doesn't exist it returns an error.
func (c *Middleware) FailHoldForward(
	circuitKey channeldb.CircuitKey) error {

	return c.resolveHoldForward(circuitKey,
		func(resolver ForwardResolver) error {
			return resolver.Fail()
		})
}

func (c *Middleware) resolveHoldForward(
	circuitKey channeldb.CircuitKey, resolveFunc func(ForwardResolver) error) error {

	c.Lock()
	resolver, ok := c.holdForwards[circuitKey]
	c.Unlock()
	if !ok {
		return ErrFwdNotExists
	}
	if err := resolveFunc(resolver); err != nil {
		return err
	}

	c.Lock()
	delete(c.holdForwards, circuitKey)
	c.Unlock()

	return nil
}

// A compile-time constraint to ensure Middleware implements the
// Interceptor interface.
var _ Interceptor = (*Middleware)(nil)
