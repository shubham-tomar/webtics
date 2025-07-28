package main

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type Event struct {
	Event string                 `json:"event"`           // "page_view"
	TS    time.Time              `json:"ts"`              // client timestamp
	URL   string                 `json:"url,omitempty"`   // page url
	Ref   string                 `json:"ref,omitempty"`   // referrer
	Props map[string]interface{} `json:"props,omitempty"` // future‑proof
}

func main() {
	// ---------- DB setup ----------
	db, err := sql.Open("sqlite", "file:events.db?_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(`
	  CREATE TABLE IF NOT EXISTS events(
	      id       INTEGER PRIMARY KEY AUTOINCREMENT,
	      event    TEXT,
	      ts       DATETIME,
	      url      TEXT,
	      ref      TEXT,
	      props    JSON
	  );
	`); err != nil {
		log.Fatal(err)
	}

	// ---------- HTTP handlers ----------
	mux := http.NewServeMux()

	// POST /track
	mux.HandleFunc("/track", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		var ev Event
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if ev.TS.IsZero() {
			ev.TS = time.Now().UTC()
		}

		_, err := db.ExecContext(r.Context(),
			`INSERT INTO events(event, ts, url, ref, props)
			 VALUES (?, ?, ?, ?, json(?))`,
			ev.Event, ev.TS, ev.URL, ev.Ref, string(must(json.Marshal(ev.Props))),
		)
		if err != nil {
			http.Error(w, "db fail", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	// GET /static/*
	staticDir := filepath.Join(".", "static")
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(staticDir))))

	// ---------- ticker to show progress ----------
	go func(ctx context.Context) {
		t := time.NewTicker(time.Minute)
		for {
			select {
			case <-t.C:
				var c int
				if err := db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&c); err == nil {
					log.Printf("[stats] total events=%d\n", c)
				}
			case <-ctx.Done():
				return
			}
		}
	}(context.Background())

	// ---------- run ----------
	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}
