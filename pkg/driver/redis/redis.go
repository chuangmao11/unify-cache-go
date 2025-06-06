package redis

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	mutex  sync.RWMutex
	data   map[string]*item
	client redis.Cmdable
}

func NewRedisCache(client redis.Cmdable) *RedisCache {
	return &RedisCache{
		client: client,
		data:   make(map[string]*item),
	}
}

func (rc *RedisCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	item := &item{
		val:      val,
		deadline: time.Now().Add(expiration),
	}
	rc.data[key] = item

	rc.client.Set(ctx, key, val, expiration)

	return nil
}

func (rc *RedisCache) Get(ctx context.Context, key string) (any, error) {
	rc.mutex.RLock()
	res, loaded := rc.data[key]
	rc.mutex.RUnlock()
	if !loaded {
		return nil, ErrNotFound
	}

	now := time.Now()
	if res.deadlineBefore(now) {
		rc.mutex.Lock()
		defer rc.mutex.Unlock()
		rc.delete(key)
		return nil, ErrNotFound
	}

	return res.val, nil
}

func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	rc.delete(key)
	rc.client.Del(ctx, key)
	return nil
}

func (rc *RedisCache) delete(key string) {
	_, loaded := rc.data[key]
	if !loaded {
		return
	}
	delete(rc.data, key)
}

type item struct {
	val      any
	deadline time.Time
}

func (i *item) deadlineBefore(t time.Time) bool {
	return i.deadline.Before(t)
}
