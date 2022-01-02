// +build !chanreservedynamic

package lnd

import (
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lncfg"
)

func requiredRemoteChanReserve(chainCfg *lncfg.Chain,
	chanAmt, defaultReserve btcutil.Amount) (btcutil.Amount, error) {
	return defaultReserve, nil
}
