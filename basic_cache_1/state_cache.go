package main

import (
	"context"
	"log"
	"sync"
	"time"
)

type CachedItem struct {
	StateObject *MyState
	CachedAt    int64 // unix time
	ExpiresAt   int64 // unix time
}

type MyStateCache struct {
	sync.RWMutex
	items  map[string]*CachedItem
	ctx    context.Context
	cancel context.CancelFunc
}

func NewMyStateCache(ctx context.Context) *MyStateCache {
	cacheCtx, cancel := context.WithCancel(ctx)
	cache := &MyStateCache{
		items:  make(map[string]*CachedItem),
		ctx:    cacheCtx,
		cancel: cancel,
	}
	go cache.startCleanup()
	return cache
}

func (cache *MyStateCache) Set(state *MyState, lifespan time.Duration) {
	cache.Lock()
	defer cache.Unlock()

	cachedAt := time.Now().Unix()
	expiry := cachedAt + int64(lifespan.Seconds())

	cacheItem := CachedItem{
		StateObject: state,
		CachedAt:    cachedAt,
		ExpiresAt:   expiry,
	}

	cache.items[state.Id] = &cacheItem
}

func (cache *MyStateCache) Get(stateId string) *MyState {
	cache.RLock()
	defer cache.RUnlock()

	item, exists := cache.items[stateId]
	if !exists {
		return nil
	}

	if item.ExpiresAt <= time.Now().Unix() {
		return nil
	}

	return item.StateObject
}

func (cache *MyStateCache) Shutdown() {
	log.Print("shutting down cache...")
	cache.cancel()
}

func (cache *MyStateCache) startCleanup() {
	ticker := time.NewTicker(15 * time.Second)
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

	now := time.Now().Unix()
	log.Printf("cleaning for expiries older than %v", now)

	for id, item := range cache.items {
		if item.ExpiresAt <= now {
			delete(cache.items, id)
			log.Printf("deleted item %v\n", id)
		}
	}
	log.Print("cache cleanup completed")
}
