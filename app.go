package belt

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"

	"vallon.me/shortening"

	"fakco.in/addrgen"
	"fakco.in/fakd/chaincfg"
	"fakco.in/fakd/rpcclient"
	"fakco.in/fakutil"
)

type App struct {
	Upgrader websocket.Upgrader

	Token string
	db    *gorm.DB

	rpcClient *rpcclient.Client

	hub *Hub

	ll LogLog
}

func NewApp(db *gorm.DB, token string) *App {
	return &App{
		Token: token,
		db:    db,
		hub:   NewHub(),
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// TODO: remove
func (app *App) Ping() {
	app.hub.Ping()
}

func (app *App) StartRPC(cfg *rpcclient.ConnConfig, params *chaincfg.Params) (err error) {
	app.rpcClient, err = rpcclient.New(cfg,
		&rpcclient.NotificationHandlers{
			OnRelevantTxAccepted: app.onRelevantTxAccepted,
		})

	if err != nil {
		return fmt.Errorf("[app.StartRPC] %s", err)
	}

	return nil
}

func (app *App) RPCLoad(addrs []fakutil.Address) error {
	return app.rpcClient.LoadTxFilter(false, addrs, nil)
}

func Index(w http.ResponseWriter, r *http.Request) {
	db, ok := r.Context().Value("database").(*gorm.DB)
	if !ok {
		log.Println("could not find database in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var res struct{ ID uint64 }
	db.Table("belts").Select("id").First(&res)

	loc := path.Join(r.URL.Path, "belt", string(shortening.Encode(res.ID)))
	http.Redirect(w, r, loc, http.StatusSeeOther)
}

func View(w http.ResponseWriter, r *http.Request) {
	belt, ok := r.Context().Value("beltctx").(*BeltCtx)
	if !ok {
		log.Println("could not find belt in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	app, ok := r.Context().Value("belt.app").(*App)
	if !ok {
		log.Println("could not find belt.app in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	r.ParseForm()
	if _, ok := r.Form["json"]; ok {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")

		w.Header().Set("Content-Type", "application/json")
		if err := enc.Encode(belt.Belt); err != nil {
			log.Println(err)
		}
		return
	}

	_, ctrl := r.Form["ctrl"]

	var isAdmin bool
	if session, ok := r.Context().Value("session").(*sessions.Session); ok {
		if tk, ok := session.Values["token"].(string); ok && tk == app.Token {
			isAdmin = true
		}
	}

	if err := beltTmpl.ExecuteTemplate(w, "index.html", struct {
		Base         string
		Belt         *Belt
		IsAdmin      bool
		CtrlsEnabled bool
		BeltHolder   int64
	}{r.URL.Path, belt.Belt, isAdmin, ctrl, app.hub.BeltHolder()}); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// GetAddr requires app context for registering on txfilter
func GetAddr(w http.ResponseWriter, r *http.Request) {
	app, ok := r.Context().Value("belt.app").(*App)
	if !ok {
		log.Println("could not find belt.app context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	optionID, err := strconv.Atoi(r.PostFormValue("optionID"))
	if err != nil {
		log.Println("[GetAddr] bad option id", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var opt Option
	if res := app.db.First(&opt, uint(optionID)); res.Error != nil {
		log.Printf("[GetAddr] '%d' %s\n", optionID, res.Error)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	// TODO: validate back reference

	payAddr, err := fakutil.DecodeAddress(r.PostFormValue("payAddr"), &chaincfg.MainNetParams)
	if err != nil {
		errMsg := fmt.Sprintf("%s - %s", http.StatusText(http.StatusBadRequest), err)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	var privKey string
	var watchAddr fakutil.Address
	{
		wif, watch := addrgen.MakeAddress()
		addr, err := fakutil.DecodeAddress(watch.String(), &chaincfg.MainNetParams)
		if err != nil {
			errMsg := fmt.Sprintf("%s - %s", http.StatusText(http.StatusBadRequest), err)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}

		privKey = wif.String()
		watchAddr = addr
	}

	bet := Bet{
		WIFKey:    privKey,
		WatchAddr: watchAddr.String(),
		PayAddr:   payAddr.String(),
	}

	if res := app.db.Model(&opt).Association("Bets").Append(&bet); res.Error != nil {
		log.Println("[GetAddr]", res.Error)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := app.rpcClient.LoadTxFilter(false, []fakutil.Address{watchAddr}, nil); err != nil {
		log.Println("[GetAddr]", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(bet); err != nil {
		log.Println(err)
	}
}
