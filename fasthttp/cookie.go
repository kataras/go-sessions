package sessions

import (
	"github.com/valyala/fasthttp"
	"time"
)

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found
func GetCookie(name string, reqCtx *fasthttp.RequestCtx) (val string) {
	bcookie := reqCtx.Request.Header.Cookie(name)
	if bcookie != nil {
		val = string(bcookie)
	}
	return
}

// AddCookie adds a cookie to the client
func AddCookie(c *fasthttp.Cookie, reqCtx *fasthttp.RequestCtx) {
	reqCtx.Response.Header.SetCookie(c)
}

// RemoveCookie deletes a cookie by it's name/key
func RemoveCookie(name string, reqCtx *fasthttp.RequestCtx) {
	reqCtx.Response.Header.DelCookie(name)

	cookie := fasthttp.AcquireCookie()
	//cookie := &fasthttp.Cookie{}
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	exp := time.Now().Add(-time.Duration(1) * time.Minute) //RFC says 1 second, but let's do it 1 minute to make sure is working...
	cookie.SetExpire(exp)
	AddCookie(cookie, reqCtx)
	fasthttp.ReleaseCookie(cookie)
	// delete request's cookie also, which is temporarly available
	reqCtx.Request.Header.DelCookie(name)
}
