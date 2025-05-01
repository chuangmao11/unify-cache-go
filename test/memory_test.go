package test

import (
	"context"
	"testing"
	"time"

	"unify-cache-go/pkg/driver/local"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInLocalCache_Set(t *testing.T) {
	cache := local.NewBuildInLocalCache()
	ctx := context.Background()

	testCases := []struct {
		name       string
		key        string
		value      any
		expiration time.Duration
		wantErr    bool
	}{
		{
			name:       "normal string value",
			key:        "test_string",
			value:      "test_value",
			expiration: time.Minute,
			wantErr:    false,
		},
		{
			name:       "normal int value",
			key:        "test_int",
			value:      123,
			expiration: time.Hour,
			wantErr:    false,
		},
		{
			name:       "empty key",
			key:        "",
			value:      "value",
			expiration: time.Hour,
			wantErr:    false,
		},
		{
			name:       "zero expiration",
			key:        "test_key",
			value:      "value",
			expiration: 0,
			wantErr:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := cache.Set(ctx, tc.key, tc.value, tc.expiration)
			if (err != nil) != tc.wantErr {
				t.Errorf("expected error: %v, but got: %v", tc.wantErr, err)
				return
			}
		})
	}
}

func TestBuildInLocalCache_Set_Concurrent(t *testing.T) {
	cache := local.NewBuildInLocalCache()
	ctx := context.Background()
	key := "test_key"
	value := "test_value"
	expiration := time.Hour

	concurrency := 100
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go func() {
			err := cache.Set(ctx, key, value, expiration)
			if err != nil {
				t.Errorf("expected error: %v, but got: %v", nil, err)
			}
			done <- true
		}()
	}
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestBuildInLocalCache_Set_Expiration(t *testing.T) {
	cache := local.NewBuildInLocalCache()
	ctx := context.Background()
	key := "expiration_test"
	value := "test_value"

	Expiration := time.Millisecond * 100
	err := cache.Set(ctx, key, value, Expiration)
	if err != nil {
		t.Errorf("expected error: %v, but got: %v", nil, err)
	}
	time.Sleep(Expiration)
}

func TestBuildInLocalCache_Get(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		key     string
		cache   func() *local.BuildInLocalCache
		wantVal any
		wantErr error
	}{
		{
			name: "key not found",
			key:  "not exist key",
			cache: func() *local.BuildInLocalCache {
				return local.NewBuildInLocalCache()
			},
			wantErr: local.ErrNotFound,
		},
		{
			name: "get value",
			key:  "test_key",
			cache: func() *local.BuildInLocalCache {
				cache := local.NewBuildInLocalCache()
				err := cache.Set(ctx, "test_key", "test_value", time.Minute)
				require.NoError(t, err)
				return cache
			},
			wantVal: "test_value",
		},
		{
			name: "expired",
			key:  "expired key",
			cache: func() *local.BuildInLocalCache {
				cache := local.NewBuildInLocalCache()
				err := cache.Set(ctx, "expired key", "expired value", time.Second)
				require.NoError(t, err)
				time.Sleep(time.Second * 2)
				return cache
			},
			wantErr: local.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := tc.cache().Get(ctx, tc.key)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantVal, val)
		})
	}
}

func TestBuildInLocalCache_Delete(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name    string
		key     string
		cache   func() *local.BuildInLocalCache
		wantErr error
	}{
		{
			name: "key not found",
			key:  "test_key",
			cache: func() *local.BuildInLocalCache {
				cache := local.NewBuildInLocalCache()
				err := cache.Set(ctx, "test_key", "test_value", time.Minute)
				require.NoError(t, err)
				return cache
			},
			wantErr: nil,
		},
		{
			name: "delete non-existent key",
			key:  "non_existent_key",
			cache: func() *local.BuildInLocalCache {
				return local.NewBuildInLocalCache()
			},
			wantErr: nil,
		},
		{
			name: "delete expired key",
			key:  "expired_key",
			cache: func() *local.BuildInLocalCache {
				cache := local.NewBuildInLocalCache()
				err := cache.Set(ctx, "expired_key", "expired_value", time.Second)
				require.NoError(t, err)
				time.Sleep(time.Second * 2)
				return cache
			},
			wantErr: nil, // 删除已过期的键应该不返回错误
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := tc.cache()
			err := cache.Delete(ctx, tc.key)
			assert.Equal(t, tc.wantErr, err)

			// 验证键确实被删除
			val, err := cache.Get(ctx, tc.key)
			assert.Equal(t, local.ErrNotFound, err)
			assert.Nil(t, val)
		})
	}
}

func TestBuildInLocalCache_Delete_Concurrent(t *testing.T) {
	ctx := context.Background()
	cache := local.NewBuildInLocalCache()
	key := "concurrent_key"
	value := "concurrent_value"

	// 先设置一个值
	err := cache.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// 并发删除测试
	concurrency := 100
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go func() {
			err := cache.Delete(ctx, key)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// 验证键确实被删除
	val, err := cache.Get(ctx, key)
	assert.Equal(t, local.ErrNotFound, err)
	assert.Nil(t, val)
}
