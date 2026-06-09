package fzf

import (
	"sync"
	"testing"
)

func TestChunkCache(t *testing.T) {
	cache := NewChunkCache()
	chunk1p := &Chunk{}
	chunk2p := &Chunk{count: chunkSize}
	bm1 := ChunkBitmap{1}
	bm2 := ChunkBitmap{1, 2}
	cache.Add(chunk1p, "foo", bm1, 1)
	cache.Add(chunk2p, "foo", bm1, 1)
	cache.Add(chunk2p, "bar", bm2, 2)

	{ // chunk1 is not full
		cached := cache.Lookup(chunk1p, "foo")
		if cached != nil {
			t.Error("Cached disabled for non-full chunks", cached)
		}
	}
	{
		cached := cache.Lookup(chunk2p, "foo")
		if cached == nil || cached[0] != 1 {
			t.Error("Expected bitmap cached", cached)
		}
	}
	{
		cached := cache.Lookup(chunk2p, "bar")
		if cached == nil || cached[1] != 2 {
			t.Error("Expected bitmap cached", cached)
		}
	}
	{
		cached := cache.Lookup(chunk1p, "foobar")
		if cached != nil {
			t.Error("Expected nil cached", cached)
		}
	}
}

func TestChunkBitmapIsAllZero(t *testing.T) {
	var bm ChunkBitmap
	if !bm.IsAllZero() {
		t.Error("Zero bitmap should be all zero")
	}
	bm[0] = 1
	if bm.IsAllZero() {
		t.Error("Non-zero bitmap should not be all zero")
	}
	bm[0] = 0
	bm[chunkBitWords-1] = 1 << 63
	if bm.IsAllZero() {
		t.Error("Last-word non-zero bitmap should not be all zero")
	}
}

func TestChunkCacheConcurrentReads(t *testing.T) {
	cache := NewChunkCache()
	chunk := &Chunk{count: chunkSize}
	bm := ChunkBitmap{1, 2, 3}
	cache.Add(chunk, "query", bm, 3)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				result := cache.Lookup(chunk, "query")
				if result == nil || result[0] != 1 {
					t.Error("Unexpected lookup result")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestAllZeroBitmapSkipsCache(t *testing.T) {
	cache := NewChunkCache()
	chunk := &Chunk{count: chunkSize}

	// Cache an all-zero bitmap for prefix "ab"
	var zeroBm ChunkBitmap
	cache.Add(chunk, "ab", zeroBm, 0)

	// Search for "abc" should find "ab" as prefix and return all-zero bitmap
	result := cache.Search(chunk, "abc")
	if result == nil {
		t.Error("Expected to find prefix bitmap")
		return
	}
	if !result.IsAllZero() {
		t.Error("Expected all-zero bitmap from prefix search")
	}
}
