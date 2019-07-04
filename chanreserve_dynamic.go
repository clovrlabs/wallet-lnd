// +build chainreservedynamic

package lnd

import (
	"encoding/base64"

	"github.com/btcsuite/btcutil"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

func requiredRemoteChanReserve(chainCfg *chainConfig,
	chanAmt, defaultReserve btcutil.Amount) (reserve btcutil.Amount, err error) {
	reserve = defaultReserve
	if len(chainCfg.ChanReserveWASM) == 0 {
		return
	}
	var wasmBytes []byte
	wasmBytes, err = base64.StdEncoding.DecodeString(chainCfg.ChanReserveWASM)
	if err != nil {
		return
	}

	var instance wasm.Instance
	// Instantiates the WebAssembly module.
	instance, err = wasm.NewInstance(wasmBytes)
	if err != nil {
		return
	}
	defer instance.Close()

	var r wasm.Value
	if requiredRemoteChanReserve, ok := instance.Exports["required_remote_chan_reserve"]; ok {
		r, err = requiredRemoteChanReserve(int64(chanAmt))
		if err != nil {
			return
		}
		reserve = btcutil.Amount(r.ToI64())
	}
	return
}
