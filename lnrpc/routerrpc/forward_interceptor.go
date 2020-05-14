// +build routerrpc

package routerrpc

import (
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

	// ErrInterceptorAlreadyExists is an error returned when the a new stream
	// is opened and there is already one active interceptor.
	// The user must disconnect prior to open another stream.
	ErrInterceptorAlreadyExists = errors.New("interceptor already exists")
)

// HtlcInterceptor is a bidirectional stream for streaming interception
// requests to the caller.
// Upon connection it does the following:
// 1. Check if there is already a live stream, if yes it rejects the request.
// 2. Regsitered a ForwardInterceptor
// 3. Delivers to the caller every √√ and detect his answer.
// It uses a local implementation of holdForwardsStore to keep all the hold
// forwards and find them when manual resolution is later needed.
func (s *Server) HtlcInterceptor(stream Router_HtlcInterceptorServer) error {
	// We ensure there is only one interceptor at a time.
	s.rpcInterceptorMtx.Lock()
	if s.currentRpcInterceptor != nil {
		s.rpcInterceptorMtx.Unlock()
		return ErrInterceptorAlreadyExists
	}
	rpcInterceptor := newRpcInterceptor(stream)
	s.currentRpcInterceptor = rpcInterceptor
	s.rpcInterceptorMtx.Unlock()

	var wg sync.WaitGroup
	quit := make(chan struct{})
	defer s.onInterceptorDisconnected(rpcInterceptor, quit, wg)

	// this go routine reads from the client stream and and then signal to the
	// main loop to handle the packet.
	clientResolutionsChan := make(chan *ForwardHtlcInterceptResponse)
	errChan := make(chan error)
	wg.Add(1)
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
			case clientResolutionsChan <- resp:
			case <-quit:
				return
			case <-s.quit:
				return
			}
		}
	}()

	// Our interceptor makes sure we hold the packet and then signal to the
	// main loop to handle the packet. We only return true if we were able
	// to deliver the packet to the main loop.
	interceptedPacketsChan := make(chan htlcswitch.InterceptedForward)
	fromSwitchInterceptor := func(packet htlcswitch.InterceptedForward) bool {
		rpcInterceptor.hold(packet)
		select {
		case interceptedPacketsChan <- packet:
			return true
		case <-quit:
			return false
		case <-s.quit:
			return false
		}
	}

	// Register our interceptor and make sure we resolve the holden packets
	// on disconnect.
	interceptableForwarder := s.cfg.RouterBackend.InterceptableForwarder
	interceptableForwarder.SetInterceptor(fromSwitchInterceptor)

	for {
		select {
		case intercepted := <-interceptedPacketsChan:
			// in case we couldn't forward we exit the loop and drain the
			// current interceptor as this indicates on a connection problem.
			if err := rpcInterceptor.forwardToClient(intercepted); err != nil {
				return err
			}
		case resolution := <-clientResolutionsChan:
			// in case we couldn't resolve we just add a log line since this
			// does not indicate on any connection problem.
			if err := rpcInterceptor.resolveFromClient(resolution); err != nil {
				log.Warnf("client resolution of intercepted "+
					"packet failed %v", err)
			}
		case err := <-errChan:
			log.Errorf("Received an error: %v, shutting down", err)
			return err
		case <-s.quit:
			return nil
		}
	}
}

// onInterceptorDisconnected removes all previousely holden forwards from
// the store. Before they are removed it ensure to resume as the default
// behavior.
func (s *Server) onInterceptorDisconnected(interceptor *rpcInterceptor,
	quit chan struct{}, wg sync.WaitGroup) {

	interceptableForwarder := s.cfg.RouterBackend.InterceptableForwarder

	// First de-register the current interceptor so we'll stop getting
	// more interceptions and wait for the go routine to exit.
	interceptableForwarder.SetInterceptor(nil)
	close(quit)
	wg.Wait()

	// Now drain and resume all previousely forwards.
	interceptor.drain(func(forward htlcswitch.InterceptedForward) {
		if err := forward.Resume(); err != nil {
			log.Errorf("failed to resume hold forward %v", err)
		}
	})

	// finally remove the current interceptor allows another one to connect.
	s.rpcInterceptorMtx.Lock()
	s.currentRpcInterceptor = nil
	s.rpcInterceptorMtx.Unlock()
}

// rpcInterceptor is a helper struct that handles the lifecycle of an rpc
// interceptor streaming session.
// It is created when the stream opens and drains when the stream closes.
type rpcInterceptor struct {
	sync.Mutex

	// holdForwards is a map of current hold forwards and their corresponding
	// ForwardResolver.
	holdForwards map[channeldb.CircuitKey]htlcswitch.InterceptedForward

	// stream is the bidirectional stream
	stream Router_HtlcInterceptorServer
}

// newRpcInterceptor creates a new rpcInterceptor
func newRpcInterceptor(stream Router_HtlcInterceptorServer) *rpcInterceptor {
	return &rpcInterceptor{
		stream: stream,
		holdForwards: make(
			map[channeldb.CircuitKey]htlcswitch.InterceptedForward),
	}
}

// hold just store the intercepted forward
func (r *rpcInterceptor) hold(forward htlcswitch.InterceptedForward) {
	r.Lock()
	r.holdForwards[forward.CircuitKey()] = forward
	r.Unlock()
}

// forwardToClient forwards the intercepted htlc to the client.
func (r *rpcInterceptor) forwardToClient(forward htlcswitch.InterceptedForward) error {
	htlc := forward.Packet()
	inKey := forward.CircuitKey()

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

	return r.stream.Send(interceptionRequest)
}

// resolveFromClient handles a resolution arrived from the client.
func (r *rpcInterceptor) resolveFromClient(in *ForwardHtlcInterceptResponse) error {
	circuitKey := channeldb.CircuitKey{
		ChanID: lnwire.NewShortChanIDFromInt(in.CircuitKey.IncomingChanId),
		HtlcID: in.CircuitKey.HtlcId,
	}
	var interceptedForward htlcswitch.InterceptedForward
	r.Lock()
	interceptedForward, ok := r.holdForwards[circuitKey]
	if !ok {
		r.Unlock()
		return ErrFwdNotExists
	}
	delete(r.holdForwards, circuitKey)
	r.Unlock()

	switch in.Action {
	case ResolveHoldForwardAction_RESUME:
		return interceptedForward.Resume()
	case ResolveHoldForwardAction_FAIL:
		return interceptedForward.Fail()
	case ResolveHoldForwardAction_SETTLE:
		if in.RPreimage == nil {
			return ErrMissingPreimage
		}
		preimage, err := lntypes.MakePreimage(in.RPreimage)
		if err != nil {
			return err
		}
		err = interceptedForward.Settle(preimage)
		return err
	default:
		return fmt.Errorf("unrecognized resolve action %v", in.Action)
	}
}

// drain removes all previousely holden forwards and allows to pass a callback
// that will be called for each one before it is removed.
func (h *rpcInterceptor) drain(
	resolver func(forward htlcswitch.InterceptedForward)) {

	h.Lock()
	defer h.Unlock()
	for key, forward := range h.holdForwards {
		resolver(forward)
		delete(h.holdForwards, key)
	}
}
