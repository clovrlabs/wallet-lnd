// +build routerrpc

package routerrpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/htlcswitch"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/lnwire"
)

var (
	// defaultForwardInterceptionTimeout is the time after which an
	// interception request will time out and return false if it hasn't yet
	// received a response.
	defaultForwardInterceptionTimeout = 3 * time.Second

	// ErrFwdNotExists is an error returned when the caller tries to resolve
	// a forward that doesn't exist anymore.
	ErrFwdNotExists = errors.New("forward does not exist")

	// ErrMissingPreimage is an error returned when the caller tries to settle
	// a forward and doesn't provide a preimage.
	ErrMissingPreimage = errors.New("missing preimage")
)

// interceptionInfo is used in the HtlcInterceptor bidirectional stream and
// encapsulates the request information sent from the ForwardInterceptor to the
// RPCServer.
type interceptionInfo struct {
	packet       htlcswitch.InterceptedForward
	responseChan chan bool
}

// HtlcInterceptor is a bidirectional stream for streaming interception
// requests to the caller.
// Upon connection it does the following:
// 1. Check if there is already a live stream, if yes it rejects the request.
// 2. Regsitered a ForwardInterceptor
// 3. Delivers to the caller every √√ and detect his answer.
// It uses a local implementation of holdForwardsStore to keep all the hold
// forwards and find them when manual resolution is later needed.
func (s *Server) HtlcInterceptor(stream Router_HtlcInterceptorServer) error {

	// Create two channels to handle requests and responses respectively.
	newRequests := make(chan *interceptionInfo)
	responses := make(chan ForwardHtlcInterceptResponse)

	// Define a quit channel that will be used to signal to the
	// HtlcForwardHandler whether the stream still exists.
	quit := make(chan struct{})
	defer close(quit)

	// forwardHandler coordinates between the InterceptableHtlcForwarder and
	// the RPC responses. It intercepts the requests, wait for the RPC response
	// and return the desired result.
	forwardHandler := func(packet htlcswitch.InterceptedForward) bool {

		respChan := make(chan bool, 1)
		newRequest := &interceptionInfo{
			packet:       packet,
			responseChan: respChan,
		}

		// timeout is the time after which ForwardHtlcInterceptRequest expire.
		timeout := time.After(defaultForwardInterceptionTimeout)

		// Send the request to the newRequests channel.
		select {
		case newRequests <- newRequest:
		case <-timeout:
			log.Errorf("Forward handler returned false - reached timeout of %d",
				defaultForwardInterceptionTimeout)
			return false
		case <-quit:
			return false
		case <-s.quit:
			return false
		}

		// Receive the response and return it. If no response has been received
		// in defaultForwardInterceptionTimeout, then return false.
		select {
		case resp := <-respChan:
			return resp
		case <-timeout:
			log.Errorf("Forward handler returned false - reached timeout of %d",
				defaultForwardInterceptionTimeout)
			return false
		case <-quit:
			return false
		case <-s.quit:
			return false
		}
	}

	// Register our forwardHandler.
	interceptableForwarder := s.cfg.RouterBackend.InterceptableForwarder
	interceptableForwarder.SetInterceptor(forwardHandler)
	defer s.onInterceptorDisconnected()

	// errChan is used by the receive loop to signal any errors that occur
	// during reading from the stream. This is primarily used to shutdown the
	// send loop in the case of an RPC client disconnecting.
	errChan := make(chan error, 1)

	// Read user responses and dispatch them to the responses channel.
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				errChan <- err
				return
			}

			// Now that we have the response from the RPC client, send it to
			// the responses chan.
			select {
			case responses <- *resp:
			case <-quit:
				return
			case <-s.quit:
				return
			}
		}
	}()

	// Read both dispatched requests and responses.
	// For every request send it over the user rpc stream to reach the caller.
	// For every response coming back from the RPC stream signal its assigned
	// response channel so the switch will have its result back.
	pendingRequests := make(map[channeldb.CircuitKey]chan bool)
	for {
		select {
		case newRequest := <-newRequests:

			htlc := newRequest.packet.Packet()
			inKey := newRequest.packet.CircuitKey()
			pendingRequests[inKey] = newRequest.responseChan

			// A ChannelAcceptRequest has been received, send it to the client.
			interceptionRequest := &ForwardHtlcInterceptRequest{
				CircuitKey: &CircuitKey{
					IncomingChanId: inKey.ChanID.ToUint64(),
					HtlcId:         inKey.HtlcID,
				},
				HtlcPaymentHash: htlc.PaymentHash[:],
				AmountSat:       uint64(htlc.Amount.ToSatoshis()),
				Expiry:          htlc.Expiry,
			}

			if err := stream.Send(interceptionRequest); err != nil {
				return err
			}
		case resp := <-responses:
			// Look up the appropriate response channel to send on given
			// the circuit key. If a channel is found, send the response over
			// it.
			circuitKey := channeldb.CircuitKey{
				ChanID: lnwire.NewShortChanIDFromInt(resp.CircuitKey.IncomingChanId),
				HtlcID: resp.CircuitKey.HtlcId,
			}
			respChan, ok := pendingRequests[circuitKey]
			if !ok {
				continue
			}

			// Send the response boolean over the buffered response channel.
			respChan <- resp.Hold

			// Delete the channel from the acceptRequests map.
			delete(pendingRequests, circuitKey)
		case err := <-errChan:
			log.Errorf("Received an error: %v, shutting down", err)
			return err
		case <-s.quit:
			return fmt.Errorf("RPC server is shutting down")
		}
	}
}

// onInterceptorDisconnected removes all previousely holden forwards from
// the store. Before they are removed it ensure to resume as the default
// behavior.
func (s *Server) onInterceptorDisconnected() {
	// First de-register the current interceptor so we'll stop getting
	// more interceptions.
	interceptableForwarder := s.cfg.RouterBackend.InterceptableForwarder
	interceptableForwarder.SetInterceptor(nil)

	// Now drain and resume all previousely forwards.
	s.holdStore.drain(func(forward htlcswitch.InterceptedForward) {
		if err := forward.Resume(); err != nil {
			log.Errorf("failed to resume hold forward %v", err)
		}
	})
}

// ResolveHoldForward resolves a previousely intercepted forward by either
// Resume, Fail or Settle. In case of settle a valid preimage needs to be
// populated.
func (s *Server) ResolveHoldForward(ctx context.Context,
	in *ResolveHoldForwardRequest) (*ResolveHoldForwardResponse, error) {

	circuitKey := channeldb.CircuitKey{
		ChanID: lnwire.NewShortChanIDFromInt(in.CircuitKey.IncomingChanId),
		HtlcID: in.CircuitKey.HtlcId,
	}
	interceptedForward, err := s.holdStore.get(circuitKey)
	if err != nil {
		return nil, err
	}
	switch in.Action {
	case ResolveHoldForwardAction_RESUME:
		return &ResolveHoldForwardResponse{}, interceptedForward.Resume()
	case ResolveHoldForwardAction_FAIL:
		return &ResolveHoldForwardResponse{}, interceptedForward.Fail()
	case ResolveHoldForwardAction_SETTLE:
		if in.RPreimage == nil {
			return nil, ErrMissingPreimage
		}
		preimage, err := lntypes.MakePreimage(in.RPreimage)
		if err != nil {
			return nil, err
		}
		err = interceptedForward.Settle(preimage)
		return &ResolveHoldForwardResponse{}, err
	default:
		return nil, fmt.Errorf("unrecognized resolve action %v", in.Action)
	}
}

type holdForwardsStore struct {
	sync.Mutex

	forwarder htlcswitch.InterceptableHtlcForwarder
	// holdForwards is a map of current hold forwards and their corresponding
	// ForwardResolver.
	holdForwards map[channeldb.CircuitKey]htlcswitch.InterceptedForward
}

// newHoldForwardsStore creates a new store
func newHoldForwardsStore() *holdForwardsStore {
	return &holdForwardsStore{
		holdForwards: make(
			map[channeldb.CircuitKey]htlcswitch.InterceptedForward),
	}
}

// hold just store the intercepted forward
func (h *holdForwardsStore) hold(forward htlcswitch.InterceptedForward) {
	h.Lock()
	h.holdForwards[forward.CircuitKey()] = forward
	h.Unlock()
}

// get find a previousely holden forward providing its key.
func (h *holdForwardsStore) get(key channeldb.CircuitKey) (
	htlcswitch.InterceptedForward, error) {

	h.Lock()
	defer h.Unlock()
	interceptedForward, ok := h.holdForwards[key]
	if !ok {
		return nil, ErrFwdNotExists
	}
	return interceptedForward, nil
}

// drain removes all previousely holden forwards and allows to pass a callback
// that will be called for each one before it is removed.
func (h *holdForwardsStore) drain(
	resolver func(forward htlcswitch.InterceptedForward)) {

	h.Lock()
	defer h.Unlock()
	for key, forward := range h.holdForwards {
		resolver(forward)
		delete(h.holdForwards, key)
	}
}
