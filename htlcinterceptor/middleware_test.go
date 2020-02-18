package htlcinterceptor

import (
	"testing"

	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/lnwire"
)

type resolveStatus uint8

const (
	Hold resolveStatus = iota
	Resumed
	Failed
	Settled
)

var (
	packet = lnwire.UpdateAddHTLC{
		Amount: 100,
		ID:     1,
	}
	circuitKey = channeldb.CircuitKey{
		ChanID: lnwire.NewShortChanIDFromInt(1),
		HtlcID: 1,
	}
)

type mockForwardResolver struct {
	status resolveStatus
}

func (m *mockForwardResolver) Settle(preimage lntypes.Preimage) error {
	m.status = Settled
	return nil
}

func (m *mockForwardResolver) Fail() error {
	m.status = Failed
	return nil
}

func (m *mockForwardResolver) Resume() error {
	m.status = Resumed
	return nil
}

func TestResumeHoldForward(t *testing.T) {

	testResolveForward(t,
		func(m *Middleware, circuitKey channeldb.CircuitKey) error {
			return m.ResumeHoldForward(circuitKey)
		}, Resumed)
}

func TestFailHoldForward(t *testing.T) {
	testResolveForward(t,
		func(m *Middleware, circuitKey channeldb.CircuitKey) error {
			return m.FailHoldForward(circuitKey)
		}, Failed)
}

func TestSettleHoldForward(t *testing.T) {
	testResolveForward(t,
		func(m *Middleware, circuitKey channeldb.CircuitKey) error {
			return m.SettleHoldForward(circuitKey, lntypes.Preimage{})
		}, Settled)
}

func TestRemoveHandler(t *testing.T) {
	middleware := NewMiddleware()
	handlerID := middleware.AddForwardHtlcHandler(func(channeldb.CircuitKey,
		lnwire.UpdateAddHTLC) bool {

		return true
	})
	resolver := mockForwardResolver{status: Hold}

	// pass through the middleware.
	intercepted := middleware.InterceptForwardHtlc(circuitKey, packet, &resolver)
	if !intercepted {
		t.Fatal("packet should be intercepted.")
	}

	middleware.RemoveForwardHtlcHandler(handlerID)

	// pass through the middleware.
	intercepted = middleware.InterceptForwardHtlc(circuitKey, packet, &resolver)
	if intercepted {
		t.Fatal("packet should not be intercepted.")
	}
}

func testResolveForward(t *testing.T,
	resolveFunc func(*Middleware, channeldb.CircuitKey) error,
	expectedStatus resolveStatus) {

	middleware := NewMiddleware()
	middleware.AddForwardHtlcHandler(func(channeldb.CircuitKey,
		lnwire.UpdateAddHTLC) bool {

		return true
	})
	resolver := mockForwardResolver{status: Hold}

	// pass through the middleware.
	intercepted := middleware.InterceptForwardHtlc(circuitKey, packet, &resolver)
	if !intercepted {
		t.Fatal("packet should be intercepted.")
	}

	if err := resolveFunc(middleware, circuitKey); err != nil {
		t.Fatalf("failed to resolve forward %v", err)
	}
	if resolver.status != expectedStatus {
		t.Fatalf("expected status %v got %v", expectedStatus, resolver.status)
	}

	err := resolveFunc(middleware, circuitKey)
	if err != ErrFwdNotExists {
		t.Fatalf("expected ErrFwdNotExists got %v", err)
	}
	if resolver.status != expectedStatus {
		t.Fatalf("expected status %v got %v", expectedStatus, resolver.status)
	}
}
