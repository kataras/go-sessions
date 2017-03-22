package sessions

import (
	"encoding/base64"
	"time"
)

const (
	// DefaultCookieName the secret cookie's name for sessions
	DefaultCookieName = "gosessionsid"
	// DefaultCookieLength is the default Session Manager's CookieLength, which is 32
	DefaultCookieLength = 32
)

// Config the configuration for sessions
// has 5 fields
// first is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
// second enable if you want to decode the cookie's key also
// third is the time which the client's cookie expires
// forth is the cookie length (sessionid) int, defaults to 32, do not change if you don't have any reason to do
// fifth is the DisableSubdomainPersistence which you can set it to true in order dissallow your q subdomains to have access to the session cook
type (
	// OptionSetter used as the type of return of a func which sets a configuration field's value
	OptionSetter interface {
		// Set receives a pointer to the Config type and does the job of filling it
		Set(c *Config)
	}
	// OptionSet implements the OptionSetter
	OptionSet func(c *Config)

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
		DisableSubdomainPersistence bool
	}
)

// Set implements the OptionSetter
func (c Config) Set(main *Config) {
	c = c.Validate()
	*main = c
}

// Set is the func which makes the OptionSet an OptionSetter, this is used mostly
func (o OptionSet) Set(c *Config) {
	o(c)
}

// Cookie string, the session's client cookie name, for example: "mysessionid"
//
// Defaults to "gosessionid"
func Cookie(val string) OptionSet {
	return func(c *Config) {
		c.Cookie = val
	}
}

// DecodeCookie if setted to true then the cookie's name will be url-encoded.
// Note: Users should not confuse the 'Encode' and 'Decode' configuration fields,
// these are for the cookie's value, which is the session id.
//
// Defaults to false
func DecodeCookie(val bool) OptionSet {
	return func(c *Config) {
		c.DecodeCookie = val
	}
}

// Encode sets the encode func for the cookie value.
// Should accept as first argument the cookie name (config.Name)
//         as second argument the server's generated session id.
// Should return the new session id, if error the session id setted to empty which is invalid.
//
// Note: Errors are not printed, so you have to know what you're doing,
// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
// You either need to provide exactly that amount or you derive the key from what you type in.
//
// Defaults to nil
func Encode(val func(cookieName string, value interface{}) (string, error)) OptionSet {
	return func(c *Config) {
		c.Encode = val
	}
}

// Decode  sets the decode func for the cookie value.
// Should accept as first argument the cookie name (config.Name)
//               as second second accepts the client's cookie value (the encoded session id).
// Should return an error if decode operation failed.
//
// Note: Errors are not printed, so you have to know what you're doing,
// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
// You either need to provide exactly that amount or you derive the key from what you type in.
//
// Defaults to nil
func Decode(val func(cookieName string, cookieValue string, v interface{}) error) OptionSet {
	return func(c *Config) {
		c.Decode = val
	}
}

// Expires the duration of which the cookie must expires (created_time.Add(Expires)).
// If you want to delete the cookie when the browser closes, set it to -1 but in this case, the server side's session duration is up to GcDuration
//
// Defaults to infinitive/unlimited life duration(0)
func Expires(val time.Duration) OptionSet {
	return func(c *Config) {
		c.Expires = val
	}
}

// CookieLength the length of the sessionid's cookie's value, let it to 0 if you don't want to change it
//
// Defaults to 32
func CookieLength(val int) OptionSet {
	return func(c *Config) {
		c.CookieLength = val
	}
}

// DisableSubdomainPersistence set it to true in order dissallow your q subdomains to have access to the session cookie
//
// Defaults to false
func DisableSubdomainPersistence(val bool) OptionSet {
	return func(c *Config) {
		c.DisableSubdomainPersistence = val
	}
}

// Validate corrects missing fields configuration fields and returns the right configuration
func (c Config) Validate() Config {

	if c.Cookie == "" {
		c.Cookie = DefaultCookieName
	}

	if c.DecodeCookie {
		// just the cookie name
		// use 'Encode' and 'Decode' for the session id "safety".
		c.Cookie = base64.URLEncoding.EncodeToString([]byte(c.Cookie))
		// get the real value for your tests by:
		//sessIdKey := url.QueryEscape(base64.URLEncoding.EncodeToString([]byte(Sessions.Cookie)))
	}

	if c.CookieLength <= 0 {
		c.CookieLength = DefaultCookieLength
	}

	return c
}
