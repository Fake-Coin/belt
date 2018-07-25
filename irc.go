package belt

import (
	"encoding/json"
	"fmt"
	"log"
	"math/bits"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc"
)

type ChatMsg struct {
	twitch.User
	twitch.Message
	Belt BeltHolder
}
type LogLog struct {
	chatLog []*ChatMsg
}

func (app *App) StartChatMonitor() {
	client := twitch.NewClient("justinfan123123", "oauth:123123123")

	client.OnNewMessage(app.OnNewChatMessage)

	client.Join("nightattack")

	err := client.Connect()
	if err != nil {
		log.Println(err)
	}
}

func (app *App) OnNewChatMessage(channel string, user twitch.User, message twitch.Message) {
	lower := strings.ToLower(message.Text)
	if !strings.Contains(lower, "belt") {
		return
	}

	guess := Check(lower)

	if guess == 256 {
		return
	}

	app.ll.chatLog = append(app.ll.chatLog, &ChatMsg{user, message, guess})

	app.checkLog(app.ll)
}

func (h *Hub) sendChatUpdate(msg chatMsg) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if n := h.hub.Flood(-1, b); 0 < n {
		log.Printf("[INFO] %d skipped\n", n)
	}

	return nil
}

type chatMsg struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (app *App) checkLog(ll LogLog) {
	logStart := ll.chatLog[0].Time

	belt := BeltHolder(0)
	for j := 1; j < len(ll.chatLog); j++ {
		bl := ll.chatLog[:j]
		end := len(bl) - 1
		prev := bl[end].Time

		var holds [4]int
		for k, v := range [4]BeltHolder{Brian, Jury, Bryce, Bonnie} {
			if belt == v {
				holds[k]--
			}
		}

		for i := end; 0 < i && prev.Sub(bl[i].Time) < 1*time.Minute; i-- {
			current := bl[i]

			newS := current.Belt.Split()
			for k, v := range newS {
				holds[k] += v
			}

			prev = current.Time
		}

		pWin, max := Win(holds)

		if len(pWin) == 1 && len(bl[end].Text) < 80 && bl[end].Belt&256 != 256 {
			belt = pWin[0]
		} else if 2 < max {
			belt = 0
		}

		if max == 0 {
			fmt.Println()
		}
		b := bl[end]
		name := fmt.Sprintf("[%s]", b.DisplayName)
		var body [80]byte
		copy(body[:], []byte(b.Text))
		var sb strings.Builder
		fmt.Fprintf(&sb, "%23s %6s %12s", fmt.Sprintf("%v, %d", pWin, max), belt, b.Time.Sub(logStart))
		fmt.Fprintf(&sb, "%s %18s %s\n", "", name, b.Belt.Split2())

		app.hub.sendChatUpdate(chatMsg{Type: "beltchat", Text: sb.String()})

		// log.Println(bl[end].CreatedAt.Sub(logStart), holds, belt, pWin, bl[end].Message.Body)
	}
}

func Check(msg string) (hold BeltHolder) {
	if strings.Contains(msg, "?") {
		hold = 256
	}

	if strings.Contains(msg, "bri") {
		hold |= Brian
	}
	if strings.Contains(msg, "bry") {
		hold |= Bryce
	}
	if strings.Contains(msg, "jury") || strings.Contains(msg, "justin") || strings.Contains(msg, "jerb") {
		hold |= Jury
	}
	if strings.Contains(msg, "bon") {
		hold |= Bonnie
	}

	switch bits.OnesCount(uint(hold)) {
	case 0:
		return 128 | hold
	case 1:
		return hold
	default:
		return 512 | hold
	}
}

type BeltHolder uint

const (
	Brian BeltHolder = 1 << iota
	Jury
	Bryce
	Bonnie
)

func (h BeltHolder) String() string {
	switch h {
	case Brian:
		return "Brian"
	case Jury:
		return "JuRY"
	case Bryce:
		return "Bryce"
	case Bonnie:
		return "Bonnie"
	default:
		return "???"
	}
}

func (h BeltHolder) Split2() (ret []BeltHolder) {
	if h&Brian == Brian {
		ret = append(ret, Brian)
	}
	if h&Jury == Jury {
		ret = append(ret, Jury)
	}
	if h&Bryce == Bryce {
		ret = append(ret, Bryce)
	}
	if h&Bonnie == Bonnie {
		ret = append(ret, Bonnie)
	}
	if h&256 == 256 {
		ret = append(ret, 256)
	}
	return ret
}

func (h BeltHolder) Split() (ret [4]int) {
	if h&Brian == Brian {
		ret[0]++
	}
	if h&Jury == Jury {
		ret[1]++
	}
	if h&Bryce == Bryce {
		ret[2]++
	}
	if h&Bonnie == Bonnie {
		ret[3]++
	}
	if h&128 == 128 {
		for i := range ret {
			ret[i]++
		}
	}
	return ret
}

func Win(h [4]int) ([]BeltHolder, int) {
	brian, jury, bryce, bonnie := h[0], h[1], h[2], h[3]
	if brian == jury && jury == bryce && bryce == bonnie {
		return nil, h[0]
	}

	var max int
	for _, v := range h {
		if max < v {
			max = v
		}
	}

	var win []BeltHolder
	set := [4]BeltHolder{Brian, Jury, Bryce, Bonnie}
	for i, v := range h {
		if max == v {
			win = append(win, set[i])
		}
	}

	return win, max
}
