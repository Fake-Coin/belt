package belt

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
)

func (app *App) SetBelt(w http.ResponseWriter, r *http.Request) {
	optionID, err := strconv.Atoi(r.PostFormValue("optionID"))
	if err != nil {
		log.Println("[GetAddr] bad option id", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if optionID == -1 {
		app.hub.SetHolder(int64(optionID))
		app.hub.sendBeltUnset()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"ok": true}`)
		return
	}

	db, ok := r.Context().Value("database").(*gorm.DB)
	if !ok {
		log.Println("could not find database in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var opt Option
	if res := db.First(&opt, uint(optionID)); res.Error != nil {
		log.Printf("[GetAddr] '%d' %s\n", optionID, res.Error)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	// TODO: validate back reference

	app.hub.SetHolder(int64(opt.ID))

	if err := app.hub.sendBeltUpdate(opt); err != nil {
		log.Println("[SetBelt][app.sendBeltUpdate]", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"ok": true}`)
}

var userID int64

func (app *App) Socket(w http.ResponseWriter, r *http.Request) {
	c, err := app.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[app.Socket] upgrade:", err)
		return
	}
	defer c.Close()

	ch := make(chan []byte, 10)
	id, err := app.hub.Subscribe(ch)
	if err != nil {
		log.Println("[app.Socket] blstr:", err)
		return
	}
	defer app.hub.Unsubscribe(id)

	{
		b, err := json.Marshal(chatMsg{Type: "beltchat", Text: "Connected to channel: 'nightattack'"})
		if err == nil {
			app.hub.hub.Send(id, b)
		}
	}

	kill := make(chan struct{})
	go func() {
		defer close(kill)
		c.ReadMessage()
	}()

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case msg := <-ch:
			if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println("[app.Socket] write error:", err)
				return
			}
		case <-tick.C:
		case <-kill:
			return
		}
	}
}
