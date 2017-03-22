// Package sessions provides sessions support for net/http and valyala/fasthttp
// unique with auto-GC, register unlimited number of databases to Load and Update/Save the sessions in external server or to an external (no/or/and sql) database
// Usage net/http:
// // init a new sessions manager( if you use only one web framework inside your app then you can use the package-level functions like: sessions.Start/sessions.Destroy)
// manager := sessions.New(sessions.Config{})
// // start a session for a particular client
// manager.Start(http.ResponseWriter, *http.Request)
//
// // destroy a session from the server and client,
//  // don't call it on each handler, only on the handler you want the client to 'logout' or something like this:
// manager.Destroy(http.ResponseWriter, *http.Request)
//
//
// Usage valyala/fasthttp:
// // init a new sessions manager( if you use only one web framework inside your app then you can use the package-level functions like: sessions.Start/sessions.Destroy)
// manager := sessions.New(sessions.Config{})
// // start a session for a particular client
// manager.StartFasthttp(*fasthttp.RequestCtx)
//
// // destroy a session from the server and client,
//  // don't call it on each handler, only on the handler you want the client to 'logout' or something like this:
// manager.DestroyFasthttp(*fasthttp.Request)
//
// Note that, now, you can use both fasthttp and net/http within the same sessions manager(.New) instance!
// So now, you can share sessions between a net/http app and valyala/fasthttp app
package sessions

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	// Version current version number
	Version = "1.0.2"
)

type (
	// Sessions is the start point of this package
	// contains all the registered sessions and manages them
	Sessions interface {
		// Set options/configuration fields in runtime
		Set(...OptionSetter)

		// UseDatabase ,optionally, adds a session database to the manager's provider,
		// a session db doesn't have write access
		// see https://github.com/kataras/go-sessions/tree/master/sessiondb
		UseDatabase(Database)

		// Start starts the session for the particular net/http request
		Start(http.ResponseWriter, *http.Request) Session

		// Destroy kills the net/http session and remove the associated cookie
		Destroy(http.ResponseWriter, *http.Request)

		// StartFasthttp starts the session for the particular valyala/fasthttp request
		StartFasthttp(*fasthttp.RequestCtx) Session

		// DestroyFasthttp kills the valyala/fasthttp session and remove the associated cookie
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
	}

	// sessions contains the cookie's name, the provider and a duration for GC and cookie life expire
	sessions struct {
		config   Config
		provider *provider
	}
)

// New creates & returns a new Sessions(manager) and start its GC (calls the .Init)
func New(setters ...OptionSetter) Sessions {
	c := Config{}.Validate()
	sess := &sessions{config: c, provider: newProvider()}
	sess.Set(setters...)

	return sess
}

var defaultSessions = New(Config{}.Validate())

// Set options/configuration fields in runtime
func Set(setters ...OptionSetter) {
	defaultSessions.Set(setters...)
}

func (s *sessions) Set(setters ...OptionSetter) {
	for _, setter := range setters {
		setter.Set(&s.config)
	}
}

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func UseDatabase(db Database) {
	defaultSessions.UseDatabase(db)
}

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func (s *sessions) UseDatabase(db Database) {
	s.provider.RegisterDatabase(db)
}

// Start starts the session for the particular net/http request
func Start(res http.ResponseWriter, req *http.Request) Session {
	return defaultSessions.Start(res, req)
}

// Start starts the session for the particular net/http request
func (s *sessions) Start(res http.ResponseWriter, req *http.Request) Session {
	var sess Session

	cookieValue := GetCookie(s.config.Cookie, req)
	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := SessionIDGenerator(s.config.CookieLength)
		sess = s.provider.Init(sid, s.config.Expires)
		cookie := &http.Cookie{}

		// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
		cookie.Name = s.config.Cookie
		cookie.Value = sid
		cookie.Path = "/"
		if !s.config.DisableSubdomainPersistence {

			requestDomain := req.URL.Host
			if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
				requestDomain = requestDomain[0:portIdx]
			}
			if IsValidCookieDomain(requestDomain) {

				// RFC2109, we allow level 1 subdomains, but no further
				// if we have localhost.com , we want the localhost.cos.
				// so if we have something like: mysubdomain.localhost.com we want the localhost here
				// if we have mysubsubdomain.mysubdomain.localhost.com we want the .mysubdomain.localhost.com here
				// slow things here, especially the 'replace' but this is a good and understable( I hope) way to get the be able to set cookies from subdomains & domain with 1-level limit
				if dotIdx := strings.LastIndexByte(requestDomain, '.'); dotIdx > 0 {
					// is mysubdomain.localhost.com || mysubsubdomain.mysubdomain.localhost.com
					s := requestDomain[0:dotIdx] // set mysubdomain.localhost || mysubsubdomain.mysubdomain.localhost
					if secondDotIdx := strings.LastIndexByte(s, '.'); secondDotIdx > 0 {
						//is mysubdomain.localhost ||  mysubsubdomain.mysubdomain.localhost
						s = s[secondDotIdx+1:] // set to localhost || mysubdomain.localhost
					}
					// replace the s with the requestDomain before the domain's siffux
					subdomainSuff := strings.LastIndexByte(requestDomain, '.')
					if subdomainSuff > len(s) { // if it is actual exists as subdomain suffix
						requestDomain = strings.Replace(requestDomain, requestDomain[0:subdomainSuff], s, 1) // set to localhost.com || mysubdomain.localhost.com
					}
				}
				// finally set the .localhost.com (for(1-level) || .mysubdomain.localhost.com (for 2-level subdomain allow)
				cookie.Domain = "." + requestDomain // . to allow persistance
			}

		}
		cookie.HttpOnly = true

		// MaxAge=0 means no 'Max-Age' attribute specified.
		// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
		// MaxAge>0 means Max-Age attribute present and given in seconds
		if s.config.Expires >= 0 {
			if s.config.Expires == 0 { // unlimited life
				cookie.Expires = CookieExpireUnlimited
			} else { // > 0
				cookie.Expires = time.Now().Add(s.config.Expires)
			}
			cookie.MaxAge = int(cookie.Expires.Sub(time.Now()).Seconds())
		}
		// encode the session id cookie client value right before send it.
		cookie.Value = s.encodeCookieValue(cookie.Value)
		AddCookie(cookie, res)
	} else {
		cookieValue = s.decodeCookieValue(cookieValue)

		sess = s.provider.Read(cookieValue, s.config.Expires)
	}
	return sess
}

// Destroy kills the net/http session and remove the associated cookie
func Destroy(res http.ResponseWriter, req *http.Request) {
	defaultSessions.Destroy(res, req)
}

// Destroy kills the net/http session and remove the associated cookie
func (s *sessions) Destroy(res http.ResponseWriter, req *http.Request) {
	cookieValue := GetCookie(s.config.Cookie, req)
	// decode the client's cookie value in order to find the server's session id
	// to destroy the session data.
	cookieValue = s.decodeCookieValue(cookieValue)
	if cookieValue == "" { // nothing to destroy
		return
	}
	RemoveCookie(s.config.Cookie, res, req)
	s.provider.Destroy(cookieValue)
}

// DestroyByID removes the session entry
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
//
// It's safe to use it even if you are not sure if a session with that id exists.
//
// Note: sid is the original session id (may fetched by a store).
//
// Works for both net/http & fasthttp
func DestroyByID(sid string) {
	defaultSessions.DestroyByID(sid)
}

// DestroyByID removes the session entry
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
//
// It's safe to use it even if you are not sure if a session with that id exists.
//
// Note: sid is the original session id (may fetched by a store).
//
// Works for both net/http & fasthttp
func (s *sessions) DestroyByID(sid string) {
	s.provider.Destroy(sid)
}

// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
// Works for both net/http & fasthttp
func DestroyAll() {
	defaultSessions.DestroyAll()
}

// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
// Works for both net/http & fasthttp
func (s *sessions) DestroyAll() {
	s.provider.DestroyAll()
}

// StartFasthttp starts the session for the particular valyala/fasthttp request
func StartFasthttp(reqCtx *fasthttp.RequestCtx) Session {
	return defaultSessions.StartFasthttp(reqCtx)
}

// Start starts the session for the particular valyala/fasthttp request
func (s *sessions) StartFasthttp(reqCtx *fasthttp.RequestCtx) Session {
	var sess Session

	cookieValue := GetFasthttpCookie(s.config.Cookie, reqCtx)

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := SessionIDGenerator(s.config.CookieLength)
		sess = s.provider.Init(sid, s.config.Expires)
		cookie := fasthttp.AcquireCookie()
		// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
		cookie.SetKey(s.config.Cookie)
		cookie.SetValue(sid)
		cookie.SetPath("/")
		if !s.config.DisableSubdomainPersistence {
			requestDomain := string(reqCtx.Host())
			if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
				requestDomain = requestDomain[0:portIdx]
			}
			if IsValidCookieDomain(requestDomain) {

				// RFC2109, we allow level 1 subdomains, but no further
				// if we have localhost.com , we want the localhost.cos.
				// so if we have something like: mysubdomain.localhost.com we want the localhost here
				// if we have mysubsubdomain.mysubdomain.localhost.com we want the .mysubdomain.localhost.com here
				// slow things here, especially the 'replace' but this is a good and understable( I hope) way to get the be able to set cookies from subdomains & domain with 1-level limit
				if dotIdx := strings.LastIndexByte(requestDomain, '.'); dotIdx > 0 {
					// is mysubdomain.localhost.com || mysubsubdomain.mysubdomain.localhost.com
					s := requestDomain[0:dotIdx] // set mysubdomain.localhost || mysubsubdomain.mysubdomain.localhost
					if secondDotIdx := strings.LastIndexByte(s, '.'); secondDotIdx > 0 {
						//is mysubdomain.localhost ||  mysubsubdomain.mysubdomain.localhost
						s = s[secondDotIdx+1:] // set to localhost || mysubdomain.localhost
					}
					// replace the s with the requestDomain before the domain's siffux
					subdomainSuff := strings.LastIndexByte(requestDomain, '.')
					if subdomainSuff > len(s) { // if it is actual exists as subdomain suffix
						requestDomain = strings.Replace(requestDomain, requestDomain[0:subdomainSuff], s, 1) // set to localhost.com || mysubdomain.localhost.com
					}
				}
				// finally set the .localhost.com (for(1-level) || .mysubdomain.localhost.com (for 2-level subdomain allow)
				cookie.SetDomain("." + requestDomain) // . to allow persistance
			}

		}
		cookie.SetHTTPOnly(true)
		if s.config.Expires == 0 {
			// unlimited life
			cookie.SetExpire(CookieExpireUnlimited)
		} else if s.config.Expires > 0 {
			cookie.SetExpire(time.Now().Add(s.config.Expires))
		} // if it's -1 then the cookie is deleted when the browser closes

		// encode the session id cookie client value right before send it.
		cookie.SetValue(s.encodeCookieValue(string(cookie.Value())))
		AddFasthttpCookie(cookie, reqCtx)
		fasthttp.ReleaseCookie(cookie)
	} else {
		cookieValue = s.decodeCookieValue(cookieValue)

		sess = s.provider.Read(cookieValue, s.config.Expires)
	}
	return sess
}

// DestroyFasthttp kills the valyala/fasthttp session and remove the associated cookie
func DestroyFasthttp(reqCtx *fasthttp.RequestCtx) {
	defaultSessions.DestroyFasthttp(reqCtx)
}

// DestroyFasthttp kills the valyala/fasthttp session and remove the associated cookie
func (s *sessions) DestroyFasthttp(reqCtx *fasthttp.RequestCtx) {
	cookieValue := GetFasthttpCookie(s.config.Cookie, reqCtx)
	// decode the client's cookie value in order to find the server's session id
	// to destroy the session data.
	cookieValue = s.decodeCookieValue(cookieValue)
	if cookieValue == "" { // nothing to destroy
		return
	}
	RemoveFasthttpCookie(s.config.Cookie, reqCtx)
	s.provider.Destroy(cookieValue)
}

// Global generator, no logic for per-manager for now.

// SessionIDGenerator returns a random string, used to set the session id
// you are able to override this to use your own method for generate session ids
var SessionIDGenerator = func(strLength int) string {
	return base64.URLEncoding.EncodeToString(random(strLength))
}

// let's keep these funcs simple, we can do it with two lines but we may add more things in the future.
func (s *sessions) decodeCookieValue(cookieValue string) string {
	var cookieValueDecoded *string
	if decode := s.config.Decode; decode != nil {
		err := decode(s.config.Cookie, cookieValue, &cookieValueDecoded)
		if err == nil {
			cookieValue = *cookieValueDecoded
		} else {
			cookieValue = ""
		}
	}
	return cookieValue
}

func (s *sessions) encodeCookieValue(cookieValue string) string {
	if encode := s.config.Encode; encode != nil {
		newVal, err := encode(s.config.Cookie, cookieValue)
		if err == nil {
			cookieValue = newVal
		} else {
			cookieValue = ""
		}
	}

	return cookieValue
}
