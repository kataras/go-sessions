package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kataras/go-sessions/v3"
	"github.com/kataras/go-sessions/v3/sessiondb/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// replace with your running mongo' server settings:
	cred := options.Credential{
		AuthSource: "admin",
		Username:   "user",
		Password:   "password",
	}

	clientOpts := options.Client().ApplyURI("mongodb://127.0.0.1:27017").SetAuth(cred)
	db, err := mongo.New(clientOpts, "sessions")
	if err != nil {
		log.Fatal(err)
	}

	// defer db.Close()

	sess := sessions.New(sessions.Config{Cookie: "sessionscookieid"})

	//
	// IMPORTANT:
	//
	sess.UseDatabase(db)

	// the rest of the code stays the same.
	app := http.NewServeMux()

	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("You should navigate to the /set, /get, /delete, /clear,/destroy instead")))
	})
	app.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		s := sess.Start(w, r)
		//set session values
		s.Set("name", "iris")

		//test if setted here
		w.Write([]byte(fmt.Sprintf("All ok session setted to: %s", s.GetString("name"))))
	})

	app.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		// get a specific key, as string, if no found returns just an empty string
		name := sess.Start(w, r).GetString("name")

		w.Write([]byte(fmt.Sprintf("The name on the /set was: %s", name)))
	})

	app.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		// delete a specific key
		sess.Start(w, r).Delete("name")
	})

	app.HandleFunc("/clear", func(w http.ResponseWriter, r *http.Request) {
		// removes all entries
		sess.Start(w, r).Clear()
	})

	app.HandleFunc("/destroy", func(w http.ResponseWriter, r *http.Request) {
		//destroy, removes the entire session data and cookie
		sess.Destroy(w, r)
	})

	app.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		// updates expire date with a new date
		sess.ShiftExpiration(w, r)
	})

	log.Fatal(http.ListenAndServe(":8081", app))

	// block := make(chan bool)
	// <-block
}
