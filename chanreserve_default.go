// +build !chainreservedynamic

package lnd

import (
	"github.com/btcsuite/btcutil"
)

func requiredRemoteChanReserve(chainCfg *chainConfig,
	chanAmt, defaultReserve btcutil.Amount) (btcutil.Amount, error) {
	return defaultReserve, nil
}
