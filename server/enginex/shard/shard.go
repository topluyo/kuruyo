package shard

import ( "sync" )

// Shard yapısı
type Shard[T any] struct {
	size   int
	shards []map[uint64]*T
	mutex  []sync.RWMutex
}

// NewShard oluşturucu
// size = 2^size kadar shard oluşturur
func New[T any](size int) *Shard[T] {
	shardCount := 1 << size
	s := &Shard[T]{
		size:   size,
		shards: make([]map[uint64]*T, shardCount),
		mutex:  make([]sync.RWMutex, shardCount),
	}

	for i := 0; i < shardCount; i++ {
		s.shards[i] = make(map[uint64]*T)
	}

	return s
}

// getShardIndex: key’in üst 'size' bitini alarak shard index’i döndürür
func (s *Shard[T]) getShardIndex(key uint64) int {
	return int(key >> (64 - s.size)) // key’in üst size kadar bitini al
}

// Add: Veriyi ekler
func (s *Shard[T]) Add(key uint64, val T) {
	idx := s.getShardIndex(key)
	s.mutex[idx].Lock()
	s.shards[idx][key] = &val
	s.mutex[idx].Unlock()
}

// Get: Veriyi getirir
func (s *Shard[T]) Get(key uint64) (*T, bool) {
	idx := s.getShardIndex(key)
	s.mutex[idx].RLock()
	val, ok := s.shards[idx][key]
	s.mutex[idx].RUnlock()
	return val, ok
}

func (s *Shard[T]) Find(key uint64) *T {
	idx := s.getShardIndex(key)

	// Önce hızlıca read lock ile kontrol
	s.mutex[idx].RLock()
	val, ok := s.shards[idx][key]
	s.mutex[idx].RUnlock()
	if ok {
		return val
	}

	// Eğer yoksa boş T oluştur
	var v T

	// Write lock ile ekle
	s.mutex[idx].Lock()
	defer s.mutex[idx].Unlock()

	// double-check
	if val, ok := s.shards[idx][key]; ok {
		return val
	}

	s.shards[idx][key] = &v
	return &v
}


func (s *Shard[T]) Reset(shardIndex int) {
	if shardIndex < 0 || shardIndex >= (1<<s.size) {
		return
	}
	s.mutex[shardIndex].Lock()
	s.shards[shardIndex] = make(map[uint64]*T)
	s.mutex[shardIndex].Unlock()
}




