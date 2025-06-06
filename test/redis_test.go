package test

import (
	"context"
	"testing"
	"time"
	rd "unify-cache-go/pkg/driver/redis"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupRedisClient() *redis.Client {
	// 创建一个连接到本地 Redis 服务器的客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis 服务器地址
		Password: "",               // 如果有密码，在这里设置
		DB:       0,                // 使用默认 DB
	})

	// 测试连接
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}

	return client
}

func TestRedisCache_Set(t *testing.T) {
	client := setupRedisClient()
	defer client.Close()

	testCases := []struct {
		name       string
		key        string
		value      any
		expiration time.Duration
		wantErr    bool
	}{
		{
			name:       "set valid value",
			key:        "test-key",
			value:      "test-value",
			expiration: time.Hour,
			wantErr:    false,
		},
		{
			name:       "zero expiration",
			key:        "test-key-zero",
			value:      "test-value-zero",
			expiration: 0,
			wantErr:    false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cache := rd.NewRedisCache(client)
			ctx := context.Background()

			err := cache.Set(ctx, tt.key, tt.value, tt.expiration)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// 验证值是否真的被设置到 Redis 中
			val, err := client.Get(ctx, tt.key).Result()
			if !tt.wantErr {
				assert.NoError(t, err)
				assert.Equal(t, tt.value, val)
			}
		})
	}
}

func TestRedisCache_Get(t *testing.T) {
	client := setupRedisClient()
	defer client.Close()

	testCases := []struct {
		name       string
		key        string
		value      any
		expiration time.Duration
		wantValue  any
		wantErr    error
	}{
		{
			name:       "get existing value",
			key:        "test-key-get",
			value:      "get-value",
			expiration: time.Hour,
			wantValue:  "get-value",
			wantErr:    nil,
		},
		{
			name:       "get non-existent key",
			key:        "non-existent-key",
			value:      nil,
			expiration: 0,
			wantValue:  nil,
			wantErr:    rd.ErrNotFound,
		},
		{
			name:       "get expired value",
			key:        "expired-key",
			value:      "expired-value",
			expiration: time.Millisecond,
			wantValue:  nil,
			wantErr:    rd.ErrNotFound,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cache := rd.NewRedisCache(client)
			ctx := context.Background()

			if tt.value != nil {
				err := cache.Set(ctx, tt.key, tt.value, tt.expiration)
				assert.NoError(t, err)

				if tt.name == "get expired value" {
					time.Sleep(2 * time.Millisecond)
					// 确保 Redis 中的键也过期
					client.Del(ctx, tt.key)
				}
			}

			gotValue, err := cache.Get(ctx, tt.key)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValue, gotValue)
			}
		})
	}
}

func TestRedisCache_Delete(t *testing.T) {
	client := setupRedisClient()
	defer client.Close()

	testCases := []struct {
		name    string
		key     string
		value   any
		wantErr bool
	}{
		{
			name:    "delete existing key",
			key:     "delete-key",
			value:   "value-to-delete",
			wantErr: false,
		},
		{
			name:    "delete non-existent key",
			key:     "non-existent-delete",
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cache := rd.NewRedisCache(client)
			ctx := context.Background()

			if tt.value != nil {
				err := cache.Set(ctx, tt.key, tt.value, time.Hour)
				assert.NoError(t, err)
			}

			err := cache.Delete(ctx, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// 验证键是否真的被删除
			_, err = client.Get(ctx, tt.key).Result()
			assert.Error(t, err)
			assert.Equal(t, redis.Nil, err)

			// 验证通过 cache.Get 也无法获取到已删除的键
			_, err = cache.Get(ctx, tt.key)
			assert.ErrorIs(t, err, rd.ErrNotFound)
		})
	}
}
