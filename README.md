[Travis Widget]: https://img.shields.io/travis/kataras/go-sessions.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/go-sessions
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/go-sessions/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v0.0.1-blue.svg?style=flat-square
[Release]: https://github.com/kataras/go-sessions/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/go-sessions
[ChatMain]: https://kataras.rocket.chat/channel/go-sessions
[ChatAlternative]: https://gitter.im/kataras/go-sessions
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/go-sessions
[Documentation Widget]: https://img.shields.io/badge/documentation-reference-5272B4.svg?style=flat-square
[Documentation]: https://godoc.org/github.com/kataras/go-sessions
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-any--OS-yellow.svg?style=flat-square


The **fastest** (web) session manager for the Go Programming Language.


[![Travis Widget]][Travis] [![Release Widget]][Release] [![Documentation Widget]][Documentation] [![Chat Widget]][Chat] [![Report Widget]][Report] [![License Widget]][License]  [![Language Widget]][Language] ![Platform Widget]

- Cleans the temp memory when a session is idle, and re-allocates it to the temp memory when it's necessary. The most used sessions are optimized to be in the front of the memory's list.

- Supports any type of [database](https://github.com/kataras/go-sessions/tree/master/examples/3_redis_sessiondb), currently only Redis.

**A session can be defined as a server-side storage of information that is desired to persist throughout the user's interaction with the web application.**

Instead of storing large and constantly changing data via cookies in the user's browser (i.e. CookieStore), **only a unique identifier is stored on the client side** called a "session id". This session id is passed to the web server on every request. The web application uses the session id as the key for retrieving the stored data from the database/memory. The session data is then available from the net/http Handler when calls the `sessions.Start`.






Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.7.

```bash
$ go get -u github.com/kataras/go-sessions
```

Examples
------------

Take a look at the [./examples](https://github.com/kataras/go-sessions/tree/master/examples) , it's a simple (yet strong) package, easy to understand.


Usage
------------

**OUTLINE**

```go
// Start starts the session for the particular request
Start(http.ResponseWriter, *http.Request) Session
// Destroy kills the session and remove the associated cookie
Destroy(http.ResponseWriter, *http.Request)


// UseDatabase ,optionally, adds a session database to the manager's provider,
// a session db doesn't have write access
// see https://github.com/kataras/go-sessions/tree/master/sessiondb
UseDatabase(Database)
```

`Start` returns a `Session`, **Session outline**

```go
type Session interface {
  ID() string
	Get(string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetAll() map[string]interface{}
	VisitAll(cb func(k string, v interface{}))
	Set(string, interface{})
	Delete(string)
	Clear()
}
```

```go
package main

import (
	"fmt"
	"github.com/kataras/go-sessions"
	"net/http"
)

func main() {

	// set some values to the session
	setHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		values := map[string]interface{}{
			"Name":   "go-sessions",
			"Days":   "1",
			"Secret": "dsads£2132215£%%Ssdsa",
		}

		sess := sessions.Start(res, req) // init the session
		// sessions.Start returns:
		// type Session interface {
		//  ID() string
		//	Get(string) interface{}
		//	GetString(key string) string
		//	GetInt(key string) int
		//	GetAll() map[string]interface{}
		//	VisitAll(cb func(k string, v interface{}))
		//	Set(string, interface{})
		//	Delete(string)
		//	Clear()
		//}

		for k, v := range values {
			sess.Set(k, v) // fill session, set each of the key-value pair
		}
		res.Write([]byte("Session saved, go to /get to view the results"))
	})
	http.Handle("/set/", setHandler)

	// get the values from the session
	getHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := sessions.Start(res, req) // init the session
		sessValues := sess.GetAll()      // get all values from this session

		res.Write([]byte(fmt.Sprintf("%#v", sessValues)))
	})
	http.Handle("/get/", getHandler)

	// clear all values from the session
	clearHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := sessions.Start(res, req)
		sess.Clear()
	})
	http.Handle("/clear/", clearHandler)

	// destroys the session, clears the values and removes the server-side entry and client-side sessionid cookie
	destroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sessions.Destroy(res, req)
	})
	http.Handle("/destroy/", destroyHandler)

	fmt.Println("Open a browser tab and navigate to the localhost:8080/set/")
	http.ListenAndServe(":8080", nil)
}

```

FAQ
------------

If you'd like to discuss this package, or ask questions about it, feel free to

 * Explore [these questions](https://github.com/kataras/go-sessions/issues?go-sessions=label%3Aquestion).
 * Post an issue or  idea [here](https://github.com/kataras/go-sessions/issues).
 * Navigate to the [Chat][Chat].



Versioning
------------

Current: **v0.0.1**

Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions



People
------------
The author of go-sessions is [@kataras](https://github.com/kataras).


Contributing
------------
If you are interested in contributing to the go-sessions project, please make a PR.

License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).
