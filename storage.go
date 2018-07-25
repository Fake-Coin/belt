package belt

import (
	"fmt"
	"html/template"
	"time"

	"github.com/jinzhu/gorm"
	blackfriday "gopkg.in/russross/blackfriday.v2"
	"vallon.me/shortening"
)

type FAK uint64

func (f FAK) String() string {
	return fmt.Sprintf("%g", float64(f)/100000000)
}

func (f FAK) Int() uint64 {
	return uint64(f)
}

type Belt struct {
	gorm.Model

	Title   string    `json:"title" gorm:"unique_index;not null"`
	Message string    `json:"message"`
	EndTime time.Time `json:"endtime"`

	Options []Option `json:"options"`
}

func (b Belt) Info() template.HTML {
	return template.HTML(
		blackfriday.Run([]byte(b.Message),
			blackfriday.WithExtensions(
				blackfriday.NoEmptyLineBeforeBlock|
					blackfriday.CommonExtensions)))
}

func (b Belt) ShortID() string {
	return string(shortening.Encode(uint64(b.ID)))
}

func (b Belt) Ended() bool {
	return b.EndTime.Before(time.Now())
}

type Option struct {
	ID     uint `gorm:"primary_key"`
	BeltID uint `json:"-" gorm:"unique_index:unique_belt_option"`

	Name  string `json:"name" gorm:"unique_index:unique_belt_option;not null"`
	Image string `json:"image"`

	Bets []Bet
}

func (o Option) Value() FAK {
	var val FAK
	for _, bet := range o.Bets {
		val += bet.Value()
	}
	return val
}

type Bet struct {
	ID        uint      `json:"-" gorm:"primary_key"`
	CreatedAt time.Time `json:"-"`
	OptionID  uint      `json:"-"`

	WIFKey    string `json:"-"`
	WatchAddr string `json:"watchaddr" gorm:"size:35;unique_index;not null"`
	PayAddr   string `json:"payaddr" gorm:"size:35"`

	Transactions []BetTx `json:"txs"`
}

func (b Bet) Value() FAK {
	var val FAK
	for _, tx := range b.Transactions {
		val += tx.Value
	}
	return val
}

type BetTx struct {
	ID        uint      `json:"-" gorm:"primary_key"`
	CreatedAt time.Time `json:"-"`
	BetID     uint      `json:"-"`

	Hash  string `json:"hash" gorm:"size:64;unique_index;not null"`
	Index uint32 `json:"index" gorm:"not null"`
	Value FAK    `json:"value" gorm:"not null"`
}

// const HashSize = 32
// const MaxHashStringSize = HashSize * 2
