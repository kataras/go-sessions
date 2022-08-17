package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	r "github.com/go-redis/redis/v8"
	"github.com/kataras/go-sessions/v3"
	"github.com/kataras/go-sessions/v3/sessiondb/go-redis"
)

const msg = `You can visit:

- /private
- /login
- /logout`

func main() {
	redisSession := redis.NewSessions(
		&r.Options{
			Addr:     os.Getenv("REDIS_ADDR"),
			Username: os.Getenv("REDIS_USER"),
			Password: os.Getenv("REDIS_PASS"),
		},
		&redis.Config{
			Prefix: "prefix",
		},
	)
	session := sessions.New(sessions.Config{
		Cookie:  "session-name",
		Expires: time.Hour * 24 * 7,
	})
	session.UseDatabase(redisSession)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(msg))
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		s := session.Start(w, r)
		s.Set("authenticated", true)
		s.Set("last_login", time.Now().Format(time.RFC3339))

		w.Write([]byte("Logged in!"))
	})

	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		s := session.Start(w, r)
		s.Destroy()

		w.Write([]byte("Logged out!"))
	})

	mux.HandleFunc("/private", func(w http.ResponseWriter, r *http.Request) {
		s := session.Start(w, r)
		if s.GetBooleanDefault("authenticated", false) {
			ll := s.GetStringDefault("last_login", "unknown")
			w.Write([]byte(fmt.Sprintf("Welcome back! Last login %s.", ll)))
			return
		}

		w.Write([]byte("Unauthenticated!"))
	})

	log.Println("Listening on port 8080")
	log.Println(msg)
	http.ListenAndServe(":8080", mux)
}
