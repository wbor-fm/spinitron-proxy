package cache

import (
	"log"
	"net/http"
	"time"

	"github.com/WBOR-91-1-FM/spinitron-proxy/api"
	"github.com/Yiling-J/theine-go"
)

// MAX_CACHE_SIZE determines the maximum number of cache entries that can be
// stored at once.
const MAX_CACHE_SIZE = 2000

// Cache wraps a theine.Cache for storing []byte responses keyed by string.
// Theine is a simple, thread-safe, in-memory cache library. It is used here
// to store responses from the Spinitron API.
type Cache struct {
	tcache *theine.Cache[string, []byte] // Underlying cache from theine-go library.
}

// Initializes the theine cache
func (c *Cache) Init() {
	// If tcache is already set, return immediately and do nothing.
	if c.tcache != nil {
		return
	}

	// Build a theine cache with our maximum size. Provide a RemovalListener
	// to remove all related items from the cache when a collection path expires.
	cache, err := theine.NewBuilder[string, []byte](MAX_CACHE_SIZE).RemovalListener(func(k string, v []byte, r theine.RemoveReason) {
		// RemovalListener is called whenever an item is removed from the cache.
		// We're interested in the RemoveReason, which tells us why the item was
		// removed. We only care about expired items here.

		// When a collection path expires, we also want to remove all associated
		// resources in that collection.
		if api.IsCollectionPath(k) && r == theine.EXPIRED {
			c.evictCollection(api.GetCollectionName(k))
		}
	}).Build()

	if err != nil {
		// If building the cache fails, panic to crash early.
		panic(err)
	}

	// Assign the newly created cache to our struct.
	c.tcache = cache
}

// Get retrieves a value from the cache by key. It returns the value (if found)
// and a boolean to indicate whether the key was present in the cache.
func (c *Cache) Get(key string) ([]byte, bool) {
	tick := time.Now()
	x, y := c.tcache.Get(key)
	log.Println("cache.get", time.Since(tick), key)
	return x, y
}

// Set adds a new key-value pair to the cache with a time-to-live determined by 
// getTTL(key) (defined below). Returns true if set was successful.
// If setting to a key that already exists, the value is updated and the TTL is
// reset (done by the theine library).
func (c *Cache) Set(key string, value []byte) bool {
	tick := time.Now()
	// Theine supports setting entries with a TTL. 
	// The '1' argument is for cost (weight) of the entry, used for cache 
	// eviction strategies. We don't use it here, so it's set to 1 for all
	// entries.
	res := c.tcache.SetWithTTL(key, value, 1, getTTL(key))
	log.Println("cache.set", time.Since(tick), key)
	return res
}

// MakeCacheKey uses request info to build a consistent cache key. If the path 
// is a "collection path," it appends the query parameters.
func (c *Cache) MakeCacheKey(req *http.Request) string {
	result := req.URL.Path

	// If it's a collection path, include query parameters since they may change
	// the data, but skip the `forceRefresh` parameter if present (since it's
	// only used to skip the cache, and a key with `forceRefresh=1` is the same
	// as a key without it - in other words, requests without the param would
	// return potentially old data).
	if api.IsCollectionPath(result) {
		// Copy the query to avoid mutating the original.
        q := req.URL.Query()
        q.Del("forceRefresh") // Remove the param from the key

		// Encode the query parameters and append them to the path.
        encoded := q.Encode()
        if encoded != "" {
            result += "?" + encoded
        }
	}
	return result
}

// getTTL defines how long each type of endpoint is cached. Resource paths and
// collection paths have different time durations.
func getTTL(key string) time.Duration {
	// If it's a resource path, we cache for 3 minutes.
	if api.IsResourcePath(key) {
		return 3 * time.Minute
	}

	// Otherwise, get the collection name and look up its specific TTL.
	c := api.GetCollectionName(key)

	var ttl = map[string]time.Duration{
		"personas":  5 * time.Minute,
		"shows":     5 * time.Minute,
		"playlists": 3 * time.Minute,
		"spins":     30 * time.Second,
	}

	return ttl[c]
}

// evictCollection removes all cached entries from a specific collection.
// This is called when a collection item is removed due to expiration.
func (c *Cache) evictCollection(name string) {
	tick := time.Now()
	// Range over every key in the cache. If the key belongs to the same 
	// collection name, delete that entry. This is done concurrently via a 
	// goroutine (go keyword).
	c.tcache.Range(func(k string, v []byte) bool {
		if api.GetCollectionName(k) == name {
			go c.tcache.Delete(k)
		}
		return true
	})
	log.Println("cache.evicting", time.Since(tick), name)
}

// Len returns the current number of entries in the cache.
func (c *Cache) Len() int {
	return c.tcache.Len()
}