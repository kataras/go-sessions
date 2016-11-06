package sessions

import (
	"github.com/kataras/go-errors"
	"strconv"
	"sync"
	"time"
)

type (
	// Session is  session's session interface, think it like a local store of each session's values
	// implemented by the internal session iteral, normally the end-user will never use this interface.
	// gettable by the provider -> sessions -> your app
	Session interface {
		ID() string
		Get(string) interface{}
		GetString(key string) string
		GetInt(key string) (int, error)
		GetInt64(key string) (int64, error)
		GetFloat32(key string) (float32, error)
		GetFloat64(key string) (float64, error)
		GetBoolean(key string) (bool, error)
		GetAll() map[string]interface{}
		VisitAll(cb func(k string, v interface{}))
		Set(string, interface{})
		Delete(string)
		Clear()
	}
	// session is an 'object' which wraps the session provider with its session databases, only frontend user has access to this session object.
	// implements the Session interface
	session struct {
		sid              string
		values           map[string]interface{} // here are the real values
		mu               sync.Mutex
		lastAccessedTime time.Time
		createdAt        time.Time
		provider         *Provider
	}
)

// ID returns the session's id
func (s *session) ID() string {
	return s.sid
}

// Get returns the value of an entry by its key
func (s *session) Get(key string) interface{} {
	s.provider.update(s.sid)
	if value, found := s.values[key]; found {
		return value
	}
	return nil
}

// GetString same as Get but returns as string, if nil then returns an empty string
func (s *session) GetString(key string) string {
	if value := s.Get(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}

	}

	return ""
}

var errFindParse = errors.New("Unable to find the %s with key: %s. Found? %#v")

// GetInt same as Get but returns as int, if not found then returns -1 and an error
func (s *session) GetInt(key string) (int, error) {
	v := s.Get(key)
	if vint, ok := v.(int); ok {
		return vint, nil
	} else if vstring, sok := v.(string); sok {
		return strconv.Atoi(vstring)
	}

	return -1, errFindParse.Format("int", key, v)
}

// GetInt64 same as Get but returns as int64, if not found then returns -1 and an error
func (s *session) GetInt64(key string) (int64, error) {
	v := s.Get(key)
	if vint64, ok := v.(int64); ok {
		return vint64, nil
	} else if vint, ok := v.(int); ok {
		return int64(vint), nil
	} else if vstring, sok := v.(string); sok {
		return strconv.ParseInt(vstring, 10, 64)
	}

	return -1, errFindParse.Format("int64", key, v)

}

// GetFloat32 same as Get but returns as float32, if not found then returns -1 and an error
func (s *session) GetFloat32(key string) (float32, error) {
	v := s.Get(key)
	if vfloat32, ok := v.(float32); ok {
		return vfloat32, nil
	} else if vfloat64, ok := v.(float64); ok {
		return float32(vfloat64), nil
	} else if vint, ok := v.(int); ok {
		return float32(vint), nil
	} else if vstring, sok := v.(string); sok {
		vfloat64, err := strconv.ParseFloat(vstring, 32)
		if err != nil {
			return -1, err
		}
		return float32(vfloat64), nil
	}

	return -1, errFindParse.Format("float32", key, v)
}

// GetFloat64 same as Get but returns as float64, if not found then returns -1 and an error
func (s *session) GetFloat64(key string) (float64, error) {
	v := s.Get(key)
	if vfloat32, ok := v.(float32); ok {
		return float64(vfloat32), nil
	} else if vfloat64, ok := v.(float64); ok {
		return vfloat64, nil
	} else if vint, ok := v.(int); ok {
		return float64(vint), nil
	} else if vstring, sok := v.(string); sok {
		return strconv.ParseFloat(vstring, 32)
	}

	return -1, errFindParse.Format("float64", key, v)
}

// GetBoolean same as Get but returns as boolean, if not found then returns -1 and an error
func (s *session) GetBoolean(key string) (bool, error) {
	v := s.Get(key)
	// here we could check for "true", "false" and 0 for false and 1 for true
	// but this may cause unexpected behavior from the developer if they expecting an error
	// so we just check if bool, if yes then return that bool, otherwise return false and an error
	if vb, ok := v.(bool); ok {
		return vb, nil
	}

	return false, errFindParse.Format("bool", key, v)
}

// GetAll returns all session's values
func (s *session) GetAll() map[string]interface{} {
	return s.values
}

// VisitAll loop each one entry and calls the callback function func(key,value)
func (s *session) VisitAll(cb func(k string, v interface{})) {
	for key := range s.values {
		cb(key, s.values[key])
	}
}

// Set fills the session with an entry, it receives a key and a value
// returns an error, which is always nil
func (s *session) Set(key string, value interface{}) {
	s.mu.Lock()
	s.values[key] = value
	s.mu.Unlock()
	s.provider.update(s.sid)
}

// Delete removes an entry by its key
// returns an error, which is always nil
func (s *session) Delete(key string) {
	s.mu.Lock()
	delete(s.values, key)
	s.mu.Unlock()
	s.provider.update(s.sid)
}

// Clear removes all entries
func (s *session) Clear() {
	s.mu.Lock()
	for key := range s.values {
		delete(s.values, key)
	}
	s.mu.Unlock()
	s.provider.update(s.sid)
}
