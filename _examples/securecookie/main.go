package main

// developers can use any library to add a custom cookie encoder/decoder.
// At this example we use the gorilla's securecookie package:
// $ go get github.com/gorilla/securecookie
// $ go run main.go

import (
	"fmt"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/kataras/go-sessions"
)

func main() {
	app := http.NewServeMux()

	cookieName := "mycustomsessionid"
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey := []byte("the-big-and-secret-fash-key-here")
	blockKey := []byte("lot-secret-of-characters-big-too")
	secureCookie := securecookie.New(hashKey, blockKey)

	mySessions := sessions.New(sessions.Config{
		Cookie: cookieName,
		Encode: secureCookie.Encode,
		Decode: secureCookie.Decode,
	})

	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("You should navigate to the /set, /get, /delete, /clear,/destroy instead")))
	})

	app.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		//set session values
		s := mySessions.Start(w, r)
		s.Set("name", "iris")

		//test if setted here
		w.Write([]byte(fmt.Sprintf("All ok session setted to: %s", s.GetString("name"))))
	})

	app.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		// get a specific key, as string, if no found returns just an empty string
		s := mySessions.Start(w, r)
		name := s.GetString("name")

		w.Write([]byte(fmt.Sprintf("The name on the /set was: %s", name)))
	})

	app.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		// delete a specific key
		s := mySessions.Start(w, r)
		s.Delete("name")
	})

	app.HandleFunc("/clear", func(w http.ResponseWriter, r *http.Request) {
		// removes all entries
		mySessions.Start(w, r).Clear()
	})

	app.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		// updates expire date with a new date
		mySessions.ShiftExpiration(w, r)
	})

	app.HandleFunc("/destroy", func(w http.ResponseWriter, r *http.Request) {
		//destroy, removes the entire session data and cookie
		mySessions.Destroy(w, r)
	})
	http.ListenAndServe(":8080", app)
}
