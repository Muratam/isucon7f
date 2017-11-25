package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
	"net/http/pprof"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/go-agent"
)

var (
	db *sqlx.DB
	app newrelic.Application
	roomMutex sync.Map
)

func globalTicker() {
	log.Println("Start globalTicker")
	for {
		mutexes := make([]*sync.RWMutex, 0, 100)
		rooms := make([]string, 0, 100)
		roomMutex.Range(func(key, value interface{}) bool {
			mutexes = append(mutexes, value.(*sync.RWMutex))
			rooms = append(rooms, key.(string))
			return true
		})
		log.Println("Attempt to lock")
		for i:=0; i<len(rooms); i++ {
			mutexes[i].Lock()
			log.Println("Lock:", rooms[i])
		}
		log.Println("sleep 500")
		time.Sleep(500 * time.Second)
		log.Println("Attempt to forget")
		for _, room := range rooms {
			group.Forget(room)
		}
		log.Println("Attempt to unlock")
		for i:=0; i<len(rooms); i++ {
			mutexes[i].Unlock()
			log.Println("Unlock:", rooms[i])
		}
		log.Println("sleep 100")
		time.Sleep(100 * time.Second)
	}
}

func initDB() {
	db_host := os.Getenv("ISU_DB_HOST")
	if db_host == "" {
		db_host = "127.0.0.1"
	}
	db_port := os.Getenv("ISU_DB_PORT")
	if db_port == "" {
		db_port = "3306"
	}
	db_user := os.Getenv("ISU_DB_USER")
	if db_user == "" {
		db_user = "root"
	}
	db_password := os.Getenv("ISU_DB_PASSWORD")
	if db_password != "" {
		db_password = ":" + db_password
	}

	dsn := fmt.Sprintf("%s%s@tcp(%s:%s)/isudb?parseTime=true&loc=Local&charset=utf8mb4",
		db_user, db_password, db_host, db_port)

	log.Printf("Connecting to db: %q", dsn)
	db, _ = sqlx.Connect("mysql", dsn)
	for {
		err := db.Ping()
		if err == nil {
			break
		}
		log.Println(err)
		time.Sleep(time.Second * 3)
	}

	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Printf("Succeeded to connect db.")
}

func getInitializeHandler(w http.ResponseWriter, r *http.Request) {
	db.MustExec("TRUNCATE TABLE adding")
	db.MustExec("TRUNCATE TABLE buying")
	db.MustExec("TRUNCATE TABLE room_time")
	w.WriteHeader(204)
}

func getRoomHandler(w http.ResponseWriter, r *http.Request) {
	txn := app.StartTransaction("getRoomHandler", w, r)
	defer txn.End()
	vars := mux.Vars(r)

	roomName := vars["room_name"]
	if _, ok := roomMutex.Load(roomName); !ok {
		roomMutex.Store(roomName, new(sync.RWMutex))
	}
	path := "/ws/" + url.PathEscape(roomName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Host string `json:"host"`
		Path string `json:"path"`
	}{
		Host: "",
		Path: path,
	})
}

func wsGameHandler(w http.ResponseWriter, r *http.Request) {
	txn := app.StartTransaction("wsGameHandler", w, r)
	defer txn.End()
	vars := mux.Vars(r)

	roomName := vars["room_name"]

	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		log.Println("Failed to upgrade", err)
		return
	}
	go serveGameConn(ws, roomName)
}

func AttachProfiler(router *mux.Router) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	// Manually add support for paths linked to by index page at /debug/pprof/
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	initDB()

	cfg := newrelic.NewConfig("isucon7f", os.Getenv("NEW_RELIC_KEY"))
	var err error
	app, err = newrelic.NewApplication(cfg)
	if err != nil {
		log.Fatalln("Failed to connect to New Relic:", err)
	}

	go globalTicker()

	r := mux.NewRouter()
	AttachProfiler(r)
	r.HandleFunc("/initialize", getInitializeHandler)
	r.HandleFunc("/room/", getRoomHandler)
	r.HandleFunc("/room/{room_name}", getRoomHandler)
	r.HandleFunc("/ws/", wsGameHandler)
	r.HandleFunc("/ws/{room_name}", wsGameHandler)
	_, fileserver := newrelic.WrapHandle(app, "/", http.FileServer(http.Dir("../public/")))
	r.PathPrefix("/").Handler(fileserver)

	log.Fatal(http.ListenAndServe(":5000", handlers.LoggingHandler(os.Stderr, r)))
}
