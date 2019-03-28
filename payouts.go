package belt

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"

	"fakco.in/fakd/btcec"
	"fakco.in/fakd/chaincfg"
	"fakco.in/fakd/chaincfg/chainhash"
	"fakco.in/fakd/txscript"
	"fakco.in/fakd/wire"
	"fakco.in/fakutil"
)

func (app *App) Payout(w http.ResponseWriter, r *http.Request) {
	belt, ok := r.Context().Value("beltctx").(*BeltCtx)
	if !ok {
		log.Println("could not find belt in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	winID := app.hub.BeltHolder()

	redeemTx, err := app.legacyPayout(belt.Belt, uint(winID))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var buf bytes.Buffer
	if err := redeemTx.Serialize(&buf); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, "Bytes:", redeemTx.SerializeSize())

	fmt.Fprintf(w, "%x\n", buf.Bytes())

	// enc := json.NewEncoder(w)
	// enc.SetIndent("", "\t")
	//
	// w.Header().Set("Content-Type", "application/json")
	// if err := enc.Encode(tx); err != nil {
	// 	log.Println(err)
	// }
}

func (app *App) legacyPayout(b *Belt, winner uint) (*wire.MsgTx, error) {
	const FeeAddr = "tPNdHXzqg9YCkgKa92ZDtmrQV4bpLRjff9"
	const oneFAK = 100000000

	var winOpt Option
	var totalVal, winVal, loseVal int64
	for _, opt := range b.Options {
		val := int64(opt.Value(1))
		totalVal += val
		if opt.ID == winner {
			winOpt = opt
			winVal += val
		} else {
			loseVal += val
		}
	}

	if winVal <= 0 {
		return nil, errors.New("there are no winners here")
	}

	var winBets []Bet
	for _, bet := range winOpt.Bets {
		if bet.Value(1) <= 0 {
			continue
		}
		winBets = append(winBets, bet)
	}

	tbaddrs, txin := b.TxIn()

	bytesIn := int64(len(txin) * 180)
	bytesOut := int64((1 + len(winBets)) * 34)
	bytesTotal := (bytesIn + bytesOut)

	feeTotal := ((bytesTotal * (oneFAK / 1000)) / 1000)
	loseVal -= feeTotal
	totalVal -= feeTotal

	subFee := big.NewRat(int64(loseVal)*99, 100)
	remainder := totalVal

	var txout []*wire.TxOut
	for _, winner := range winBets {
		wval := int64(winner.Value(1))
		r := big.NewRat(wval, winVal)
		fp, _ := new(big.Rat).Mul(r, subFee).Float64()

		newValue := wval + int64(math.Floor(fp))
		remainder -= newValue

		if 0 <= newValue {
			txout = append(txout, wire.NewTxOut(newValue, makeScript(winner.PayAddr)))
		}
	}

	txout = append(txout, wire.NewTxOut(remainder, makeScript(FeeAddr)))

	redeemTx := &wire.MsgTx{
		Version: wire.TxVersion,
		TxIn:    txin,
		TxOut:   txout,
	}

	for i := range redeemTx.TxIn {
		var err error
		redeemTx.TxIn[i].SignatureScript, err = txscript.SignTxOutput(
			&chaincfg.MainNetParams,
			redeemTx,
			i,
			makeScript(tbaddrs[i]),
			txscript.SigHashAll,
			txscript.KeyClosure(app.keyLookup),
			nil, nil)
		if err != nil {
			return nil, err
		}
	}

	return redeemTx, nil
}

func (app *App) keyLookup(a fakutil.Address) (*btcec.PrivateKey, bool, error) {
	var bet Bet
	if res := app.db.Where("watch_addr = ?", a.String()).First(&bet); res.Error != nil {
		log.Println("[app.keyLookup]", res.Error)
		return nil, false, res.Error
	}

	wif, err := fakutil.DecodeWIF(bet.WIFKey)
	if err != nil {
		log.Println("[app.keyLookup]", err)
		return nil, false, err
	}

	return wif.PrivKey, wif.CompressPubKey, nil
}

func (b Belt) TxIn() ([]string, []*wire.TxIn) {
	var addrs []string
	var inputs []*wire.TxIn
	for _, opt := range b.Options {
		for _, bet := range opt.Bets {
			for _, tx := range bet.Transactions {
				addrs = append(addrs, bet.WatchAddr)
				inputs = append(inputs, tx.WireInput())
			}
		}
	}

	return addrs, inputs
}

func (tx BetTx) WireInput() *wire.TxIn {
	var dst chainhash.Hash
	if err := chainhash.Decode(&dst, tx.Hash); err != nil {
		log.Fatal(err)
	}

	return wire.NewTxIn(
		wire.NewOutPoint(&dst, tx.Index), nil, nil)
}

func (o Option) WireOutput(amount FAK) []*wire.TxOut {
	optValue := o.Value(1)

	var outputs []*wire.TxOut
	for _, bet := range o.Bets {
		propPay := (bet.Value(1) * amount) / optValue
		outputs = append(outputs, wire.NewTxOut(int64(propPay), makeScript(bet.PayAddr)))
	}

	return outputs
}

func makeScript(a string) []byte {
	addr, err := fakutil.DecodeAddress(a, &chaincfg.MainNetParams)
	if err != nil {
		log.Fatal(err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		log.Fatal(err)
	}
	return pkScript
}
