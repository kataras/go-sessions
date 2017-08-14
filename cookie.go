package sessions

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	// CookieExpireDelete may be set on Cookie.Expire for expiring the given cookie.
	CookieExpireDelete = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	// CookieExpireUnlimited indicates that the cookie doesn't expire.
	CookieExpireUnlimited = time.Now().AddDate(24, 10, 10)
)

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found
func GetCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}

	return c.Value
}

// GetCookieFasthttp returns cookie's value by it's name
// returns empty string if nothing was found.
func GetCookieFasthttp(ctx *fasthttp.RequestCtx, name string) (value string) {
	bcookie := ctx.Request.Header.Cookie(name)
	if bcookie != nil {
		value = string(bcookie)
	}
	return
}

// AddCookie adds a cookie.
func AddCookie(w http.ResponseWriter, cookie *http.Cookie) {
	http.SetCookie(w, cookie)
}

// AddCookieFasthttp adds a cookie.
func AddCookieFasthttp(ctx *fasthttp.RequestCtx, cookie *fasthttp.Cookie) {
	ctx.Response.Header.SetCookie(cookie)
}

// RemoveCookie deletes a cookie by it's name/key.
func RemoveCookie(w http.ResponseWriter, r *http.Request, name string) {
	c, err := r.Cookie(name)
	if err != nil {
		return
	}

	c.Expires = CookieExpireDelete
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	c.MaxAge = -1
	c.Value = ""
	c.Path = "/"
	AddCookie(w, c)
}

// RemoveCookieFasthttp deletes a cookie by it's name/key.
func RemoveCookieFasthttp(ctx *fasthttp.RequestCtx, name string) {
	ctx.Response.Header.DelCookie(name)

	cookie := fasthttp.AcquireCookie()
	//cookie := &fasthttp.Cookie{}
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	exp := time.Now().Add(-time.Duration(1) * time.Minute) //RFC says 1 second, but let's do it 1 minute to make sure is working...
	cookie.SetExpire(exp)
	AddCookieFasthttp(ctx, cookie)
	fasthttp.ReleaseCookie(cookie)
	// delete request's cookie also, which is temporary available
	ctx.Request.Header.DelCookie(name)
}

// IsValidCookieDomain returns true if the receiver is a valid domain to set
// valid means that is recognised as 'domain' by the browser, so it(the cookie) can be shared with subdomains also
func IsValidCookieDomain(domain string) bool {
	if domain == "0.0.0.0" || domain == "127.0.0.1" {
		// for these type of hosts, we can't allow subdomains persistence,
		// the web browser doesn't understand the mysubdomain.0.0.0.0 and mysubdomain.127.0.0.1 mysubdomain.32.196.56.181. as scorrectly ubdomains because of the many dots
		// so don't set a cookie domain here, let browser handle this
		return false
	}

	dotLen := strings.Count(domain, ".")
	if dotLen == 0 {
		// we don't have a domain, maybe something like 'localhost', browser doesn't see the .localhost as wildcard subdomain+domain
		return false
	}
	if dotLen >= 3 {
		if lastDotIdx := strings.LastIndexByte(domain, '.'); lastDotIdx != -1 {
			// chekc the last part, if it's number then propably it's ip
			if len(domain) > lastDotIdx+1 {
				_, err := strconv.Atoi(domain[lastDotIdx+1:])
				if err == nil {
					return false
				}
			}
		}
	}

	return true
}
