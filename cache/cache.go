package cache

import (
	"container/list"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Cache struct {
	size  int
	items map[string]*CacheItem
	list  *list.List
	Mutex sync.RWMutex
}

type CacheItem struct {
	Key         string
	listElement *list.Element
	Packet      *http.Response
	BodyBytes   []byte
}

func New(size int) *Cache {
	return &Cache{
		size:  size,
		items: make(map[string]*CacheItem, size),
		list:  list.New(),
		Mutex: sync.RWMutex{},
	}
}

func (cache *Cache) Get(key string) *CacheItem {
	item, exists := cache.items[key]

	if !exists {
		return nil
	}
	cache.moveToFront(item)

	return item
}

func (cache *Cache) moveToFront(item *CacheItem) {
	cache.list.MoveToFront(item.listElement)
}

func (cache *Cache) Set(item *CacheItem) {
	// Check that there's enough room in the cache.
	if cache.list.Len() >= cache.size {
		// Clear up some space.
		cache.prune()
	}

	// Check if the item already exists.
	cacheItem, exists := cache.items[item.Key]
	if exists {
		cache.moveToFront(cacheItem)
	} else {
		// Add to cache.
		item.listElement = cache.list.PushFront(item)
		cache.items[item.Key] = item
	}
}

func (cache *Cache) prune() {
	tail := cache.list.Back()
	if tail == nil {
		return
	}

	item := cache.list.Remove(tail).(*CacheItem)
	delete(cache.items, item.Key)
}

func (cache *Cache) ItemIsFresh(item *CacheItem) (bool, error) {
	header := item.Packet.Header
	if header.Get("Cache-Control") != "" && header.Get("Date") != "" {
		cacheControl := header.Get("Cache-Control")
		cacheControl = strings.TrimSpace(cacheControl)
		controlElements := strings.Split(cacheControl, ",")

		for _, val := range controlElements {
			if strings.Contains(val, "max-age") {
				date, err := http.ParseTime(header.Get("Date"))
				if err != nil {
					return false, err
				}
				maxAge, err := strconv.Atoi(strings.Split(val, "=")[1])
				if err != nil {
					return false, err
				}

				if time.Now().Unix() < date.Unix()+int64(maxAge) {
					return true, nil
				}

				return false, nil
			}
		}
	} else if header.Get("Expires") != "" {
		date, err := http.ParseTime(header.Get("Expires"))
		if err != nil {
			return false, err
		}
		currentDate := time.Now()

		if currentDate.Before(date) {
			return true, nil
		}
	}

	// Checking with the server to see if stale packets are still ok.
	if header.Get("ETag") != "" {
		etag := header.Get("Etag")
		client := &http.Client{}

		req, err := http.NewRequest("HEAD", item.Key, nil)
		if err != nil {
			return false, err
		}

		req.Header.Set("If-None-Match", etag)

		resp, err := client.Do(req)

		if resp.StatusCode == http.StatusNotModified {
			return true, nil
		}
	} else if header.Get("Last-Modified") != "" {
		lastModified := header.Get("Last-Modified")
		client := &http.Client{}

		req, err := http.NewRequest("HEAD", item.Key, nil)
		if err != nil {
			return false, err
		}

		req.Header.Set("If-Modified-Since", lastModified)

		resp, err := client.Do(req)

		if resp.StatusCode == http.StatusNotModified {
			return true, nil
		}
	}

	return false, nil
}
