package belt

import (
	"encoding/json"
	"log"
	"sync/atomic"

	"vallon.me/blstr"
)

type Hub struct {
	userID     *int64
	hub        *blstr.ByteHub
	beltHolder *int64

	// chatLog []*ChatMsg
}

func NewHub() *Hub {
	var no int64 = -1
	return &Hub{
		userID:     new(int64),
		hub:        blstr.New(),
		beltHolder: &no,
	}
}

func (h *Hub) BeltHolder() int64 {
	return atomic.LoadInt64(h.beltHolder)
}

func (h *Hub) SetHolder(id int64) {
	atomic.SwapInt64(h.beltHolder, id)
}

func (h *Hub) Subscribe(ch chan []byte) (int, error) {
	id := int(atomic.AddInt64(h.userID, 1))
	if err := h.hub.Subscribe(id, ch); err != nil {
		return 0, err
	}
	return id, nil
}

func (h *Hub) Unsubscribe(id int) {
	h.hub.Unsubscribe(id)
}

type pingMsg struct {
	Type  string `json:"type"`
	Count uint64 `json:"count"`
}

var pingCount uint64

func (h *Hub) Ping() {
	b, err := json.Marshal(pingMsg{"ping", atomic.AddUint64(&pingCount, 1)})
	if err != nil {
		log.Println("[app.Ping]", err)
	}

	if n := h.hub.Flood(-1, b); 0 < n {
		log.Printf("[INFO] %d skipped\n", n)
	}
}

type beltUpdateMsg struct {
	Type     string `json:"type"`
	OptionID int64  `json:"optionid"`
	SFXMode  int    `json:"sfxMode"`
}

const (
	mute = iota
	voice
	belt
)

func (h *Hub) sendBeltUpdate(opt Option, sfxMode int) error {
	b, err := json.Marshal(beltUpdateMsg{"beltUpdate", int64(opt.ID), sfxMode})
	if err != nil {
		return err
	}

	if n := h.hub.Flood(-1, b); 0 < n {
		log.Printf("[INFO] %d skipped\n", n)
	}

	return nil
}

func (h *Hub) sendBeltUnset() error {
	b, err := json.Marshal(beltUpdateMsg{"beltUpdate", -1, mute})
	if err != nil {
		return err
	}

	if n := h.hub.Flood(-1, b); 0 < n {
		log.Printf("[INFO] %d skipped\n", n)
	}

	return nil
}

type newBetMsg struct {
	Type     string `json:"type"`
	OptionID int64  `json:"optionid"`
	Tx       BetTx  `json:"tx"`
}

func (h *Hub) notifyBet(opt uint, btx BetTx) error {
	b, err := json.Marshal(newBetMsg{
		Type:     "newBet",
		OptionID: int64(opt),
		Tx:       btx,
	})
	if err != nil {
		return err
	}

	if n := h.hub.Flood(-1, b); 0 < n {
		log.Printf("[INFO] %d skipped\n", n)
	}

	return nil
}
