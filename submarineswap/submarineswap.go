package submarineswap

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
	"github.com/lightningnetwork/lnd/channeldb"
	"github.com/lightningnetwork/lnd/input"
	"github.com/lightningnetwork/lnd/kvdb"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/lightningnetwork/lnd/lnwallet/btcwallet"
	"github.com/lightningnetwork/lnd/lnwallet/chainfee"
)

const (
	defaultLockHeight      = 288
	redeemWitnessInputSize = 1 + 1 + 73 + 1 + 32 + 1 + 100
	refundWitnessInputSize = 1 + 1 + 73 + 1 + 0 + 1 + 100
)

var (
	submarineBucket      = []byte("submarineTransactions")
	wtxmgrNamespaceKey   = []byte("wtxmgr")
	waddrmgrNamespaceKey = []byte("waddrmgr")
)

func genSubmarineSwapScript(swapperPubKey, payerPubKey, hash []byte, lockHeight int64) ([]byte, error) {
	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_HASH160)
	builder.AddData(input.Ripemd160H(hash))
	builder.AddOp(txscript.OP_EQUAL) // Leaves 0P1 (true) on the stack if preimage matches
	builder.AddOp(txscript.OP_IF)
	builder.AddData(swapperPubKey) // Path taken if preimage matches
	builder.AddOp(txscript.OP_ELSE)
	builder.AddInt64(lockHeight)
	builder.AddOp(txscript.OP_CHECKSEQUENCEVERIFY)
	builder.AddOp(txscript.OP_DROP)
	builder.AddData(payerPubKey) // Refund back to payer
	builder.AddOp(txscript.OP_ENDIF)
	builder.AddOp(txscript.OP_CHECKSIG)

	return builder.Script()
}

func saveSubmarineData(c *channeldb.ChannelStateDB, netID byte, address btcutil.Address, creationHeight, lockHeight int64, preimage, key, swapperPubKey, script []byte) error {

	if len(preimage) != 32 {
		return errors.New("preimage not valid")
	}
	if len(key) != btcec.PrivKeyBytesLen {
		return errors.New("key not valid")
	}
	if len(swapperPubKey) != btcec.PubKeyBytesLenCompressed {
		return errors.New("swapperPubKey not valid")
	}

	return kvdb.Update(c.GetParentDB(), func(tx walletdb.ReadWriteTx) error {
		bucket, err := tx.CreateTopLevelBucket(submarineBucket)
		if err != nil {
			return err
		}

		var submarineData bytes.Buffer
		err = submarineData.WriteByte(netID)
		if err != nil {
			return err
		}
		b := make([]byte, 16)
		binary.BigEndian.PutUint64(b[0:], uint64(creationHeight))
		binary.BigEndian.PutUint64(b[8:], uint64(lockHeight))
		_, err = submarineData.Write(b)
		if err != nil {
			return err
		}
		_, err = submarineData.Write(preimage)
		if err != nil {
			return err
		}
		_, err = submarineData.Write(key)
		if err != nil {
			return err
		}
		_, err = submarineData.Write(swapperPubKey)
		if err != nil {
			return err
		}
		_, err = submarineData.Write(script)
		if err != nil {
			return err
		}

		return bucket.Put([]byte("swapClient:"+address.String()), submarineData.Bytes())
	}, func() {})
}

func getSubmarineData(c *channeldb.ChannelStateDB, netID byte, address btcutil.Address) (creationHeight, lockHeight int64, preimage, key, swapperPubKey, script []byte, err error) {
	err = kvdb.View(c.GetParentDB(), func(tx walletdb.ReadTx) error {

		bucket := tx.ReadBucket(submarineBucket)
		if bucket == nil {
			return errors.New("Not found")
		}

		value := bucket.Get([]byte("swapClient:" + address.String()))
		if value == nil {
			return errors.New("Not found")
		}

		submarineData := bytes.NewBuffer(value)
		savedNetID, err := submarineData.ReadByte()
		if err != nil {
			return err
		}
		if savedNetID != netID {
			return errors.New("Not the same network")
		}
		b := make([]byte, 16)
		_, err = submarineData.Read(b)
		if err != nil {
			return err
		}
		creationHeight = int64(binary.BigEndian.Uint64(b[0:]))
		lockHeight = int64(binary.BigEndian.Uint64(b[8:]))

		preimage = make([]byte, 32)
		_, err = submarineData.Read(preimage)
		if err != nil {
			return err
		}
		key = make([]byte, btcec.PrivKeyBytesLen)
		_, err = submarineData.Read(key)
		if err != nil {
			return err
		}
		swapperPubKey = make([]byte, btcec.PubKeyBytesLenCompressed)
		_, err = submarineData.Read(swapperPubKey)
		if err != nil {
			return err
		}

		script = make([]byte, submarineData.Len())
		_, err = submarineData.Read(script)
		if err != nil {
			return err
		}

		return nil
	}, func() {})
	return
}

func saveSwapperSubmarineData(c *channeldb.ChannelStateDB, netID byte, hash []byte, creationHeight, lockHeight int64, swapperKey []byte, script []byte) error {

	/**
	key: swapper:<hash>
	value:
		[0]: netID
		[1:9]: creationHeight
		[9:17]: lockHeight
		[17:17+btcec.PrivKeyBytesLen]: swapperKey
		[17+btcec.PrivKeyBytesLen:]: script
	*/

	if len(swapperKey) != btcec.PrivKeyBytesLen {
		return errors.New("swapperKey not valid")
	}

	return kvdb.Update(c.GetParentDB(), func(tx walletdb.ReadWriteTx) error {
		bucket, err := tx.CreateTopLevelBucket(submarineBucket)
		if err != nil {
			return err
		}

		var key bytes.Buffer
		_, err = key.WriteString("swapper:")
		if err != nil {
			return err
		}
		_, err = key.Write(hash)
		if err != nil {
			return err
		}

		var submarineData bytes.Buffer
		err = submarineData.WriteByte(netID)
		if err != nil {
			return err
		}
		b := make([]byte, 16)
		binary.BigEndian.PutUint64(b[0:], uint64(creationHeight))
		binary.BigEndian.PutUint64(b[8:], uint64(lockHeight))
		_, err = submarineData.Write(b)
		if err != nil {
			return err
		}
		_, err = submarineData.Write(swapperKey)
		if err != nil {
			return err
		}
		_, err = submarineData.Write(script)
		if err != nil {
			return err
		}

		return bucket.Put(key.Bytes(), submarineData.Bytes())
	}, func() {})
}

func getSwapperSubmarineData(c *channeldb.ChannelStateDB, netID byte, hash []byte) (creationHeight, lockHeight int64, swapperKey, script []byte, err error) {

	err = kvdb.View(c.GetParentDB(), func(tx walletdb.ReadTx) error {

		bucket := tx.ReadBucket(submarineBucket)
		if bucket == nil {
			return errors.New("Not found")
		}

		var key bytes.Buffer
		_, err = key.WriteString("swapper:")
		if err != nil {
			return err
		}
		_, err = key.Write(hash)
		if err != nil {
			return err
		}

		value := bucket.Get(key.Bytes())
		if value == nil {
			return errors.New("Not found")
		}

		submarineData := bytes.NewBuffer(value)
		savedNetID, err := submarineData.ReadByte()
		if err != nil {
			return err
		}
		if savedNetID != netID {
			return errors.New("Not the same network")
		}
		b := make([]byte, 16)
		_, err = submarineData.Read(b)
		if err != nil {
			return err
		}
		creationHeight = int64(binary.BigEndian.Uint64(b[0:]))
		lockHeight = int64(binary.BigEndian.Uint64(b[8:]))

		swapperKey = make([]byte, btcec.PrivKeyBytesLen)
		_, err = submarineData.Read(swapperKey)
		if err != nil {
			return err
		}

		script = make([]byte, submarineData.Len())
		_, err = submarineData.Read(script)
		if err != nil {
			return err
		}

		return nil
	}, func() {})

	return
}

func newAddressWitnessScriptHash(script []byte, net *chaincfg.Params) (*btcutil.AddressWitnessScriptHash, error) {
	witnessProg := sha256.Sum256(script)
	return btcutil.NewAddressWitnessScriptHash(witnessProg[:], net)
}

// AddressFromHash
func AddressFromHash(net *chaincfg.Params, c *channeldb.ChannelStateDB, hash []byte) (address btcutil.Address, creationHeight, lockHeight int64, err error) {
	var script []byte
	creationHeight, lockHeight, _, script, err = getSwapperSubmarineData(c, net.ScriptHashAddrID, hash)
	if err != nil {
		return
	}
	address, err = newAddressWitnessScriptHash(script, net)
	return
}

// CreationHeight
func CreationHeight(net *chaincfg.Params, c *channeldb.ChannelStateDB, address btcutil.Address) (creationHeight, lockHeight int64, err error) {
	creationHeight, lockHeight, _, _, _, _, err = getSubmarineData(c, net.ScriptHashAddrID, address)
	return
}

func SubmarineSwapInit() (preimage, hash, key, pubKey []byte, err error) {
	preimage = make([]byte, 32)
	rand.Read(preimage)
	h := sha256.Sum256(preimage)
	hash = h[:]

	k, err := btcec.NewPrivateKey()
	if err != nil {
		return
	}
	key = k.Serialize()
	pubKey = k.PubKey().SerializeCompressed()
	return
}

func NewSubmarineSwap(wdb walletdb.DB, manager *waddrmgr.Manager, net *chaincfg.Params,
	chainClient chain.Interface, c *channeldb.ChannelStateDB, pubKey, hash []byte) (address btcutil.Address, script, swapperPubKey []byte, lockHeight int64, err error) {

	if len(pubKey) != btcec.PubKeyBytesLenCompressed {
		err = errors.New("pubKey not valid")
		return
	}

	if len(hash) != 32 {
		err = errors.New("hash not valid")
		return
	}

	//Need to check that the hash doesn't already exists in our db
	_, _, _, _, errGet := getSwapperSubmarineData(c, net.ScriptHashAddrID, hash)
	if errGet == nil {
		err = errors.New("Hash already exists")
		return
	}

	//Create swapperKey and swapperPubKey
	key, err := btcec.NewPrivateKey()
	if err != nil {
		return
	}
	swapperKey := key.Serialize()
	swapperPubKey = key.PubKey().SerializeCompressed()
	lockHeight = defaultLockHeight

	//Create the script
	script, err = genSubmarineSwapScript(swapperPubKey, pubKey, hash, defaultLockHeight)
	if err != nil {
		return
	}

	currentHash, currentHeight, err := chainClient.GetBestBlock()
	if err != nil {
		return
	}

	address, err = importScript(
		wdb,
		manager,
		net,
		currentHeight,
		*currentHash,
		script,
	)
	if err != nil {
		return
	}
	//Watch the new address
	err = chainClient.NotifyReceived([]btcutil.Address{address})
	if err != nil {
		return
	}
	//Need to save the data keyed by hash
	err = saveSwapperSubmarineData(c, net.ScriptHashAddrID, hash, int64(currentHeight), lockHeight, swapperKey, script)

	return
}

func WatchSubmarineSwap(wdb walletdb.DB, manager *waddrmgr.Manager, net *chaincfg.Params,
	chainClient chain.Interface, c *channeldb.ChannelStateDB,
	preimage, key, swapperPubKey []byte, lockHeight int64) (address btcutil.Address, script []byte, err error) {

	currentHash, currentHeight, err := chainClient.GetBestBlock()
	if err != nil {
		return
	}

	_, pu := btcec.PrivKeyFromBytes(key)
	hash := sha256.Sum256(preimage)
	//Create the script
	script, err = genSubmarineSwapScript(swapperPubKey, pu.SerializeCompressed(), hash[:], lockHeight)
	if err != nil {
		return
	}

	address, err = importScript(
		wdb,
		manager,
		net,
		currentHeight,
		*currentHash,
		script,
	)
	if err != nil {
		return
	}
	//Watch the new address
	err = chainClient.NotifyReceived([]btcutil.Address{address})
	if err != nil {
		return
	}

	err = saveSubmarineData(c, net.ScriptHashAddrID, address, int64(currentHeight), lockHeight, preimage, key, swapperPubKey, script)
	return
}

type Utxo struct {
	Value       btcutil.Amount
	BlockHeight int32
	wire.OutPoint
}

func importScript(db walletdb.DB, manager *waddrmgr.Manager, net *chaincfg.Params, startHeight int32, startHash chainhash.Hash, script []byte) (btcutil.Address, error) {
	var p2wshAddr *btcutil.AddressWitnessScriptHash
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		bs := &waddrmgr.BlockStamp{
			Hash:   startHash,
			Height: startHeight,
		}

		// As this is a regular P2SH script, we'll import this into the
		// BIP0044 scope.
		bip44Mgr, err := manager.FetchScopedKeyManager(
			waddrmgr.KeyScopeBIP0084,
		)
		if err != nil {
			return err
		}

		addrInfo, err := bip44Mgr.ImportWitnessScript(addrmgrNs, script, bs, 0, false)
		if err != nil {
			if waddrmgr.IsError(err, waddrmgr.ErrDuplicateAddress) {
				p2wshAddr, _ = newAddressWitnessScriptHash(script, net)
				return nil
			}
			return err
		}

		p2wshAddr = addrInfo.Address().(*btcutil.AddressWitnessScriptHash)
		return nil
	})
	return p2wshAddr, err
}

// GetUtxos returns the list of utxos into a specific address from a start height
func GetUtxos(db walletdb.DB, txstore *wtxmgr.Store, net *chaincfg.Params, start int32, address string) ([]Utxo, error) {
	var txos []Utxo
	outPoints := make(map[string]struct{})
	spentOutPoints := make(map[string]struct{})
	err := walletdb.View(db, func(dbtx walletdb.ReadTx) error {
		txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)
		rangeFn := func(details []wtxmgr.TxDetails) (bool, error) {
			// TODO: probably should make RangeTransactions not reuse the
			// details backing array memory.
			dets := make([]wtxmgr.TxDetails, len(details))
			copy(dets, details)
			details = dets

			//txs := make([]TransactionSummary, 0, len(details))
			for _, d := range details {
				//txs = append(txs, makeTxSummary(dbtx, w, &details[i]))
				if d.Block.Height != -1 {
					for i, txout := range d.MsgTx.TxOut {
						_, addrs, _, err := txscript.ExtractPkScriptAddrs(txout.PkScript, net)
						if err == nil {
							if len(addrs) == 0 {
								if len(txout.PkScript) == 0 {
									log.Errorf("found script with zero addresses and zero length")
									continue
								}
								log.Warnf("found script with zero addresses %v", hex.EncodeToString(txout.PkScript))
								dis, err := txscript.DisasmString(txout.PkScript)
								if err != nil {
									log.Errorf("unable to parse script")
								} else {
									log.Infof("parsed script: %v", dis)
								}

								continue
							}
							if addrs[0].String() == address {
								h := d.MsgTx.TxHash()
								op := wire.NewOutPoint(&h, uint32(i))
								txos = append(txos, Utxo{
									Value:       btcutil.Amount(txout.Value),
									BlockHeight: d.Block.Height,
									OutPoint:    *op,
								})
								outPoints[op.String()] = struct{}{}
								//return true, nil
							}
						}
					}
					for _, txin := range d.MsgTx.TxIn {
						if _, ok := outPoints[txin.PreviousOutPoint.String()]; ok {
							spentOutPoints[txin.PreviousOutPoint.String()] = struct{}{}
						}
					}
				}
			}
			return false, nil
		}

		return txstore.RangeTransactions(txmgrNs, start, int32(^uint32(0)>>1), rangeFn)
	})
	if err != nil {
		return nil, err
	}
	var utxos []Utxo
	for _, txo := range txos {
		if _, ok := spentOutPoints[txo.OutPoint.String()]; !ok {
			utxos = append(utxos, txo)
		}
	}
	return utxos, nil
}

func RedeemFees(c *channeldb.ChannelStateDB, net *chaincfg.Params, wallet *lnwallet.LightningWallet, hash []byte, feePerKw chainfee.SatPerKWeight) (btcutil.Amount, error) {
	creationHeight, _, _, script, err := getSwapperSubmarineData(c, net.ScriptHashAddrID, hash[:])
	if err != nil {
		return 0, err
	}
	address, err := newAddressWitnessScriptHash(script, net)
	if err != nil {
		return 0, err
	}
	w := wallet.WalletController.(*btcwallet.BtcWallet).InternalWallet()
	utxos, err := GetUtxos(w.Database(), w.TxStore, net, int32(creationHeight), address.String())
	if err != nil {
		return 0, err
	}
	if len(utxos) == 0 {
		return 0, errors.New("no utxo")
	}

	redeemTx := wire.NewMsgTx(1)

	// Add the inputs without the witness and calculate the amount to redeem
	var amount btcutil.Amount
	for _, utxo := range utxos {
		amount += utxo.Value
		txIn := wire.NewTxIn(&utxo.OutPoint, nil, nil)
		txIn.Sequence = 0
		redeemTx.AddTxIn(txIn)
	}

	//Generate a random address
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		return 0, err
	}
	redeemAddress, err := btcutil.NewAddressPubKey(privateKey.PubKey().SerializeCompressed(), net)
	if err != nil {
		return 0, err
	}
	// Add the single output
	redeemScript, err := txscript.PayToAddrScript(redeemAddress)
	if err != nil {
		return 0, err
	}
	txOut := wire.TxOut{PkScript: redeemScript}
	redeemTx.AddTxOut(&txOut)

	_, currentHeight, err := w.ChainClient().GetBestBlock()
	if err != nil {
		return 0, err
	}
	redeemTx.LockTime = uint32(currentHeight)

	// Calcluate the weight and the fee
	weight := 4*redeemTx.SerializeSizeStripped() + redeemWitnessInputSize*len(redeemTx.TxIn)
	// Adjust the amount in the txout
	return feePerKw.FeeForWeight(int64(weight)), nil
}

// Redeem
func Redeem(c *channeldb.ChannelStateDB, net *chaincfg.Params, wallet *lnwallet.LightningWallet, preimage []byte, redeemAddress btcutil.Address, feePerKw chainfee.SatPerKWeight) (*wire.MsgTx, error) {

	hash := sha256.Sum256(preimage)
	creationHeight, _, serviceKey, script, err := getSwapperSubmarineData(c, net.ScriptHashAddrID, hash[:])
	if err != nil {
		return nil, err
	}
	address, err := newAddressWitnessScriptHash(script, net)
	if err != nil {
		return nil, err
	}
	w := wallet.WalletController.(*btcwallet.BtcWallet).InternalWallet()
	utxos, err := GetUtxos(w.Database(), w.TxStore, net, int32(creationHeight), address.String())
	if err != nil {
		return nil, err
	}
	if len(utxos) == 0 {
		return nil, errors.New("no utxo")
	}

	redeemTx := wire.NewMsgTx(1)

	// Add the inputs without the witness and calculate the amount to redeem
	var amount btcutil.Amount
	for _, utxo := range utxos {
		amount += utxo.Value
		txIn := wire.NewTxIn(&utxo.OutPoint, nil, nil)
		txIn.Sequence = 0
		redeemTx.AddTxIn(txIn)
	}

	// Add the single output
	redeemScript, err := txscript.PayToAddrScript(redeemAddress)
	if err != nil {
		return nil, err
	}
	txOut := wire.TxOut{PkScript: redeemScript}
	redeemTx.AddTxOut(&txOut)

	_, currentHeight, err := w.ChainClient().GetBestBlock()
	if err != nil {
		return nil, err
	}
	redeemTx.LockTime = uint32(currentHeight)

	// Calcluate the weight and the fee
	weight := 4*redeemTx.SerializeSizeStripped() + redeemWitnessInputSize*len(redeemTx.TxIn)
	// Adjust the amount in the txout
	redeemTx.TxOut[0].Value = int64(amount - feePerKw.FeeForWeight(int64(weight)))

	sigHashes := input.NewTxSigHashesV0Only(redeemTx)
	privateKey, _ := btcec.PrivKeyFromBytes(serviceKey)
	for idx := range redeemTx.TxIn {
		scriptSig, err := txscript.RawTxInWitnessSignature(redeemTx, sigHashes, idx, int64(utxos[idx].Value), script, txscript.SigHashAll, privateKey)
		if err != nil {
			return nil, err
		}
		redeemTx.TxIn[idx].Witness = [][]byte{scriptSig, preimage, script}
	}

	err = wallet.PublishTransaction(redeemTx, "SubSwap Redeem")
	if err != nil {
		return nil, err
	}

	return redeemTx, nil
}

// RefundTx
func RefundTx(c *channeldb.ChannelStateDB, net *chaincfg.Params, wallet *lnwallet.LightningWallet, address, refundAddress btcutil.Address, feePerKw chainfee.SatPerKWeight) (*wire.MsgTx, btcutil.Amount, error) {

	creationHeight, lockHeight, _, clientKey, _, script, err := getSubmarineData(c, net.ScriptHashAddrID, address)
	if err != nil {
		return nil, 0, err
	}

	w := wallet.WalletController.(*btcwallet.BtcWallet).InternalWallet()
	utxos, err := GetUtxos(w.Database(), w.TxStore, net, int32(creationHeight), address.String())
	if err != nil {
		return nil, 0, err
	}
	if len(utxos) == 0 {
		return nil, 0, errors.New("no utxo")
	}

	refundTx := wire.NewMsgTx(2)

	// Add the inputs without the witness and calculate the amount to redeem
	var amount btcutil.Amount
	for _, utxo := range utxos {
		amount += utxo.Value
		txIn := wire.NewTxIn(&utxo.OutPoint, nil, nil)
		txIn.Sequence = uint32(lockHeight)
		refundTx.AddTxIn(txIn)
	}

	// Add the single output
	refundScript, err := txscript.PayToAddrScript(refundAddress)
	if err != nil {
		return nil, 0, err
	}
	txOut := wire.TxOut{PkScript: refundScript}
	refundTx.AddTxOut(&txOut)

	_, currentHeight, err := w.ChainClient().GetBestBlock()
	if err != nil {
		return nil, 0, err
	}
	lockTime := uint32(utxos[len(utxos)-1].BlockHeight) + uint32(lockHeight)

	if lockTime < uint32(currentHeight) {
		lockTime = uint32(currentHeight)
	}
	refundTx.LockTime = lockTime

	// Calcluate the weight and the fee
	weight := 4*refundTx.SerializeSizeStripped() + refundWitnessInputSize*len(refundTx.TxIn)
	fees := feePerKw.FeeForWeight(int64(weight))
	// Adjust the amount in the txout
	refundTx.TxOut[0].Value = int64(amount - fees)

	err = txrules.CheckOutput(
		refundTx.TxOut[0], txrules.DefaultRelayFeePerKb,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("fees are to high for the given amount %w", err)
	}

	sigHashes := input.NewTxSigHashesV0Only(refundTx)
	privateKey, _ := btcec.PrivKeyFromBytes(clientKey)
	for idx := range refundTx.TxIn {
		scriptSig, err := txscript.RawTxInWitnessSignature(refundTx, sigHashes, idx, int64(utxos[idx].Value), script, txscript.SigHashAll, privateKey)
		if err != nil {
			return nil, 0, err
		}
		refundTx.TxIn[idx].Witness = [][]byte{scriptSig, nil, script}
	}

	return refundTx, fees, nil
}
