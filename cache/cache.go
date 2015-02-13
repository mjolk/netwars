package cache

import (
	"appengine"
	"appengine/memcache"
	"fmt"
	"time"
)

const (
	timeKey = "cachetime"
	expiry  = 600 // 10 minutes
)

func newTime() uint64 { return uint64(time.Now().Unix()) << 32 }

// Now returns the current logical datastore time to use for cache lookups.
func Now(c appengine.Context) uint64 {
	t, err := memcache.Increment(c, timeKey, 0, newTime())
	if err != nil {
		c.Errorf("cache.Now: %s", err)
		return 0
	}
	return t
}

// Tick sets the current logical datastore time to a never-before-used time
// and returns that time. It should be called to invalidate the cache.
func Tick(c appengine.Context) uint64 {
	t, err := memcache.Increment(c, timeKey, 1, newTime())
	if err != nil {
		c.Errorf("cache.Tick: %s", err)
		return 0
	}
	return t
}

func MemKey(c appengine.Context, keyStr string) string {
	return fmt.Sprintf("%s.%d", keyStr, Now(c))
}

func Delete(c appengine.Context, keyStr string) {
	memKey := MemKey(c, keyStr)
	if err := memcache.Delete(c, memKey); err != nil {
		c.Errorf("Error deleting cache item")
	}
}

func Get(c appengine.Context, keyStr string, strct interface{}) bool {
	memKey := MemKey(c, keyStr)
	_, err := memcache.JSON.Get(c, memKey, strct)
	if err == memcache.ErrCacheMiss {
		return false
	} else if err != nil {
		c.Errorf("Error: get memcache failed")
	}
	return true
}

func Add(c appengine.Context, keyStr string, strct interface{}) {
	// Add the item to the memcache, if the key does not already exist]
	memKey := MemKey(c, keyStr)
	item := &memcache.Item{
		Key:        memKey,
		Object:     strct,
		Expiration: expiry,
	}
	err := memcache.JSON.Add(c, item)
	if err == memcache.ErrNotStored {
		c.Infof("item with key %q already exists", item.Key)
	} else if err != nil {
		c.Errorf("error adding item: %s", err)
	}
}

func Set(c appengine.Context, keyStr string, strct interface{}) {
	memKey := MemKey(c, keyStr)
	err := memcache.JSON.Set(c, &memcache.Item{
		Key:        memKey,
		Object:     strct,
		Expiration: expiry,
	})
	if err != nil {
		c.Errorf("error setting memcache item: %s", err)
	}
}
