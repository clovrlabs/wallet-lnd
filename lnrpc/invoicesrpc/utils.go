package invoicesrpc

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/zpay32"
)

// CreateRPCInvoice creates an *lnrpc.Invoice from the *channeldb.Invoice.
func CreateRPCInvoice(invoice *channeldb.Invoice,
	activeNetParams *chaincfg.Params) (*lnrpc.Invoice, error) {

	paymentRequest := string(invoice.PaymentRequest)
	decoded, err := zpay32.Decode(paymentRequest, activeNetParams)
	if err != nil {
		return nil, fmt.Errorf("unable to decode payment request: %v",
			err)
	}

	var descHash []byte
	if decoded.DescriptionHash != nil {
		descHash = decoded.DescriptionHash[:]
	}

	fallbackAddr := ""
	if decoded.FallbackAddr != nil {
		fallbackAddr = decoded.FallbackAddr.String()
	}

	settleDate := int64(0)
	if !invoice.SettleDate.IsZero() {
		settleDate = invoice.SettleDate.Unix()
	}

	// Convert between the `lnrpc` and `routing` types.
	routeHints := CreateRPCRouteHints(decoded.RouteHints)

	preimage := invoice.Terms.PaymentPreimage
	satAmt := invoice.Terms.Value.ToSatoshis()
	satAmtPaid := invoice.AmtPaid.ToSatoshis()

	isSettled := invoice.Terms.State == channeldb.ContractSettled

	var state lnrpc.Invoice_InvoiceState
	switch invoice.Terms.State {
	case channeldb.ContractOpen:
		state = lnrpc.Invoice_OPEN
	case channeldb.ContractSettled:
		state = lnrpc.Invoice_SETTLED
	case channeldb.ContractCanceled:
		state = lnrpc.Invoice_CANCELED
	case channeldb.ContractAccepted:
		state = lnrpc.Invoice_ACCEPTED
	default:
		return nil, fmt.Errorf("unknown invoice state %v",
			invoice.Terms.State)
	}

	rpcHtlcs := make([]*lnrpc.InvoiceHTLC, 0, len(invoice.Htlcs))
	for key, htlc := range invoice.Htlcs {
		var state lnrpc.InvoiceHTLCState
		switch htlc.State {
		case channeldb.HtlcStateAccepted:
			state = lnrpc.InvoiceHTLCState_ACCEPTED
		case channeldb.HtlcStateSettled:
			state = lnrpc.InvoiceHTLCState_SETTLED
		case channeldb.HtlcStateCanceled:
			state = lnrpc.InvoiceHTLCState_CANCELED
		default:
			return nil, fmt.Errorf("unknown state %v", htlc.State)
		}

		rpcHtlc := lnrpc.InvoiceHTLC{
			ChanId:       key.ChanID.ToUint64(),
			HtlcIndex:    key.HtlcID,
			AcceptHeight: int32(htlc.AcceptHeight),
			AcceptTime:   htlc.AcceptTime.Unix(),
			ExpiryHeight: int32(htlc.Expiry),
			AmtMsat:      uint64(htlc.Amt),
			State:        state,
		}

		// Only report resolved times if htlc is resolved.
		if htlc.State != channeldb.HtlcStateAccepted {
			rpcHtlc.ResolveTime = htlc.ResolveTime.Unix()
		}

		rpcHtlcs = append(rpcHtlcs, &rpcHtlc)
	}

	rpcInvoice := &lnrpc.Invoice{
		Memo:            string(invoice.Memo[:]),
		Receipt:         invoice.Receipt[:],
		RHash:           decoded.PaymentHash[:],
		Value:           int64(satAmt),
		CreationDate:    invoice.CreationDate.Unix(),
		SettleDate:      settleDate,
		Settled:         isSettled,
		PaymentRequest:  paymentRequest,
		DescriptionHash: descHash,
		Expiry:          int64(invoice.Expiry.Seconds()),
		CltvExpiry:      uint64(invoice.FinalCltvDelta),
		FallbackAddr:    fallbackAddr,
		RouteHints:      routeHints,
		AddIndex:        invoice.AddIndex,
		Private:         len(routeHints) > 0,
		SettleIndex:     invoice.SettleIndex,
		AmtPaidSat:      int64(satAmtPaid),
		AmtPaidMsat:     int64(invoice.AmtPaid),
		AmtPaid:         int64(invoice.AmtPaid),
		State:           state,
		Htlcs:           rpcHtlcs,
	}

	if preimage != channeldb.UnknownPreimage {
		rpcInvoice.RPreimage = preimage[:]
	}

	return rpcInvoice, nil
}

// CreateRPCRouteHints takes in the decoded form of an invoice's route hints
// and converts them into the lnrpc type.
func CreateRPCRouteHints(routeHints [][]zpay32.HopHint) []*lnrpc.RouteHint {
	var res []*lnrpc.RouteHint

	for _, route := range routeHints {
		hopHints := make([]*lnrpc.HopHint, 0, len(route))
		for _, hop := range route {
			pubKey := hex.EncodeToString(
				hop.NodeID.SerializeCompressed(),
			)

			hint := &lnrpc.HopHint{
				NodeId:                    pubKey,
				ChanId:                    hop.ChannelID,
				FeeBaseMsat:               hop.FeeBaseMSat,
				FeeProportionalMillionths: hop.FeeProportionalMillionths,
				CltvExpiryDelta:           uint32(hop.CLTVExpiryDelta),
			}

			hopHints = append(hopHints, hint)
		}

		routeHint := &lnrpc.RouteHint{HopHints: hopHints}
		res = append(res, routeHint)
	}

	return res
}

// CreateZpay32HopHints takes in the lnrpc form of route hints and converts them
// into an invoice decoded form.
func CreateZpay32HopHints(routeHints []*lnrpc.RouteHint) ([][]zpay32.HopHint, error) {
	var res [][]zpay32.HopHint
	for _, route := range routeHints {
		hopHints := make([]zpay32.HopHint, 0, len(route.HopHints))
		for _, hop := range route.HopHints {
			pubKeyBytes, err := hex.DecodeString(hop.NodeId)
			if err != nil {
				return nil, err
			}
			p, err := btcec.ParsePubKey(pubKeyBytes, btcec.S256())
			if err != nil {
				return nil, err
			}
			hopHints = append(hopHints, zpay32.HopHint{
				NodeID:                    p,
				ChannelID:                 hop.ChanId,
				FeeBaseMSat:               hop.FeeBaseMsat,
				FeeProportionalMillionths: hop.FeeProportionalMillionths,
				CLTVExpiryDelta:           uint16(hop.CltvExpiryDelta),
			})
		}
		res = append(res, hopHints)
	}
	return res, nil
}
