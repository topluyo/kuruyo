package main

import (	
	"net/http"
	"sync"
	"time"
  "io"
  "bytes"
  "strings"
  "strconv"
)


type CacheItem struct {
	Status int
	Header http.Header
	Body   []byte
	Expire time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]CacheItem
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]CacheItem),
	}
}


func (c *Cache) Get(key string) (CacheItem, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(item.Expire) {
		return CacheItem{}, false
	}

	return item, true
}

func (c *Cache) Set(key string, item CacheItem) {
	c.mu.Lock()
	c.items[key] = item
	c.mu.Unlock()
}



func CacheKey(req *http.Request) string {
	return req.URL.Scheme + "://" +
		req.Host +
		req.URL.RequestURI()
}

type CacheTransport struct {
	Transport http.RoundTripper
	Cache     *Cache
}


func (t *CacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	key := req.Method + ":" + req.Host + ":" + req.URL.RequestURI()

	if req.Method == http.MethodGet {

		t.Cache.mu.RLock()
		item, ok := t.Cache.items[key]
		t.Cache.mu.RUnlock()

		if ok && time.Now().Before(item.Expire) {

			return &http.Response{
				StatusCode: item.Status,
				Status:     http.StatusText(item.Status),
				Header:     item.Header.Clone(),
				Body:       io.NopCloser(bytes.NewReader(item.Body)),
				Request:    req,
			}, nil
		}
	}

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if req.Method != http.MethodGet {
		return resp, nil
	}

	cacheControl := resp.Header.Get("Cache-Control")

	var maxAge int

	for _, part := range strings.Split(cacheControl, ",") {
		part = strings.TrimSpace(part)

		if strings.HasPrefix(part, "max-age=") {
			maxAge, _ = strconv.Atoi(
				strings.TrimPrefix(part, "max-age="),
			)
		}
	}

	if maxAge <= 0 {
		return resp, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()

	// Client'a gidecek body
	resp.Body = io.NopCloser(
		bytes.NewReader(body),
	)

	t.Cache.mu.Lock()

	t.Cache.items[key] = CacheItem{
		Status: resp.StatusCode,
		Header: resp.Header.Clone(),
		Body:   body,
		Expire: time.Now().Add(
			time.Duration(maxAge) * time.Second,
		),
	}

	t.Cache.mu.Unlock()

	return resp, nil
}
