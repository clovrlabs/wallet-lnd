//go:build !rpctest
// +build !rpctest

package lncfg

// ProtocolOptions is a struct that we use to be able to test backwards
// compatibility of protocol additions, while defaulting to the latest within
// lnd, or to enable experimental protocol changes.
type ProtocolOptions struct {
	// LegacyProtocol is a sub-config that houses all the legacy protocol
	// options.  These are mostly used for integration tests as most modern
	// nodes should always run with them on by default.
	LegacyProtocol `group:"legacy" namespace:"legacy"`

	// ExperimentalProtocol is a sub-config that houses any experimental
	// protocol features that also require a build-tag to activate.
	ExperimentalProtocol

	// WumboChans should be set if we want to enable support for wumbo
	// (channels larger than 0.16 BTC) channels, which is the opposite of
	// mini.
	WumboChans bool `long:"wumbo-channels" description:"if set, then lnd will create and accept requests for channels larger chan 0.16 BTC"`

	// NoAnchors should be set if we don't want to support opening or accepting
	// channels having the anchor commitment type.
	NoAnchors bool `long:"no-anchors" description:"disable support for anchor commitments"`

	// NoScriptEnforcedLease should be set if we don't want to support
	// opening or accepting channels having the script enforced commitment
	// type for leased channel.
	NoScriptEnforcedLease bool `long:"no-script-enforced-lease" description:"disable support for script enforced lease commitments"`

	// ZeroConfChans should be set if we want to enable support for zero conf
	// channels (channels that become operational before funding transaction
	// was confirmed).
	ZeroConfChans bool `long:"zero-conf-channels" description:"if set, then lnd will accept zero as minimum_depth in accept_channel message"`
}

// Wumbo returns true if lnd should permit the creation and acceptance of wumbo
// channels.
func (l *ProtocolOptions) Wumbo() bool {
	return l.WumboChans
}

// NoAnchorCommitments returns true if we have disabled support for the anchor
// commitment type.
func (l *ProtocolOptions) NoAnchorCommitments() bool {
	return l.NoAnchors
}

// NoScriptEnforcementLease returns true if we have disabled support for the
// script enforcement commitment type for leased channels.
func (l *ProtocolOptions) NoScriptEnforcementLease() bool {
	return l.NoScriptEnforcedLease
}

// ZeroConf returns true if lnd accepts zero confs channels.
// channels.
func (l *ProtocolOptions) ZeroConf() bool {
	return l.ZeroConfChans
}
