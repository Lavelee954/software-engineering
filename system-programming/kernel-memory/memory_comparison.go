package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// 일반적인 메모리 할당 (커널 메모리 관리 개념 미반영)
type SimpleAllocator struct{}

func (s *SimpleAllocator) AllocateBuffer(size int) []byte {
	return make([]byte, size)
}

func (s *SimpleAllocator) DeallocateBuffer(buf []byte) {
	// Go GC가 자동으로 처리
}

// 슬랩 할당자 개념을 반영한 메모리 풀
type SlabPool struct {
	small  sync.Pool // 256바이트 미만 (SLOB 개념)
	medium sync.Pool // 1024바이트 미만
	large  sync.Pool // 페이지 크기 미만
}

func NewSlabPool() *SlabPool {
	return &SlabPool{
		small: sync.Pool{
			New: func() interface{} {
				return make([]byte, 256)
			},
		},
		medium: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
		large: sync.Pool{
			New: func() interface{} {
				return make([]byte, 4096) // 페이지 크기
			},
		},
	}
}

func (sp *SlabPool) AllocateBuffer(size int) []byte {
	switch {
	case size <= 256:
		buf := sp.small.Get().([]byte)
		return buf[:size]
	case size <= 1024:
		buf := sp.medium.Get().([]byte)
		return buf[:size]
	case size <= 4096:
		buf := sp.large.Get().([]byte)
		return buf[:size]
	default:
		return make([]byte, size) // 큰 할당은 직접 할당
	}
}

func (sp *SlabPool) DeallocateBuffer(buf []byte, originalSize int) {
	switch cap(buf) {
	case 256:
		sp.small.Put(buf[:256])
	case 1024:
		sp.medium.Put(buf[:1024])
	case 4096:
		sp.large.Put(buf[:4096])
	}
}

// 버디 시스템 개념을 반영한 2의 거듭제곱 할당자
type BuddyAllocator struct {
	pools map[int]*sync.Pool
	mutex sync.RWMutex
}

func NewBuddyAllocator() *BuddyAllocator {
	ba := &BuddyAllocator{
		pools: make(map[int]*sync.Pool),
	}
	
	// 2의 거듭제곱 크기별 풀 초기화
	for size := 256; size <= 32768; size *= 2 {
		currentSize := size
		ba.pools[size] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, currentSize)
			},
		}
	}
	return ba
}

func (ba *BuddyAllocator) nextPowerOf2(size int) int {
	power := 256
	for power < size {
		power *= 2
	}
	return power
}

func (ba *BuddyAllocator) AllocateBuffer(size int) []byte {
	allocSize := ba.nextPowerOf2(size)
	ba.mutex.RLock()
	pool, exists := ba.pools[allocSize]
	ba.mutex.RUnlock()
	
	if exists {
		buf := pool.Get().([]byte)
		return buf[:size]
	}
	return make([]byte, size)
}

func (ba *BuddyAllocator) DeallocateBuffer(buf []byte) {
	allocSize := cap(buf)
	ba.mutex.RLock()
	pool, exists := ba.pools[allocSize]
	ba.mutex.RUnlock()
	
	if exists {
		pool.Put(buf[:allocSize])
	}
}

// 성능 테스트
func benchmarkAllocator(name string, allocFunc func(int) []byte, deallocFunc func([]byte), iterations int) {
	start := time.Now()
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	
	for i := 0; i < iterations; i++ {
		size := 100 + (i % 500) // 다양한 크기
		buf := allocFunc(size)
		deallocFunc(buf)
	}
	
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	
	duration := time.Since(start)
	fmt.Printf("%s:\n", name)
	fmt.Printf("  시간: %v\n", duration)
	fmt.Printf("  할당된 메모리: %d KB\n", (memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)
	fmt.Printf("  GC 횟수: %d\n", memAfter.NumGC-memBefore.NumGC)
	fmt.Println()
}

func main() {
	iterations := 100000
	
	// 1. 일반적인 할당
	simple := &SimpleAllocator{}
	benchmarkAllocator("일반 할당 (커널 개념 미반영)", 
		simple.AllocateBuffer,
		func(buf []byte) { simple.DeallocateBuffer(buf) },
		iterations)
	
	// 2. 슬랩 풀 할당
	slabPool := NewSlabPool()
	benchmarkAllocator("슬랩 풀 할당 (SLAB/SLOB 개념 반영)",
		slabPool.AllocateBuffer,
		func(buf []byte) { slabPool.DeallocateBuffer(buf, len(buf)) },
		iterations)
	
	// 3. 버디 시스템 할당
	buddyAlloc := NewBuddyAllocator()
	benchmarkAllocator("버디 시스템 할당 (Buddy System 개념 반영)",
		buddyAlloc.AllocateBuffer,
		func(buf []byte) { buddyAlloc.DeallocateBuffer(buf) },
		iterations)
	
	// 내부 단편화 예시
	fmt.Println("=== 내부 단편화 비교 ===")
	requestSize := 600
	
	simpleBuf := simple.AllocateBuffer(requestSize)
	fmt.Printf("일반 할당 - 요청: %d, 할당: %d, 낭비: %d\n", 
		requestSize, cap(simpleBuf), cap(simpleBuf)-requestSize)
	
	buddyBuf := buddyAlloc.AllocateBuffer(requestSize)
	fmt.Printf("버디 시스템 - 요청: %d, 할당: %d, 낭비: %d\n", 
		requestSize, cap(buddyBuf), cap(buddyBuf)-requestSize)
}