package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kataras/go-sessions/v3"
)

type Config struct {
	Prefix string
}

type Sessions struct {
	client *redis.Client
	config *Config
}

var _ sessions.Database = (*Sessions)(nil)

func (s *Sessions) key(sid string) string {
	return s.config.Prefix + `:` + sid
}

func (s *Sessions) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	ctx := context.Background()
	key := s.key(sid)
	now := time.Now().UTC()
	dur, err := s.client.TTL(ctx, key).Result()
	switch {
	case dur == -1:
		fallthrough
	case dur == -2:
		fallthrough
	case err == redis.Nil:
		if err := s.client.HSet(ctx, key, key, true).Err(); err != nil {
			return sessions.LifeTime{Time: sessions.CookieExpireDelete}
		}
		if err := s.client.Expire(ctx, key, expires).Err(); err != nil {
			return sessions.LifeTime{Time: sessions.CookieExpireDelete}
		}
		return sessions.LifeTime{}
	case err != nil:
		return sessions.LifeTime{Time: sessions.CookieExpireDelete}
	default:
		return sessions.LifeTime{Time: now.Add(dur)}
	}
}

func (s *Sessions) OnUpdateExpiration(sid string, expiration time.Duration) error {
	ctx := context.Background()
	key := s.key(sid)
	if err := s.client.Expire(ctx, key, expiration).Err(); err != nil {
		return err
	}

	return nil
}

func (s *Sessions) Set(sid string, lifetime sessions.LifeTime, field string, value interface{}, immutable bool) {
	ctx := context.Background()
	key := s.key(sid)
	if immutable {
		if err := s.client.HSetNX(ctx, key, field, value).Err(); err != nil {
			return
		}
	} else {
		if err := s.client.HSet(ctx, key, field, value).Err(); err != nil {
			return
		}
	}

	_ = s.OnUpdateExpiration(sid, lifetime.DurationUntilExpiration())
}

func (s *Sessions) Get(sid string, field string) interface{} {
	ctx := context.Background()
	key := s.key(sid)
	val, err := s.client.HGet(ctx, key, field).Result()
	switch {
	case err == redis.Nil:
		return nil
	case err != nil:
		return nil
	default:
		return val
	}
}

func (s *Sessions) Visit(sid string, cb func(field string, value interface{})) {
	ctx := context.Background()
	key := s.key(sid)
	vals, err := s.client.HGetAll(ctx, key).Result()
	switch {
	case err == redis.Nil:
		return
	case err != nil:
		return
	default:
		for field, value := range vals {
			cb(field, value)
		}
	}

}

func (s *Sessions) Len(sid string) int {
	ctx := context.Background()
	key := s.key(sid)
	l, err := s.client.HLen(ctx, key).Result()
	if err != nil {
		return 0
	}

	// Decrement by one because we store our own ID key
	return int(l - 1)
}

func (s *Sessions) Delete(sid string, field string) bool {
	ctx := context.Background()
	key := s.key(sid)
	if err := s.client.HDel(ctx, key, field).Err(); err != nil {
		return false
	}

	return true
}

func (s *Sessions) Clear(sid string) {
	ctx := context.Background()
	key := s.key(sid)
	fields, err := s.client.HKeys(ctx, key).Result()
	if err != nil {
		return
	}

	// Remove our special key from the fields to delete
	for i, f := range fields {
		if f == key {
			fields = append(fields[:i], fields[i+1:]...)
			break
		}
	}

	if len(fields) > 0 {
		if err := s.client.HDel(ctx, key, fields...).Err(); err != nil {
		}
	}
}

func (s *Sessions) Release(sid string) {
	ctx := context.Background()
	key := s.key(sid)
	if err := s.client.Del(ctx, key).Err(); err != nil {
	}
}

func (s *Sessions) Close() error {
	return s.client.Close()
}

func NewSessions(options *redis.Options, config *Config) *Sessions {
	client := redis.NewClient(options)

	return &Sessions{
		client: client,
		config: config,
	}
}
