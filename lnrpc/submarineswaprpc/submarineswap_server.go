// +build submarineswaprpc

package submarineswaprpc

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/lightningnetwork/lnd/lnwallet/btcwallet"
	"github.com/lightningnetwork/lnd/submarineswap"
	"github.com/lightningnetwork/lnd/sweep"
	"google.golang.org/grpc"
	"gopkg.in/macaroon-bakery.v2/bakery"
)

const (
	// subServerName is the name of the RPC sub-server. We'll use this name
	// to register ourselves, and we also require that the main
	// SubServerConfigDispatcher instance recognize this as the name of the
	// config file that we need.
	subServerName = "SubmarineSwapRPC"
)

var (
	// macaroonOps are the set of capabilities that our minted macaroon (if
	// it doesn't already exist) will have.
	macaroonOps = []bakery.Op{
		{
			Entity: "onchain",
			Action: "read",
		},
		{
			Entity: "onchain",
			Action: "write",
		},
		{
			Entity: "offchain",
			Action: "write",
		},
	}

	// macPermissions maps RPC calls to the permissions they require.
	macPermissions = map[string][]bakery.Op{
		"/submarineswaprpc.SubmarineSwapper/SubSwapClientInit": {{
			Entity: "offchain",
			Action: "write",
		}},
		"/submarineswaprpc.SubmarineSwapper/SubSwapServiceInit": {{
			Entity: "offchain",
			Action: "write",
		}},
		"/submarineswaprpc.SubmarineSwapper/UnspentAmount": {{
			Entity: "onchain",
			Action: "read",
		}},
		"/submarineswaprpc.SubmarineSwapper/SubSwapServiceRedeem": {{
			Entity: "onchain",
			Action: "write",
		}},
		"/submarineswaprpc.SubmarineSwapper/SubSwapClientRefund": {{
			Entity: "onchain",
			Action: "write",
		}},
		"/submarineswaprpc.SubmarineSwapper/SubSwapClientWatch": {{
			Entity: "onchain",
			Action: "read",
		}},
	}

	// DefaultSubmarineSwapperMacFilename is the default name of the submarine
	// swapper macaroon that we expect to find via a file handle within the
	// main configuration file in this package.
	DefaultSubmarineSwapperMacFilename = "submarineswap.macaroon"
)

// fileExists reports whether the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// Server is a sub-server of the main RPC server.
type Server struct {
	started uint32
	stopped uint32

	cfg Config
}

// New returns a new instance of the submarineswaprpc SubmarineSwapper
// sub-server. We also return the set of permissions for the macaroons
// that we may create within this method. If the macaroons we need aren't
// found in the filepath, then we'll create them on start up.
// If we're unable to locate, or create the macaroons we need, then we'll
// return with an error.
func New(cfg *Config) (*Server, lnrpc.MacaroonPerms, error) {
	// If the path of the submarine swapper macaroon wasn't generated, then
	// we'll assume that it's found at the default network directory.
	if cfg.SubmarineSwapMacPath == "" {
		cfg.SubmarineSwapMacPath = filepath.Join(
			cfg.NetworkDir, DefaultSubmarineSwapperMacFilename,
		)
	}

	// Now that we know the full path of the submarine swapper macaroon, we can
	// check to see if we need to create it or not.
	macFilePath := cfg.SubmarineSwapMacPath
	if cfg.MacService != nil && !fileExists(macFilePath) {
		log.Infof("Baking macaroons for SubmarineSwapper RPC Server at: %v",
			macFilePath)

		// At this point, we know that the submarine swapper macaroon
		// doesn't yet, exist, so we need to create it with the help of
		// the main macaroon service.
		submarineSwapMac, err := cfg.MacService.Oven.NewMacaroon(
			context.Background(), bakery.LatestVersion, nil,
			macaroonOps...,
		)
		if err != nil {
			return nil, nil, err
		}
		submarineSwapMacBytes, err := submarineSwapMac.M().MarshalBinary()
		if err != nil {
			return nil, nil, err
		}
		err = ioutil.WriteFile(macFilePath, submarineSwapMacBytes, 0644)
		if err != nil {
			os.Remove(macFilePath)
			return nil, nil, err
		}
	}

	return &Server{
		cfg: *cfg,
	}, macPermissions, nil
}

// Compile-time checks to ensure that Server fully implements the
// SubmarineSwapperServer gRPC service and lnrpc.SubServer interface.
var _ SubmarineSwapperServer = (*Server)(nil)
var _ lnrpc.SubServer = (*Server)(nil)

// Start launches any helper goroutines required for the server to function.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) Start() error {
	if !atomic.CompareAndSwapUint32(&s.started, 0, 1) {
		return nil
	}

	return nil
}

// Stop signals any active goroutines for a graceful closure.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) Stop() error {
	if !atomic.CompareAndSwapUint32(&s.stopped, 0, 1) {
		return nil
	}
	return nil
}

// Name returns a unique string representation of the sub-server. This can be
// used to identify the sub-server and also de-duplicate them.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) Name() string {
	return subServerName
}

// RegisterWithRootServer will be called by the root gRPC server to direct a RPC
// sub-server to register itself with the main gRPC root server. Until this is
// called, each sub-server won't be able to have requests routed towards it.
//
// NOTE: This is part of the lnrpc.SubServer interface.
func (s *Server) RegisterWithRootServer(grpcServer *grpc.Server) error {
	// We make sure that we register it with the main gRPC server to ensure
	// all our methods are routed properly.
	RegisterSubmarineSwapperServer(grpcServer, s)

	log.Debug("SubmarineSwapper RPC server successfully register with root " +
		"gRPC server")

	return nil
}

// SubSwapClientInit
func (s *Server) SubSwapClientInit(ctx context.Context,
	in *SubSwapClientInitRequest) (*SubSwapClientInitResponse, error) {

	preimage, hash, key, pubKey, err := submarineswap.SubmarineSwapInit()

	log.Infof("[SubSwapClientInit] Preimage=%x, Hash=%x, Key=%x, Pubkey=%x", preimage, hash, key, pubKey)

	return &SubSwapClientInitResponse{
		Preimage: preimage,
		Hash:     hash,
		Key:      key,
		Pubkey:   pubKey,
	}, err
}

// SubSwapServiceInit
func (s *Server) SubSwapServiceInit(ctx context.Context,
	in *SubSwapServiceInitRequest) (*SubSwapServiceInitResponse, error) {
	b := s.cfg.Wallet.WalletController.(*btcwallet.BtcWallet).InternalWallet()
	//Create a new submarine address and associated script
	addr, script, swapServicePubKey, lockHeight, err := submarineswap.NewSubmarineSwap(
		b.Database(),
		b.Manager,
		s.cfg.ActiveNetParams,
		b.ChainClient(),
		s.cfg.Wallet.Cfg.Database,
		in.Pubkey,
		in.Hash,
	)
	if err != nil {
		return nil, err
	}
	log.Infof("[SubSwapServiceInit] addr=%v script=%x pubkey=%x", addr.String(), script, swapServicePubKey)
	return &SubSwapServiceInitResponse{Address: addr.String(), Pubkey: swapServicePubKey, LockHeight: lockHeight}, nil
}

// WatchSubmarineSwap
func (s *Server) SubSwapClientWatch(ctx context.Context,
	in *SubSwapClientWatchRequest) (*SubSwapClientWatchResponse, error) {
	b := s.cfg.Wallet.WalletController.(*btcwallet.BtcWallet).InternalWallet()
	address, script, err := submarineswap.WatchSubmarineSwap(
		b.Database(),
		b.Manager,
		s.cfg.ActiveNetParams,
		b.ChainClient(),
		s.cfg.Wallet.Cfg.Database,
		in.Preimage,
		in.Key,
		in.ServicePubkey,
		in.LockHeight,
	)
	if err != nil {
		return nil, err
	}
	return &SubSwapClientWatchResponse{
		Address: address.String(),
		Script:  script,
	}, nil
}

// UnspentAmount returns the total amount of the btc received in a watched address
// and the height of the first transaction sending btc to the address.
func (s *Server) UnspentAmount(ctx context.Context,
	in *UnspentAmountRequest) (*UnspentAmountResponse, error) {
	b := s.cfg.Wallet.WalletController.(*btcwallet.BtcWallet).InternalWallet()
	address := in.Address
	var start, lockHeight int32
	if len(in.Hash) > 0 {
		addr, creationHeight, lh, err := submarineswap.AddressFromHash(s.cfg.ActiveNetParams, s.cfg.Wallet.Cfg.Database, in.Hash)
		if err != nil {
			return nil, err
		}
		address = addr.String()
		start = int32(creationHeight)
		lockHeight = int32(lh)
	} else {
		addr, err := btcutil.DecodeAddress(address, s.cfg.ActiveNetParams)
		if err != nil {
			return nil, err
		}
		creationHeight, lh, err := submarineswap.CreationHeight(s.cfg.ActiveNetParams, s.cfg.Wallet.Cfg.Database, addr)
		start = int32(creationHeight)
		lockHeight = int32(lh)
	}
	utxos, err := submarineswap.GetUtxos(b.Database(), b.TxStore, s.cfg.ActiveNetParams, start, address)
	if err != nil {
		return nil, err
	}
	var totalAmount int64
	var u []*UnspentAmountResponse_Utxo
	for _, utxo := range utxos {
		u = append(u, &UnspentAmountResponse_Utxo{
			BlockHeight: utxo.BlockHeight,
			Amount:      int64(utxo.Value),
			Txid:        utxo.Hash.String(),
			Index:       utxo.Index,
		})
		totalAmount += int64(utxo.Value)
	}
	log.Infof("[UnspentAmount] address=%v, totalAmount=%v", address, totalAmount)
	return &UnspentAmountResponse{Amount: totalAmount, LockHeight: lockHeight, Utxos: u}, nil
}

func (s *Server) SubSwapServiceRedeem(ctx context.Context,
	in *SubSwapServiceRedeemRequest) (*SubSwapServiceRedeemResponse, error) {

	redeemAddress, err := s.cfg.Wallet.NewAddress(lnwallet.WitnessPubKey, false)
	if err != nil {
		return nil, err
	}

	satPerKw := lnwallet.SatPerKVByte(in.SatPerByte * 1000).FeePerKWeight()
	feePerKw, err := sweep.DetermineFeePerKw(
		s.cfg.FeeEstimator, sweep.FeePreference{
			ConfTarget: uint32(in.TargetConf),
			FeeRate:    satPerKw,
		},
	)
	if err != nil {
		return nil, err
	}

	tx, err := submarineswap.Redeem(s.cfg.Wallet.Cfg.Database,
		s.cfg.ActiveNetParams,
		s.cfg.Wallet,
		in.Preimage,
		redeemAddress,
		feePerKw,
	)

	if err != nil {
		return nil, err
	}
	log.Infof("[subswapserviceredeem] txid: %v", tx.TxHash().String())
	return &SubSwapServiceRedeemResponse{Txid: tx.TxHash().String()}, nil
}

func (s *Server) SubSwapClientRefund(ctx context.Context,
	in *SubSwapClientRefundRequest) (*SubSwapClientRefundResponse, error) {

	address, err := btcutil.DecodeAddress(in.Address, nil)
	if err != nil {
		return nil, err
	}
	refundAddress, err := btcutil.DecodeAddress(in.RefundAddress, nil)
	if err != nil {
		return nil, err
	}

	satPerKw := lnwallet.SatPerKVByte(in.SatPerByte * 1000).FeePerKWeight()
	feePerKw, err := sweep.DetermineFeePerKw(
		s.cfg.FeeEstimator, sweep.FeePreference{
			ConfTarget: uint32(in.TargetConf),
			FeeRate:    satPerKw,
		},
	)
	if err != nil {
		return nil, err
	}

	tx, err := submarineswap.Refund(s.cfg.Wallet.Cfg.Database,
		s.cfg.ActiveNetParams,
		s.cfg.Wallet,
		address,
		refundAddress,
		feePerKw,
	)

	if err != nil {
		return nil, err
	}
	log.Infof("[subswapclientrefund] txid: %v", tx.TxHash().String())
	return &SubSwapClientRefundResponse{Txid: tx.TxHash().String()}, nil
}
