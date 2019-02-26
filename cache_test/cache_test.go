package cache_test

import (
	"net/http"
	"strconv"
	"testing"

	"../cache"
)

func TestCacheInit(t *testing.T) {
	cache := cache.New(10)

	test := cache.Get("google.com")
	if test != nil {
		t.Error("Expected an empty cache with packets in the store.")
	}
}

func TestCacheSet(t *testing.T) {
	// Testing setting google.com
	response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
	}

	cacheItem := &cache.CacheItem{
		Key:    "google.com",
		Packet: response,
	}

	lruCache := cache.New(3)

	lruCache.Set(cacheItem)

	expectedResult := lruCache.Get("google.com")
	if expectedResult != cacheItem {
		t.Error("Expected a matching cache item.")
	}

	// Test adding more than the response to the cache.
	for i := 0; i < 5; i++ {
		item := &cache.CacheItem{
			Key:    "http://test.com/" + strconv.Itoa(i),
			Packet: response,
		}

		lruCache.Set(item)

		expectedResult = lruCache.Get("http://test.com/" + strconv.Itoa(i))
		if expectedResult != item {
			t.Error("Expected a matching cache item after setting.")
		}

		if i == 4 {
			expectedResult = lruCache.Get("http://test.com/0")
			if expectedResult != nil {
				t.Error("Expected item with key http://test.com/0 to be removed from cache to make room for new element.")
			}
		}
	}
}

func TestCacheGet(t *testing.T) {
	// Generic response.
	response := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
	}

	lruCache := cache.New(3)

	// Testing getting two cache items.
	cacheItem0 := &cache.CacheItem{
		Key:    "google.com/0",
		Packet: response,
	}

	cacheItem1 := &cache.CacheItem{
		Key:    "google.com/1",
		Packet: response,
	}

	lruCache.Set(cacheItem0)
	lruCache.Set(cacheItem1)

	retCache := lruCache.Get("google.com/0")
	if retCache != cacheItem0 {
		t.Error("Expected return same cache item 0 after get.")
	}

	retCache = lruCache.Get("google.com/1")
	if retCache != cacheItem1 {
		t.Error("Expected return same cache item 1 after get.")
	}

	// Testing adding 10 elements to a 3 element cache.
	for i := 0; i < 10; i++ {
		cacheItem := &cache.CacheItem{
			Key:    "google.com/" + strconv.Itoa(i),
			Packet: response,
		}

		lruCache.Set(cacheItem)
	}

	// State of cache [7, 8, 9].
	for i := 7; i < 10; i++ {
		result := lruCache.Get("google.com/" + strconv.Itoa(i))
		if result == nil {
			t.Error("Expected cache item to be present after set.")
		}
	}

	result := lruCache.Get("google.com/6")
	if result != nil {
		t.Error("google.com/6 should not be in the cache!")
	}

	// After get 7, new state of cache [8, 9, 7]
	lruCache.Get("google.com/7")

	cacheItem2 := &cache.CacheItem{
		Key:    "google.com/10",
		Packet: response,
	}

	// After set 10, new state of cache [9, 7, 10]
	lruCache.Set(cacheItem2)

	for i := 9; i < 11; i++ {
		result = lruCache.Get("google.com/10")
		if result == nil {
			t.Error("Expected google.com/" + strconv.Itoa(i) + "to be in the cache.")
		}
	}

	result = lruCache.Get("google.com/7")
	if result == nil {
		t.Error("Expected google.com/7 to be in the cache.")
	}

	result = lruCache.Get("google.com/8")
	if result != nil {
		t.Error("Expected that google.com/8 should not be in the cache.")
	}
}
