package main

import (
	"net/http"

	"github.com/kataras/go-sessions/v3"
)

var (
	cookieNameForSessionID = "mycookiesessionnameid"
	sess                   = sessions.New(sessions.Config{Cookie: cookieNameForSessionID})
)

func secret(w http.ResponseWriter, r *http.Request) {

	// Check if user is authenticated
	if auth, _ := sess.Start(w, r).GetBoolean("authenticated"); !auth {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Print secret message
	w.Write([]byte("The cake is a lie!"))
}

func login(w http.ResponseWriter, r *http.Request) {
	session := sess.Start(w, r)

	// Authentication goes here
	// ...

	// Set user as authenticated
	session.Set("authenticated", true)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session := sess.Start(w, r)

	// Revoke users authentication
	session.Set("authenticated", false)
}

func main() {
	app := http.NewServeMux()

	app.HandleFunc("/secret", secret)
	app.HandleFunc("/login", login)
	app.HandleFunc("/logout", logout)

	http.ListenAndServe(":8080", app)
}
