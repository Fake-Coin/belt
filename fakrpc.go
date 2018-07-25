package belt

import (
	"bytes"
	"log"

	"fakco.in/fakd/chaincfg"
	"fakco.in/fakd/txscript"
	"fakco.in/fakd/wire"
)

func (app *App) onRelevantTxAccepted(transaction []byte) {
	var msg wire.MsgTx
	if err := msg.Deserialize(bytes.NewBuffer(transaction)); err != nil {
		log.Println("[onRelevantTxAccepted]", err)
		return
	}

	hash := msg.TxHash()
	for index, out := range msg.TxOut {
		scriptClass, addresses, reqSigs, err := txscript.ExtractPkScriptAddrs(out.PkScript, &chaincfg.MainNetParams)
		if err != nil {
			log.Println("[onRelevantTxAccepted]", err)
			continue
		}

		if scriptClass != txscript.PubKeyHashTy || reqSigs != 1 || len(addresses) != 1 {
			log.Println("[onRelevantTxAccepted] bad transaction")
			continue
		}

		var bet Bet
		if res := app.db.Where("watch_addr = ?", addresses[0].String()).First(&bet); res.Error != nil {
			log.Println("[onRelevantTxAccepted]", res.Error)
			continue
		}

		btx := BetTx{
			Hash:  hash.String(),
			Index: uint32(index),
			Value: FAK(out.Value),
		}

		res := app.db.Model(&bet).Association("Transactions").Append(&btx)
		if res.Error != nil {
			log.Println("[onRelevantTxAccepted]", res.Error)
			continue
		}

		if err := app.hub.notifyBet(bet.OptionID, btx); err != nil {
			log.Println("[onRelevantTxAccepted]", err)
			continue
		}
	}
}
