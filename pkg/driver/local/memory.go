package local

import (
	"context"
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
)

type BuildInLocalCache struct {
	mutex  sync.RWMutex
	data   map[string]*item
	client *cache.Cache
}

func NewBuildInLocalCache() *BuildInLocalCache {
	return &BuildInLocalCache{
		data:   make(map[string]*item),
		client: cache.New(5*time.Minute, 10*time.Minute), // 设置默认过期时间和清理间隔
	}
}

func (blc *BuildInLocalCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	blc.mutex.Lock()
	defer blc.mutex.Unlock()

	item := &item{
		val:      val,
		deadline: time.Now().Add(expiration),
	}

	blc.data[key] = item

	blc.client.Set(key, val, expiration)

	return nil
}

func (blc *BuildInLocalCache) Get(ctx context.Context, key string) (any, error) {
	blc.mutex.RLock()
	res, loaded := blc.data[key]
	blc.mutex.RUnlock()
	if !loaded {
		return nil, ErrNotFound
	}
	now := time.Now()
	if res.deadlineBefore(now) {
		blc.mutex.Lock()
		defer blc.mutex.Unlock()
		res, loaded = blc.data[key]
		if !loaded {
			return nil, ErrNotFound
		}
		if res.deadlineBefore(now) {
			blc.delete(key)
			return nil, ErrNotFound
		}
	}
	return res.val, nil
}

func (blc *BuildInLocalCache) Delete(ctx context.Context, key string) error {
	blc.mutex.Lock()
	defer blc.mutex.Unlock()
	blc.delete(key)
	return nil
}

func (blc *BuildInLocalCache) delete(key string) {
	_, loaded := blc.data[key]
	if !loaded {
		return
	}
	delete(blc.data, key)
}

func (i *item) deadlineBefore(t time.Time) bool {
	return !i.deadline.IsZero() && i.deadline.Before(t)
}

type item struct {
	val      any
	deadline time.Time
}
