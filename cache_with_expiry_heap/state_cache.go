package main

import (
	"container/heap"
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

type cachedItem struct {
	stateObject *MyState
	cachedAt    int64 // unix time
	expiresAt   int64 // unix time
}

type itemExpiry struct {
	itemKey        string
	unixExpiryTime int64
	index          int
}

type expirationQueue []*itemExpiry

func (q *expirationQueue) Len() int {
	return len(*q)
}
func (q *expirationQueue) Less(i, j int) bool {
	return (*q)[i].unixExpiryTime < (*q)[j].unixExpiryTime
}
func (q *expirationQueue) Swap(i, j int) {
	(*q)[i], (*q)[j] = (*q)[j], (*q)[i]
	(*q)[i].index = i
	(*q)[j].index = j
}
func (q *expirationQueue) Push(x interface{}) {
	n := len(*q)
	item := x.(*itemExpiry)
	item.index = n
	*q = append(*q, item)
}
func (q *expirationQueue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // allow for eventual GC
	item.index = -1 // help prevent accidental re-use
	*q = old[0 : n-1]
	return item
}

type MyStateCache struct {
	sync.RWMutex
	items       map[string]*cachedItem
	expirations expirationQueue        // min-heap to track item expirations
	expiryMap   map[string]*itemExpiry // track expiry entries for updates
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewMyStateCache(ctx context.Context) *MyStateCache {
	cacheCtx, cancel := context.WithCancel(ctx)
	cache := &MyStateCache{
		items:       make(map[string]*cachedItem),
		expirations: make(expirationQueue, 0),
		expiryMap:   make(map[string]*itemExpiry),
		ctx:         cacheCtx,
		cancel:      cancel,
	}
	heap.Init(&cache.expirations)
	go cache.startCleanup()
	return cache
}

func (cache *MyStateCache) Set(state *MyState, lifespan time.Duration) error {
	if state == nil {
		return errors.New("cannot cache state due to nil value")
	}

	cache.Lock()
	defer cache.Unlock()

	cachedAt := time.Now().Unix()
	expiry := cachedAt + int64(lifespan.Seconds())

	if oldExpiry, exists := cache.expiryMap[state.Id]; exists {
		oldExpiry.unixExpiryTime = expiry
		heap.Fix(&cache.expirations, oldExpiry.index)
	} else {
		expiryEntry := &itemExpiry{
			itemKey:        state.Id,
			unixExpiryTime: expiry,
		}
		cache.expiryMap[state.Id] = expiryEntry
		heap.Push(&cache.expirations, expiryEntry)
	}

	cache.items[state.Id] = &cachedItem{
		stateObject: state,
		cachedAt:    cachedAt,
		expiresAt:   expiry,
	}

	return nil
}

func (cache *MyStateCache) Get(stateId string) (*MyState, error) {
	cache.RLock()
	defer cache.RUnlock()

	item, exists := cache.items[stateId]
	if !exists {
		return nil, errors.New("state item not found")
	}

	if item.expiresAt <= time.Now().Unix() {
		return nil, errors.New("state item was found as expired")
	}

	return item.stateObject, nil
}

func (cache *MyStateCache) Shutdown() {
	log.Print("shutting down cache...")
	cache.RLock()
	defer cache.RUnlock()
	cache.items = nil
	cache.expirations = make(expirationQueue, 0)
	cache.expiryMap = make(map[string]*itemExpiry)
	cache.cancel()
}

func (cache *MyStateCache) startCleanup() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cache.clean()
		case <-cache.ctx.Done():
			log.Println("cache cleanup stopped")
			return
		}
	}
}

func (cache *MyStateCache) clean() {
	cache.Lock()
	defer cache.Unlock()

	now := time.Now()
	log.Printf("cleaning for expiries older than %s", now.Format("02/01/2006 15:04:05"))

	for cache.expirations.Len() > 0 {
		earliest := cache.expirations[0] // Peek
		if earliest.unixExpiryTime > now.Unix() {
			break
		}
		heap.Pop(&cache.expirations)          // remove from heap
		delete(cache.items, earliest.itemKey) // remove from map
		log.Printf("deleted item %v\n", earliest.itemKey)
	}
	log.Print("cache cleanup completed")
}
