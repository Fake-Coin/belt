package belt

import (
	"bytes"
	"log"
	"time"

	"fakco.in/fakd/chaincfg"
	"fakco.in/fakd/chaincfg/chainhash"
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

		go app.watchConfirmations(hash, &btx)

		if err := app.hub.notifyBet(bet.OptionID, btx); err != nil {
			log.Println("[onRelevantTxAccepted]", err)
			continue
		}
	}
}

func (app *App) watchConfirmations(hash chainhash.Hash, tx *BetTx) {
	tick := time.NewTicker(1 * time.Minute)
	defer tick.Stop()

	lastUpdate := time.Now()
	for range tick.C {
		txInfo, err := app.rpcClient.GetRawTransactionVerbose(&hash)
		if err != nil {
			log.Printf("[app.watchConfirmations][%s] %s\n", hash, err)
			continue
		}

		conf := int(txInfo.Confirmations)

		if conf == tx.Confirmations {
			if 30*time.Minute < time.Since(lastUpdate) {
				log.Printf("[app.watchConfirmations][%s] 30m since last confirmation update\n", hash)
				return
			}
			continue
		}

		res := app.db.Model(tx).Update("confirmations", conf)
		if res.Error != nil {
			log.Printf("[app.watchConfirmations][%s] %s\n", hash, res.Error)
			continue
		}

		if tx.Confirmations == 6 {
			return
		}

		lastUpdate = time.Now()
	}
}
