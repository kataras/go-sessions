package main

import (
	"fmt"
	"net/http"

	"github.com/kataras/go-sessions/v3"
)

func main() {
	app := http.NewServeMux()
	sess := sessions.New(sessions.Config{Cookie: "myappsessionid"})

	app.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		s := sess.Start(w, r)
		s.SetFlash("name", "iris")
		w.Write([]byte(fmt.Sprintf("Message setted, is available for the next request")))
	})

	app.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		s := sess.Start(w, r)
		name := s.GetFlashString("name")
		if name == "" {
			w.Write([]byte(fmt.Sprintf("Empty name!!")))
			return
		}
		w.Write([]byte(fmt.Sprintf("Hello %s", name)))
	})

	app.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		s := sess.Start(w, r)
		name := s.GetFlashString("name")
		if name == "" {
			w.Write([]byte(fmt.Sprintf("Empty name!!")))
			return
		}

		w.Write([]byte(fmt.Sprintf("Ok you are coming from /set ,the value of the name is %s", name)))
		w.Write([]byte(fmt.Sprintf(", and again from the same context: %s", name)))
	})

	http.ListenAndServe(":8080", app)
}
