package service

import (
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
)

var (
	// ErrRedisClosed an error with message 'already closed'
	ErrRedisClosed = errors.New("already closed")
	// ErrKeyNotFound a static error when key not found.
	ErrKeyNotFound = errors.New("not found")
)

// Service the Redis service, contains the config and the redis pool
type Service struct {
	// Connected is true when the Service has already connected
	Connected bool
	// Config the redis config for this redis
	Config *Config
	pool   *redis.Pool
}

// PingPong sends a ping and receives a pong, if no pong received then returns false and filled error
func (r *Service) PingPong() (bool, error) {
	c := r.pool.Get()
	defer c.Close()
	msg, err := c.Do("PING")
	if err != nil || msg == nil {
		return false, err
	}
	return (msg == "PONG"), nil
}

// CloseConnection closes the redis connection
func (r *Service) CloseConnection() error {
	if r.pool != nil {
		return r.pool.Close()
	}
	return ErrRedisClosed
}

// Set sets a key-value to the redis store.
// The expiration is setted by the MaxAgeSeconds.
func (r *Service) Set(key, field string, value interface{}, secondsLifetime int64) (err error) {
	c := r.pool.Get()
	defer c.Close()
	if c.Err() != nil {
		return c.Err()
	}

	_, err = c.Do("HSET", r.Config.Prefix+key, field, value)
	if err != nil {
		return err
	}

	// If lifetime is given then expire the map
	if secondsLifetime > 0 {
		_, err = c.Do("EXPIRE", r.Config.Prefix+key, secondsLifetime)
		return err
	}

	return
}

// Get returns value, err by its key
//returns nil and a filled error if something bad happened.
func (r *Service) Get(key, field string) (interface{}, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	redisVal, err := c.Do("HGET", r.Config.Prefix+key, field)
	if err != nil {
		return nil, err
	}
	if redisVal == nil {
		return nil, ErrKeyNotFound
	}

	return redisVal, nil
}

// TTL returns the seconds to expire, if the key has expiration and error if action failed.
// Read more at: https://redis.io/commands/ttl
func (r *Service) TTL(key string) (seconds int64, hasExpiration bool, ok bool) {
	c := r.pool.Get()
	defer c.Close()
	redisVal, err := c.Do("TTL", r.Config.Prefix+key)
	if err != nil {
		return -2, false, false
	}
	seconds = redisVal.(int64)
	// if -1 means the key has unlimited life time.
	hasExpiration = seconds == -1
	// if -2 means key does not exist.
	ok = (c.Err() != nil || seconds == -2)
	return
}

// GetAll returns all redis entries using the "SCAN" command (2.8+).
func (r *Service) GetAll() (interface{}, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	redisVal, err := c.Do("SCAN", 0) // 0 -> cursor
	if err != nil {
		return nil, err
	}
	if redisVal == nil {
		return nil, err
	}

	return redisVal, nil
}

// GetKeys returns all fields in session hash map
func (r *Service) GetKeys(key string) ([]string, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	redisVal, err := c.Do("HKEYS")
	if err != nil {
		return nil, err
	}
	if redisVal == nil {
		return nil, ErrKeyNotFound
	}

	valIfce := redisVal.([]interface{})
	keys := make([]string, len(valIfce))
	for i, v := range valIfce {
		keys[i] = v.(string)
	}

	return keys, nil
}

// GetBytes returns value, err by its key
// you can use utils.Deserialize((.GetBytes("yourkey"),&theobject{})
//returns nil and a filled error if something wrong happens
func (r *Service) GetBytes(key, field string) ([]byte, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	redisVal, err := c.Do("HGET", r.Config.Prefix+key, field)
	if err != nil {
		return nil, err
	}

	if redisVal == nil {
		return nil, ErrKeyNotFound
	}

	return redis.Bytes(redisVal, err)
}

// Delete removes redis entry by specific key
func (r *Service) Delete(key, field string) error {
	c := r.pool.Get()
	defer c.Close()

	_, err := c.Do("HDEL", r.Config.Prefix+key, field)
	return err
}

// DeleteMulti removes multiple fields from hashmap
func (r *Service) DeleteMulti(key string, fields ...string) error {
	c := r.pool.Get()
	defer c.Close()

	// Make list of args for HDEL
	args := make([]interface{}, len(fields)+1)
	args[0] = r.Config.Prefix + key
	for i := range fields {
		args[i+1] = fields[i]
	}

	_, err := c.Do("HDEL", args...)
	return err
}

// DeleteAll deletes session hash map
func (r *Service) DeleteAll(key string) error {
	c := r.pool.Get()
	defer c.Close()

	_, err := c.Do("DEL", r.Config.Prefix+key)
	return err
}

func dial(network string, addr string, pass string) (redis.Conn, error) {
	if network == "" {
		network = DefaultRedisNetwork
	}
	if addr == "" {
		addr = DefaultRedisAddr
	}
	c, err := redis.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	if pass != "" {
		if _, err = c.Do("AUTH", pass); err != nil {
			c.Close()
			return nil, err
		}
	}
	return c, err
}

// Connect connects to the redis, called only once
func (r *Service) Connect() {
	c := r.Config

	if c.IdleTimeout <= 0 {
		c.IdleTimeout = DefaultRedisIdleTimeout
	}

	if c.Network == "" {
		c.Network = DefaultRedisNetwork
	}

	if c.Addr == "" {
		c.Addr = DefaultRedisAddr
	}

	pool := &redis.Pool{IdleTimeout: DefaultRedisIdleTimeout, MaxIdle: c.MaxIdle, MaxActive: c.MaxActive}
	pool.TestOnBorrow = func(c redis.Conn, t time.Time) error {
		_, err := c.Do("PING")
		return err
	}

	if c.Database != "" {
		pool.Dial = func() (redis.Conn, error) {
			red, err := dial(c.Network, c.Addr, c.Password)
			if err != nil {
				return nil, err
			}
			if _, err = red.Do("SELECT", c.Database); err != nil {
				red.Close()
				return nil, err
			}
			return red, err
		}
	} else {
		pool.Dial = func() (redis.Conn, error) {
			return dial(c.Network, c.Addr, c.Password)
		}
	}
	r.Connected = true
	r.pool = pool
}

// New returns a Redis service filled by the passed config
// to connect call the .Connect().
func New(cfg ...Config) *Service {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	r := &Service{pool: &redis.Pool{}, Config: &c}
	return r
}
