package cache

import "time"

//create cache to avoid fetching too much
// s3 list operations can get expensive

type ItemCache struct {
	Cache map[string]CacheVal
	CacheConfig CacheConfig
}

type CacheVal struct {
	Items []string
	Time int64
}

const DEFAULT_CACHE_DURATION = 15

type CacheConfig struct {
	// CacheDuration is the duration that an item will be stored in the cache
	CacheDuration time.Duration
}

func (c ItemCache) Get(key string) ([]string, bool) {
	val, ok := c.Cache[key]
	return val.Items, ok
}

func (c ItemCache) Set(key string, items []string) {
	c.Cache[key] = CacheVal{
		Items: items,
		Time: time.Now().Unix(),
	}
}

func NewCache(config CacheConfig) *ItemCache {
	if config.CacheDuration == 0 {
		config.CacheDuration = DEFAULT_CACHE_DURATION
	}
		
	return &ItemCache{
		Cache: make(map[string]CacheVal),
		CacheConfig: config,
	}
}

// removes all items from cache that are older than currentTime
func (c ItemCache) PurgeAfterTime(currentTime time.Time) {
	for k, v := range c.Cache {
		diff := currentTime.Unix() - v.Time

		if float64(diff) > c.CacheConfig.CacheDuration.Seconds() {
			delete(c.Cache, k)
		}
	}
}
