// Package sessions provides sessions support for valyala/fasthttp, import "github.com/kataras/go-sessions/fasthttp" but package name is 'sessions.' too.
// unique with auto-GC, register unlimited number of databases to Load and Update/Save the sessions in external server or to an external (no/or/and sql) database
// Usage:
// // init a new sessions manager( if you use only one web framework inside your app then you can use the package-level functions like: sessions.Start/sessions.Destroy)
// manager := sessions.New(sessions.Config{})
// // start a session for a particular client
// manager.Start(*fasthttp.RequestCtx)
//
// // destroy a session from the server and client,
//  // don't call it on each handler, only on the handler you want the client to 'logout' or something like this:
// manager.Destroy(*fasthttp.RequestCtx)
package sessions

import (
	"github.com/kataras/go-sessions"
	"github.com/valyala/fasthttp"
	"strings"
	"time"
)

type (
	// Sessions is the start point of this package
	// contains all the registered sessions and manages them
	Sessions interface {
		// UseDatabase ,optionally, adds a session database to the manager's provider,
		// a session db doesn't have write access
		// see https://github.com/kataras/go-sessions/tree/master/sessiondb
		UseDatabase(sessions.Database)
		// Start starts the session for the particular request
		Start(*fasthttp.RequestCtx) sessions.Session
		// Destroy kills the session and remove the associated cookie
		Destroy(*fasthttp.RequestCtx)
	}
	// fasthttpsessions contains the cookie's name, the provider and a duration for GC and cookie life expire
	fasthttpsessions struct {
		config   sessions.Config
		provider *sessions.Provider
	}
)

// New creates & returns a new Sessions(manager) and start its GC
func New(cfg Config) Sessions {
	// convert the fasthttp/config to sessions/config
	// this done because I want the user to be able to import only the /fasthttp and pass the Config withot need to import the root package
	c := sessions.Config(cfg).Validate()
	// init and start the sess manager
	sess := &fasthttpsessions{config: c, provider: sessions.NewProvider(c.Expires)}
	//run the GC here
	go sess.gc()
	return sess
}

var defaultSessions = New(Config{
	Cookie:                      sessions.DefaultCookieName,
	DecodeCookie:                false,
	Expires:                     sessions.DefaultCookieExpires,
	CookieLength:                sessions.DefaultCookieLength,
	GcDuration:                  sessions.DefaultGcDuration,
	DisableSubdomainPersistence: false,
})

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func UseDatabase(db sessions.Database) {
	defaultSessions.UseDatabase(db)
}

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func (m *fasthttpsessions) UseDatabase(db sessions.Database) {
	m.provider.Expires = m.config.Expires // updae the expires confiuration field for any case
	m.provider.RegisterDatabase(db)
}

// Start starts the session for the particular request
func Start(reqCtx *fasthttp.RequestCtx) sessions.Session {
	return defaultSessions.Start(reqCtx)
}

// Start starts the session for the particular request
func (m *fasthttpsessions) Start(reqCtx *fasthttp.RequestCtx) sessions.Session {
	var sess sessions.Session

	cookieValue := GetCookie(m.config.Cookie, reqCtx)

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := sessions.GenerateSessionID(m.config.CookieLength)
		sess = m.provider.Init(sid)
		cookie := fasthttp.AcquireCookie()
		//cookie := &fasthttp.Cookie{}
		// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
		cookie.SetKey(m.config.Cookie)
		cookie.SetValue(sid)
		cookie.SetPath("/")
		if !m.config.DisableSubdomainPersistence {
			requestDomain := string(reqCtx.Host())
			if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
				requestDomain = requestDomain[0:portIdx]
			}
			if sessions.IsValidCookieDomain(requestDomain) {

				// RFC2109, we allow level 1 subdomains, but no further
				// if we have localhost.com , we want the localhost.com.
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
		if m.config.Expires == 0 {
			// unlimited life
			cookie.SetExpire(sessions.CookieExpireUnlimited)
		} else if m.config.Expires > 0 {
			cookie.SetExpire(time.Now().Add(m.config.Expires))
		} // if it's -1 then the cookie is deleted when the browser closes

		AddCookie(cookie, reqCtx)
		fasthttp.ReleaseCookie(cookie)
	} else {
		sess = m.provider.Read(cookieValue)
	}
	return sess
}

// Destroy kills the session and remove the associated cookie
func Destroy(reqCtx *fasthttp.RequestCtx) {
	defaultSessions.Destroy(reqCtx)
}

// Destroy kills the session and remove the associated cookie
func (m *fasthttpsessions) Destroy(reqCtx *fasthttp.RequestCtx) {
	cookieValue := GetCookie(m.config.Cookie, reqCtx)
	if cookieValue == "" { // nothing to destroy
		return
	}
	RemoveCookie(m.config.Cookie, reqCtx)
	m.provider.Destroy(cookieValue)
}

// GC tick-tock for the store cleanup
// it's a blocking function, so run it with go routine, it's totally safe
func (m *fasthttpsessions) gc() {
	m.provider.GC(m.config.GcDuration)
	// set a timer for the next GC
	time.AfterFunc(m.config.GcDuration, func() {
		m.gc()
	})
}
