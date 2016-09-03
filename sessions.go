// Package sessions provides sessions support for net/http
// unique with auto-GC, register unlimited number of databases to Load and Update/Save the sessions in external server or to an external (no/or/and sql) database
// Usage:
// // init a new sessions manager( if you use only one web framework inside your app then you can use the package-level functions like: sessions.Start/sessions.Destroy)
// manager := sessions.New(sessions.Config{})
// // start a session for a particular client
// manager.Start(http.ResponseWriter, *http.Request)
//
// // destroy a session from the server and client,
//  // don't call it on each handler, only on the handler you want the client to 'logout' or something like this:
// manager.Destroy(http.ResponseWriter, *http.Request)
package sessions

import (
	"container/list"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

type (
	// Sessions is the start point of this package
	// contains all the registered sessions and manages them
	//
	// Note: where you see SessionsManager think of this interface
	Sessions interface {
		// UseDatabase ,optionally, adds a session database to the manager's provider,
		// a session db doesn't have write access
		// see https://github.com/kataras/go-sessions/tree/master/sessiondb
		UseDatabase(Database)
		// Start starts the session for the particular request
		Start(http.ResponseWriter, *http.Request) Session
		// Destroy kills the session and remove the associated cookie
		Destroy(http.ResponseWriter, *http.Request)
	}
	// sessions contains the cookie's name, the provider and a duration for GC and cookie life expire
	sessions struct {
		config   Config
		provider *provider
	}
)

// New creates & returns a new Sessions(manager) and start its GC
func New(c Config) Sessions {
	c = c.validate()
	// init and start the sess manager
	sess := &sessions{config: c, provider: &provider{list: list.New(), sessions: make(map[string]*list.Element, 0), databases: make([]Database, 0), expires: c.Expires}}
	//run the GC here
	go sess.gc()
	return sess
}

var defaultSessions = New(Config{
	Cookie:                      DefaultCookieName,
	DecodeCookie:                false,
	Expires:                     DefaultCookieExpires,
	GcDuration:                  DefaultGcDuration,
	DisableSubdomainPersistence: false,
})

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func UseDatabase(db Database) {
	defaultSessions.UseDatabase(db)
}

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func (m *sessions) UseDatabase(db Database) {
	m.provider.expires = m.config.Expires // updae the expires confiuration field for any case
	m.provider.registerDatabase(db)
}

// Start starts the session for the particular request
func Start(res http.ResponseWriter, req *http.Request) Session {
	return defaultSessions.Start(res, req)
}

// Start starts the session for the particular request
func (m *sessions) Start(res http.ResponseWriter, req *http.Request) Session {
	var sess *session

	cookieValue := GetCookie(m.config.Cookie, req)

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := GenerateSessionID()
		sess = m.provider.init(sid)
		//cookie := &http.Cookie{}
		cookie := AcquireCookie()
		// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
		cookie.Name = m.config.Cookie
		cookie.Value = sid
		cookie.Path = "/"
		if !m.config.DisableSubdomainPersistence {

			requestDomain := req.Host
			if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
				requestDomain = requestDomain[0:portIdx]
			}
			if validCookieDomain(requestDomain) {

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
				cookie.Domain = "." + requestDomain // . to allow persistance
			}

		}
		cookie.HttpOnly = true
		if m.config.Expires == 0 {
			// unlimited life
			cookie.Expires = CookieExpireUnlimited
		} else if m.config.Expires > 0 {
			cookie.Expires = time.Now().Add(m.config.Expires)
		} // if it's -1 then the cookie is deleted when the browser closes

		AddCookie(cookie, res)
		ReleaseCookie(cookie)
	} else {
		sess = m.provider.read(cookieValue)
	}
	return sess
}

// Destroy kills the session and remove the associated cookie
func Destroy(res http.ResponseWriter, req *http.Request) {
	defaultSessions.Destroy(res, req)
}

// Destroy kills the session and remove the associated cookie
func (m *sessions) Destroy(res http.ResponseWriter, req *http.Request) {
	cookieValue := GetCookie(m.config.Cookie, req)
	if cookieValue == "" { // nothing to destroy
		return
	}
	RemoveCookie(m.config.Cookie, res, req)
	m.provider.destroy(cookieValue)
}

// GC tick-tock for the store cleanup
// it's a blocking function, so run it with go routine, it's totally safe
func (m *sessions) gc() {
	m.provider.gc(m.config.GcDuration)
	// set a timer for the next GC
	time.AfterFunc(m.config.GcDuration, func() {
		m.gc()
	})
}

// GenerateSessionID returns a random string, used to set the session id
func GenerateSessionID() string {
	return base64.URLEncoding.EncodeToString(Random(32))
}
