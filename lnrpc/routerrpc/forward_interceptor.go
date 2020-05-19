package routerrpc

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/htlcswitch"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/lnwire"
)

var (
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
	rpcInterceptor := newRPCInterceptor(s, stream)
	if !atomic.CompareAndSwapInt32(&s.rpcInterceptorActive, 0, 1) {
		return ErrInterceptorAlreadyExists
	}

	// run the interceptor and make sure we de-activate it when it finishes.
	defer func() {
		atomic.CompareAndSwapInt32(&s.rpcInterceptorActive, 1, 0)
	}()
	return rpcInterceptor.run()
}

// rpcInterceptor is a helper struct that handles the lifecycle of an rpc
// interceptor streaming session.
// It is created when the stream opens and disconnects when the stream closes.
type rpcInterceptor struct {

	// server is the Server reference
	server *Server

	// holdForwards is a map of current hold forwards and their corresponding
	// ForwardResolver.
	holdForwards map[channeldb.CircuitKey]htlcswitch.InterceptedForward

	// stream is the bidirectional RPC stream
	stream Router_HtlcInterceptorServer

	// quit is a channel that is closed when this rpcInterceptor is shutting
	// down.
	quit chan struct{}

	// intercepted is where we stream all intercepted packets coming from
	// the switch.
	intercepted chan htlcswitch.InterceptedForward

	wg sync.WaitGroup
}

// newRPCInterceptor creates a new rpcInterceptor.
func newRPCInterceptor(server *Server, stream Router_HtlcInterceptorServer) *rpcInterceptor {
	return &rpcInterceptor{
		server: server,
		stream: stream,
		holdForwards: make(
			map[channeldb.CircuitKey]htlcswitch.InterceptedForward),
		quit:        make(chan struct{}),
		intercepted: make(chan htlcswitch.InterceptedForward),
	}
}

// run sends the intercepted packets to the client and receives the
// corersponding responses. On one hand it regsitered itself as an interceptor
// that receives the switch packets and on the other hand launches a go routine
// to read from the client stream.
// To coordinate all this and make sure it is safe for concurrent access all
// packets are sent to the main where they are handled.
func (r *rpcInterceptor) run() error {
	// make sure we disconnect and resolves all remaining packets if any.
	defer r.onDisconnect()

	// Register our interceptor so we receive all forwarded packets.
	interceptableForwarder := r.server.cfg.RouterBackend.InterceptableForwarder
	interceptableForwarder.SetInterceptor(r.onIntercept)

	// start a go routine that reads client resolutions.
	errChan := make(chan error)
	resolutionRequests := make(chan *ForwardHtlcInterceptResponse)
	r.wg.Add(1)
	go r.readClientResponses(resolutionRequests, errChan)

	// run the main loop that synchronizes both sides input into one go routine.
	for {
		select {
		case intercepted := <-r.intercepted:
			log.Tracef("sending intercepted packet to client %v", intercepted)
			// in case we couldn't forward we exit the loop and drain the
			// current interceptor as this indicates on a connection problem.
			if err := r.holdAndforwardToClient(intercepted); err != nil {
				return err
			}
		case resolution := <-resolutionRequests:
			log.Tracef("resolving intercepted packet %v", resolution)
			// in case we couldn't resolve we just add a log line since this
			// does not indicate on any connection problem.
			if err := r.resolveFromClient(resolution); err != nil {
				log.Warnf("client resolution of intercepted "+
					"packet failed %v", err)
			}
		case err := <-errChan:
			log.Errorf("Received an error: %v, shutting down", err)
			return err
		case <-r.server.quit:
			return nil
		}
	}
}

// onIntercept is the function that is called by the switch for every forwarded
// packet. Our interceptor makes sure we hold the packet and then signal to the
// main loop to handle the packet. We only return true if we were able
// to deliver the packet to the main loop.
func (r *rpcInterceptor) onIntercept(p htlcswitch.InterceptedForward) bool {
	select {
	case r.intercepted <- p:
		return true
	case <-r.quit:
		return false
	case <-r.server.quit:
		return false
	}
}

func (r *rpcInterceptor) readClientResponses(
	resolutionChan chan *ForwardHtlcInterceptResponse, errChan chan error) {

	defer r.wg.Done()
	for {
		resp, err := r.stream.Recv()
		if err != nil {
			errChan <- err
			return
		}

		// Now that we have the response from the RPC client, send it to
		// the responses chan.
		select {
		case resolutionChan <- resp:
		case <-r.quit:
			return
		case <-r.server.quit:
			return
		}
	}
}

// holdAndforwardToClient forwards the intercepted htlc to the client.
func (r *rpcInterceptor) holdAndforwardToClient(
	forward htlcswitch.InterceptedForward) error {

	htlc := forward.Packet()
	inKey := forward.CircuitKey()

	// First hold the forward, then send to client.
	r.holdForwards[forward.CircuitKey()] = forward
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
func (r *rpcInterceptor) resolveFromClient(
	in *ForwardHtlcInterceptResponse) error {

	circuitKey := channeldb.CircuitKey{
		ChanID: lnwire.NewShortChanIDFromInt(in.CircuitKey.IncomingChanId),
		HtlcID: in.CircuitKey.HtlcId,
	}
	var interceptedForward htlcswitch.InterceptedForward
	interceptedForward, ok := r.holdForwards[circuitKey]
	if !ok {
		return ErrFwdNotExists
	}
	delete(r.holdForwards, circuitKey)

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

// onDisconnect removes all previousely held forwards from
// the store. Before they are removed it ensure to resume as the default
// behavior.
func (r *rpcInterceptor) onDisconnect() {
	// First de-register the current interceptor so we'll stop getting
	// more interceptions.
	interceptableForwarder := r.server.cfg.RouterBackend.InterceptableForwarder
	interceptableForwarder.SetInterceptor(nil)

	// Then close the channel so all go routine will exit.
	close(r.quit)

	log.Infof("RPC interceptor disconnected, resolving held packets")
	for key, forward := range r.holdForwards {
		if err := forward.Resume(); err != nil {
			log.Errorf("failed to resume hold forward %v", err)
		}
		delete(r.holdForwards, key)
	}
	r.wg.Wait()
}
