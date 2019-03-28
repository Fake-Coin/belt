package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"goji.io"
	"goji.io/pat"

	"context"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"fakco.in/belt"
	"fakco.in/fakd/chaincfg"
	"fakco.in/fakd/rpcclient"
	"fakco.in/fakutil"
)

var Cfg = struct {
	Database, Cert      string
	MasterToken, Secret string

	TwitchChannel string // dissabled if empty

	Debug       bool
	AutoMigrate bool

	RPC rpcclient.ConnConfig
}{}

var store *sessions.CookieStore

func init() {
	confDir := flag.String("conf", "", "directory for database and config files")
	debug := flag.Bool("debug", false, "enable debug logging")

	flag.Parse()
	if confDir == nil || *confDir == "" {
		flag.PrintDefaults()
	}

	confFile, err := ioutil.ReadFile(filepath.Join(*confDir, "./config.json"))
	if err != nil {
		log.Fatal(err)
	}

	if err := json.Unmarshal(confFile, &Cfg); err != nil {
		log.Fatal(err)
	}

	Cfg.Debug = Cfg.Debug || (debug != nil && *debug)

	if Cfg.Debug {
		log.Println("[init] debug logging enabled")
	}

	Cfg.Database = filepath.Join(*confDir, Cfg.Database)
	store = sessions.NewCookieStore([]byte(Cfg.Secret))

	if !Cfg.RPC.DisableTLS {
		certFile, err := ioutil.ReadFile(filepath.Join(*confDir, Cfg.Cert))
		if err != nil {
			log.Fatalf("[init] could not find RPC certificate file: %q\n", err)
		} else {
			Cfg.RPC.Certificates = certFile
		}
	}
}

func main() {
	db, err := gorm.Open("sqlite3", Cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.LogMode(Cfg.Debug)

	if Cfg.AutoMigrate {
		if Cfg.Debug {
			log.Println("[debug][AutoMigrate] checking database schema")
		}

		if err := belt.Migrate(db); err != nil {
			log.Fatalf("[main] migration failed: %q\n", err)
		} else if Cfg.Debug {
			log.Println("[debug][AutoMigrate] ok")
		}
	}

	app := belt.NewApp(db, Cfg.MasterToken)
	app.Upgrader = websocket.Upgrader{
		HandshakeTimeout: 30 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	if Cfg.TwitchChannel != "" {
		if Cfg.Debug {
			log.Printf("[debug][Twitch] starting twitch chat monitor for %q\n", Cfg.TwitchChannel)
		}

		if err := app.StartChatMonitor(Cfg.TwitchChannel); err != nil {
			log.Fatalf("[debug][Twitch] twitch connection failed: %q\n", err)
		} else if Cfg.Debug {
			log.Println("[debug][Twitch] ok")
		}
	}

	if Cfg.Debug {
		log.Println("[debug][RPC] starting FakeCoin RPC")
	}
	if err := app.StartRPC(&Cfg.RPC, &chaincfg.MainNetParams); err != nil {
		log.Fatal(err)
	}

	// incase of random restarts, reload watch addresses to prevent missed transactions.
	// This must handle duplicate mempool transactions as well as transactions that
	// were sent and confirmed durring the restart that may not be forwarded by watcher.
	// 1. load all addresses into mempool monitor.
	// 2. check all addresses for past transactions.
	// 3. merge. This can be done in the database by ingnoring duplicate errors.
	{
		var initBelt belt.Belt
		res := db.Set("gorm:auto_preload", true).First(&initBelt)
		if res.Error != nil {
			log.Fatal(res.Error)
		}

		var watchAddrs []fakutil.Address
		for _, opt := range initBelt.Options {
			for _, bet := range opt.Bets {
				addr, err := fakutil.DecodeAddress(bet.WatchAddr, &chaincfg.MainNetParams)
				if err != nil {
					log.Fatal(err)
				}

				watchAddrs = append(watchAddrs, addr)
			}
		}

		if err := app.RPCLoad(watchAddrs); err != nil {
			log.Fatal(err)
		}
	}

	// websocket layer keep-alive
	go func() {
		for range time.NewTicker(1 * time.Minute).C {
			app.Ping()
		}
	}()

	mux := goji.NewMux()
	mux.Use(Sessions)
	mux.Use(func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "belt.app", app)
			ctx = context.WithValue(ctx, "database", db)
			h.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	})

	mux.HandleFunc(pat.Get("/auth/:token"), Auth)

	mux.HandleFunc(pat.Get("/"), belt.Index)

	beltMux := goji.SubMux()
	mux.Handle(pat.New("/belt/*"), beltMux)
	BeltRoutes(beltMux, app)

	mux.Handle(pat.Get("/*"), http.FileServer(belt.AssetFS()))

	http.ListenAndServe(":8081", mux)
}

func BeltRoutes(mux *goji.Mux, app *belt.App) {
	mux.Use(belt.Context)
	mux.HandleFunc(pat.Get("/:beltid"), belt.View)
	mux.HandleFunc(pat.Get("/:beltid/ws"), app.Socket)
	mux.HandleFunc(pat.Post("/:beltid/getaddr"), belt.GetAddr) // app
	mux.HandleFunc(pat.Post("/:beltid/vote"), belt.PlaceVote)  // app

	adminMux := goji.SubMux()
	mux.Handle(pat.New("/:beltid/*"), adminMux)

	adminMux.Use(AdminOnly)
	adminMux.HandleFunc(pat.Get("/edit"), belt.Edit)
	adminMux.HandleFunc(pat.Get("/payout"), app.Payout) // app
	adminMux.HandleFunc(pat.Post("/edit"), belt.SaveEdit)
	adminMux.HandleFunc(pat.Post("/setbelt"), app.SetBelt) // app
}

func AdminOnly(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if session, ok := r.Context().Value("session").(*sessions.Session); ok {
			if tk, ok := session.Values["token"].(string); ok && tk == Cfg.MasterToken {
				h.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
	return http.HandlerFunc(fn)
}

func Auth(w http.ResponseWriter, r *http.Request) {
	log.Println(pat.Param(r, "token"), Cfg.MasterToken)
	if pat.Param(r, "token") != Cfg.MasterToken {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	session, ok := r.Context().Value("session").(*sessions.Session)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	session.Values["token"] = Cfg.MasterToken
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func Sessions(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "belt-admin")
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), "session", session)

		h.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func TODO(w http.ResponseWriter, r *http.Request) {
	name := pat.Param(r, "name")
	fmt.Fprintf(w, "Hello, %s!", name)
}
