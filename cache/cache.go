package cache

import (
	"context"
	"time"

	"github.com/jellydator/ttlcache/v3"

	"github.com/sollie/trackarr/logger"
)

var (
	cache *ttlcache.Cache[string, CacheItem]
	log   = logger.GetLogger("cache")
)

/* Public */
func Close() {
	cache.Stop()
}

func AddItem(key string, value *CacheItem) {
	cache.Set(key, *value, ttlcache.DefaultTTL)
}

func GetItem(key string) (*CacheItem, bool) {
	entry := cache.Get(key, ttlcache.WithDisableTouchOnHit[string, CacheItem]())
	if entry == nil {
		return nil, false
	}

	item := entry.Value()
	return &item, true
}

/* Private */
func Init() error {
	cache = ttlcache.New(ttlcache.WithTTL[string, CacheItem](60 * time.Second))
	cache.OnEviction(cacheItemExpired)

	go cache.Start()
	return nil
}

/* Private - Callbacks */
func cacheItemExpired(_ context.Context, _ ttlcache.EvictionReason, item *ttlcache.Item[string, CacheItem]) {
	log.Tracef("Cleared item: %s", item.Key())
}
