<p align="center">

 <a href="https://github.com/kataras/go-sessions"><img  width="600"  src="https://github.com/kataras/go-sessions/raw/master/logo_900_273_bg_white.png"></a>
 <br/><br/>

 <a href="https://travis-ci.org/kataras/go-sessions"><img src="https://img.shields.io/travis/kataras/go-sessions.svg?style=flat-square" alt="Build Status"></a>
 <a href="https://github.com/kataras/go-sessions/blob/master/LICENSE"><img src="https://img.shields.io/badge/%20license-MIT%20%20License%20-E91E63.svg?style=flat-square" alt="License"></a>
 <a href="https://github.com/kataras/go-sessions/releases"><img src="https://img.shields.io/badge/%20release%20-%20v1.0.2-blue.svg?style=flat-square" alt="Releases"></a>
 <a href="#docs"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Read me docs"></a>
 <br/>
 <a href="https://kataras.rocket.chat/channel/go-sessions"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Build Status"></a>
 <a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
 <a href="#"><img src="https://img.shields.io/badge/platform-All-yellow.svg?style=flat-square" alt="Platforms"></a>

<br/><br/>
<a href="#features" >Fast</a> http sessions manager for Go.<br/>
Simple <a href ="#docs">API</a>, while providing robust set of features.<br/>




</p>

Quick view
-----------

```go
import "github.com/kataras/go-sessions"

sess := sessions.Start(http.ResponseWriter, *http.Request)
sess.
  ID() string
  Get(string) interface{}
  HasFlash() bool
  GetFlash(string) interface{}
  GetFlashString(string) string
  GetString(key string) string
  GetInt(key string) (int, error)
  GetInt64(key string) (int64, error)
  GetFloat32(key string) (float32, error)
  GetFloat64(key string) (float64, error)
  GetBoolean(key string) (bool, error)
  GetAll() map[string]interface{}
  GetFlashes() map[string]interface{}
  VisitAll(cb func(k string, v interface{}))
  Set(string, interface{})
  SetFlash(string, interface{})
  Delete(string)
  Clear()
  ClearFlashes()

```

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.7.

```bash
$ go get -u github.com/kataras/go-sessions
```

Features
------------
- Focus on simplicity and performance.
- Flash messages.
- Supports any type of [external database](https://github.com/kataras/go-sessions/tree/master/_examples/3_redis_sessiondb).
- Works with both [net/http](https://golang.org/pkg/net/http/) and [valyala/fasthttp](https://github.com/valyala/fasthttp).


Docs
------------

Take a look at the [./examples](https://github.com/kataras/go-sessions/tree/master/_examples).


**OUTLINE**

```go
// Start starts the session for the particular net/http request
Start(http.ResponseWriter, *http.Request) Session
// Destroy kills the net/http session and remove the associated cookie
Destroy(http.ResponseWriter, *http.Request)

// Start starts the session for the particular valyala/fasthttp request
StartFasthttp(*fasthttp.RequestCtx) Session
// Destroy kills the valyala/fasthttp session and remove the associated cookie
DestroyFasthttp(*fasthttp.RequestCtx)

// DestroyByID removes the session entry
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
//
// It's safe to use it even if you are not sure if a session with that id exists.
// Works for both net/http & fasthttp
DestroyByID(string)
// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
// Works for both net/http & fasthttp
DestroyAll()

// UseDatabase ,optionally, adds a session database to the manager's provider,
// a session db doesn't have write access
// see https://github.com/kataras/go-sessions/tree/master/sessiondb
UseDatabase(Database)

// UpdateConfig updates the configuration field (Config does not receives a pointer, so this is a way to update a pre-defined configuration)
UpdateConfig(Config)
```


**CONFIGURATION**
```go
	Config struct {
		// Cookie string, the session's client cookie name, for example: "mysessionid"
		//
		// Defaults to "gosessionid"
		Cookie string
		// DecodeCookie if setted to true then the cookie's name will be url-encoded.
		// Note: Users should not confuse the 'Encode' and 'Decode' configuration fields,
		// these are for the cookie's value, which is the session id.
		//
		// Defaults to false
		DecodeCookie bool
		// Encode the cookie value if not nil.
		// Should accept as first argument the cookie name (config.Name)
		//         as second argument the server's generated session id.
		// Should return the new session id, if error the session id setted to empty which is invalid.
		//
		// Note: Errors are not printed, so you have to know what you're doing,
		// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
		// You either need to provide exactly that amount or you derive the key from what you type in.
		//
		// Defaults to nil
		Encode func(cookieName string, value interface{}) (string, error)
		// Decode the cookie value if not nil.
		// Should accept as first argument the cookie name (config.Name)
		//               as second second accepts the client's cookie value (the encoded session id).
		// Should return an error if decode operation failed.
		//
		// Note: Errors are not printed, so you have to know what you're doing,
		// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
		// You either need to provide exactly that amount or you derive the key from what you type in.
		//
		// Defaults to nil
		Decode func(cookieName string, cookieValue string, v interface{}) error

		// Expires the duration of which the cookie must expires (created_time.Add(Expires)).
		// If you want to delete the cookie when the browser closes, set it to -1 but in this case, the server side's session duration is up to GcDuration
		//
		// 0 means no expire, (24 years)
		// -1 means when browser closes
		// > 0 is the time.Duration which the session cookies should expire.
		//
		// Defaults to infinitive/unlimited life duration(0)
		Expires time.Duration

		// CookieLength the length of the sessionid's cookie's value, let it to 0 if you don't want to change it
		//
		// Defaults to 32
		CookieLength int

		// DisableSubdomainPersistence set it to true in order dissallow your q subdomains to have access to the session cookie
		//
		// Defaults to false
	}
```


Usage NET/HTTP
------------


`Start` returns a `Session`, **Session outline**

```go
Session interface {
  ID() string
  Get(string) interface{}
  HasFlash() bool
  GetFlash(string) interface{}
  GetString(key string) string
  GetFlashString(string) string
  GetInt(key string) (int, error)
  GetInt64(key string) (int64, error)
  GetFloat32(key string) (float32, error)
  GetFloat64(key string) (float64, error)
  GetBoolean(key string) (bool, error)
  GetAll() map[string]interface{}
  GetFlashes() map[string]interface{}
  VisitAll(cb func(k string, v interface{}))
  Set(string, interface{})
  SetFlash(string, interface{})
  Delete(string)
  Clear()
  ClearFlashes()
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
   	 	// sessions.Start returns the Session interface we saw before	
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

> Look [sessions_test.go](https://github.com/kataras/go-sessions/blob/master/sessions_test.go) for more, like lifetime, cookie encoding/decoding.


Usage FASTHTTP
------------

`StartFasthttp` returns again `Session`, **Session outline**

```go
Session interface {
  ID() string
  Get(string) interface{}
  HasFlash() bool
  GetFlash(string) interface{}
  GetString(key string) string
  GetFlashString(string) string
  GetInt(key string) (int, error)
  GetInt64(key string) (int64, error)
  GetFloat32(key string) (float32, error)
  GetFloat64(key string) (float64, error)
  GetBoolean(key string) (bool, error)
  GetAll() map[string]interface{}
  GetFlashes() map[string]interface{}
  VisitAll(cb func(k string, v interface{}))
  Set(string, interface{})
  SetFlash(string, interface{})
  Delete(string)
  Clear()
  ClearFlashes()
}
```

```go
package main

import (
	"fmt"
	"github.com/kataras/go-sessions"
	"github.com/valyala/fasthttp"
)

func main() {

	// set some values to the session
	setHandler := func(reqCtx *fasthttp.RequestCtx) {
		values := map[string]interface{}{
			"Name":   "go-sessions",
			"Days":   "1",
			"Secret": "dsads£2132215£%%Ssdsa",
		}

		sess := sessions.StartFasthttp(reqCtx) // init the session
		// sessions.StartFasthttp returns the, same, Session interface we saw before too

		for k, v := range values {
			sess.Set(k, v) // fill session, set each of the key-value pair
		}
		reqCtx.WriteString("Session saved, go to /get to view the results")
	}

	// get the values from the session
	getHandler := func(reqCtx *fasthttp.RequestCtx) {
		sess := sessions.StartFasthttp(reqCtx) // init the session
		sessValues := sess.GetAll()    // get all values from this session

		reqCtx.WriteString(fmt.Sprintf("%#v", sessValues))
	}

	// clear all values from the session
	clearHandler := func(reqCtx *fasthttp.RequestCtx) {
		sess := sessions.StartFasthttp(reqCtx)
		sess.Clear()
	}

	// destroys the session, clears the values and removes the server-side entry and client-side sessionid cookie
	destroyHandler := func(reqCtx *fasthttp.RequestCtx) {
		sessions.DestroyFasthttp(reqCtx)
	}

	fmt.Println("Open a browser tab and navigate to the localhost:8080/set")
	fasthttp.ListenAndServe(":8080", func(reqCtx *fasthttp.RequestCtx) {
		path := string(reqCtx.Path())

		if path == "/set" {
			setHandler(reqCtx)
		} else if path == "/get" {
			getHandler(reqCtx)
		} else if path == "/clear" {
			clearHandler(reqCtx)
		} else if path == "/destroy" {
			destroyHandler(reqCtx)
		} else {
			reqCtx.WriteString("Please navigate to /set or /get or /clear or /destroy")
		}
	})
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

Current: **v1.0.2**

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

[Travis Widget]: https://img.shields.io/travis/kataras/go-sessions.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/go-sessions
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/go-sessions/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v1.0.2-blue.svg?style=flat-square
[Release]: https://github.com/kataras/go-sessions/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/go-sessions
[ChatMain]: https://kataras.rocket.chat/channel/go-sessions
[ChatAlternative]: https://gitter.im/kataras/go-sessions
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/go-sessions
[Documentation Widget]: https://img.shields.io/badge/docs-reference-5272B4.svg?style=flat-square
[Documentation]: https://godoc.org/github.com/kataras/go-sessions
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-yellow.svg?style=flat-square
