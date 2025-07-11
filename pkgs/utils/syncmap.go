package utils

import "sync"

type SyncMap[K comparable, V any] struct {
	sm sync.Map
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		sm: sync.Map{},
	}
}

func (sm *SyncMap[K, V]) Load(key K) (V, bool) {
	value, ok := sm.sm.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	return value.(V), true
}

func (sm *SyncMap[K, V]) Store(key K, value V) {
	sm.sm.Store(key, value)
}

func (sm *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	existing, loaded := sm.sm.LoadOrStore(key, value)
	if loaded {
		return existing.(V), true
	}
	return value, false
}
