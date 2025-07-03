package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// 성능 테스트 결과 구조체
type BenchmarkResult struct {
	Name            string
	Duration        time.Duration
	MemoryUsed      uint64
	GCCount         uint32
	AllocationsPerSec float64
}

// 일반 할당자 (커널 개념 미적용)
type StandardAllocator struct {
	totalAllocations int64
	mutex           sync.Mutex
}

func (sa *StandardAllocator) Allocate(size int) []byte {
	sa.mutex.Lock()
	sa.totalAllocations++
	sa.mutex.Unlock()
	return make([]byte, size)
}

func (sa *StandardAllocator) Free(buf []byte) {
	// Go GC가 자동 처리
}

func (sa *StandardAllocator) GetStats() int64 {
	sa.mutex.Lock()
	defer sa.mutex.Unlock()
	return sa.totalAllocations
}

// 슬랩 할당자 (커널 개념 적용)
type OptimizedSlabAllocator struct {
	pools map[int]*sync.Pool
	totalAllocations int64
	mutex           sync.RWMutex
}

func NewOptimizedSlabAllocator() *OptimizedSlabAllocator {
	osa := &OptimizedSlabAllocator{
		pools: make(map[int]*sync.Pool),
	}
	
	// 다양한 크기의 풀 생성
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096, 8192}
	for _, size := range sizes {
		currentSize := size
		osa.pools[size] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, currentSize)
			},
		}
	}
	
	return osa
}

func (osa *OptimizedSlabAllocator) getOptimalSize(size int) int {
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096, 8192}
	for _, poolSize := range sizes {
		if size <= poolSize {
			return poolSize
		}
	}
	return size // 너무 큰 경우 직접 할당
}

func (osa *OptimizedSlabAllocator) Allocate(size int) []byte {
	osa.mutex.Lock()
	osa.totalAllocations++
	osa.mutex.Unlock()
	
	optimalSize := osa.getOptimalSize(size)
	
	osa.mutex.RLock()
	pool, exists := osa.pools[optimalSize]
	osa.mutex.RUnlock()
	
	if exists {
		buf := pool.Get().([]byte)
		return buf[:size]
	}
	
	return make([]byte, size)
}

func (osa *OptimizedSlabAllocator) Free(buf []byte) {
	if buf == nil {
		return
	}
	
	poolSize := cap(buf)
	
	osa.mutex.RLock()
	pool, exists := osa.pools[poolSize]
	osa.mutex.RUnlock()
	
	if exists {
		pool.Put(buf[:poolSize])
	}
}

func (osa *OptimizedSlabAllocator) GetStats() int64 {
	osa.mutex.RLock()
	defer osa.mutex.RUnlock()
	return osa.totalAllocations
}

// 버디 시스템 할당자
type BuddySystemAllocator struct {
	pools map[int]*sync.Pool
	totalAllocations int64
	mutex           sync.RWMutex
}

func NewBuddySystemAllocator() *BuddySystemAllocator {
	bsa := &BuddySystemAllocator{
		pools: make(map[int]*sync.Pool),
	}
	
	// 2의 거듭제곱 크기 풀 생성
	for size := 64; size <= 8192; size *= 2 {
		currentSize := size
		bsa.pools[size] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, currentSize)
			},
		}
	}
	
	return bsa
}

func (bsa *BuddySystemAllocator) nextPowerOf2(size int) int {
	power := 64
	for power < size {
		power *= 2
	}
	if power > 8192 {
		return size // 너무 큰 경우 직접 할당
	}
	return power
}

func (bsa *BuddySystemAllocator) Allocate(size int) []byte {
	bsa.mutex.Lock()
	bsa.totalAllocations++
	bsa.mutex.Unlock()
	
	allocSize := bsa.nextPowerOf2(size)
	
	bsa.mutex.RLock()
	pool, exists := bsa.pools[allocSize]
	bsa.mutex.RUnlock()
	
	if exists {
		buf := pool.Get().([]byte)
		return buf[:size]
	}
	
	return make([]byte, size)
}

func (bsa *BuddySystemAllocator) Free(buf []byte) {
	if buf == nil {
		return
	}
	
	allocSize := cap(buf)
	
	bsa.mutex.RLock()
	pool, exists := bsa.pools[allocSize]
	bsa.mutex.RUnlock()
	
	if exists {
		pool.Put(buf[:allocSize])
	}
}

func (bsa *BuddySystemAllocator) GetStats() int64 {
	bsa.mutex.RLock()
	defer bsa.mutex.RUnlock()
	return bsa.totalAllocations
}

// 할당자 인터페이스
type Allocator interface {
	Allocate(size int) []byte
	Free(buf []byte)
	GetStats() int64
}

// 성능 테스트 함수
func benchmarkAllocator(name string, allocator Allocator, iterations int, concurrency int) BenchmarkResult {
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	
	start := time.Now()
	
	var wg sync.WaitGroup
	requestsPerGoroutine := iterations / concurrency
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := 0; j < requestsPerGoroutine; j++ {
				// 다양한 크기의 할당 요청 시뮬레이션
				sizes := []int{64, 128, 256, 512, 1024, 2048}
				size := sizes[j%len(sizes)]
				
				buf := allocator.Allocate(size)
				
				// 메모리 사용 시뮬레이션
				if len(buf) > 0 {
					buf[0] = byte(j % 256)
				}
				
				allocator.Free(buf)
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	
	totalAllocations := allocator.GetStats()
	allocationsPerSec := float64(totalAllocations) / duration.Seconds()
	
	return BenchmarkResult{
		Name:              name,
		Duration:          duration,
		MemoryUsed:        memAfter.TotalAlloc - memBefore.TotalAlloc,
		GCCount:           memAfter.NumGC - memBefore.NumGC,
		AllocationsPerSec: allocationsPerSec,
	}
}

// 메모리 단편화 테스트
func testFragmentation() {
	fmt.Println("\n=== 메모리 단편화 분석 ===")
	
	testSizes := []int{100, 300, 700, 1500, 3000}
	
	fmt.Printf("%-15s %-10s %-15s %-15s %-10s\n", 
		"할당자", "요청크기", "실제할당", "내부단편화", "효율성")
	fmt.Println(strings.Repeat("-", 70))
	
	for _, size := range testSizes {
		// 표준 할당자
		standardBuf := make([]byte, size)
		standardWaste := cap(standardBuf) - size
		standardEfficiency := float64(size) / float64(cap(standardBuf)) * 100
		
		fmt.Printf("%-15s %-10d %-15d %-15d %.2f%%\n", 
			"표준", size, cap(standardBuf), standardWaste, standardEfficiency)
		
		// 버디 시스템
		buddy := NewBuddySystemAllocator()
		buddyBuf := buddy.Allocate(size)
		buddyWaste := cap(buddyBuf) - size
		buddyEfficiency := float64(size) / float64(cap(buddyBuf)) * 100
		
		fmt.Printf("%-15s %-10d %-15d %-15d %.2f%%\n", 
			"버디시스템", size, cap(buddyBuf), buddyWaste, buddyEfficiency)
		
		// 슬랩 할당자
		slab := NewOptimizedSlabAllocator()
		slabBuf := slab.Allocate(size)
		slabWaste := cap(slabBuf) - size
		slabEfficiency := float64(size) / float64(cap(slabBuf)) * 100
		
		fmt.Printf("%-15s %-10d %-15d %-15d %.2f%%\n", 
			"슬랩", size, cap(slabBuf), slabWaste, slabEfficiency)
		
		fmt.Println()
	}
}

// 실무 시나리오 시뮬레이션
func simulateRealWorldScenario(allocator Allocator, name string) {
	fmt.Printf("\n=== %s 실무 시나리오 ===\n", name)
	
	scenarios := []struct {
		name        string
		iterations  int
		concurrency int
		description string
	}{
		{"웹서버 요청", 50000, 100, "HTTP 요청/응답 버퍼"},
		{"데이터베이스 쿼리", 10000, 50, "쿼리 결과 버퍼"},
		{"로그 처리", 100000, 200, "로그 메시지 버퍼"},
		{"파일 I/O", 5000, 20, "파일 읽기/쓰기 버퍼"},
	}
	
	for _, scenario := range scenarios {
		result := benchmarkAllocator(
			fmt.Sprintf("%s-%s", name, scenario.name),
			allocator,
			scenario.iterations,
			scenario.concurrency,
		)
		
		fmt.Printf("%-20s: %8.2f ns/op, %8.2f MB, %d GC, %.0f alloc/sec\n",
			scenario.name,
			float64(result.Duration.Nanoseconds())/float64(scenario.iterations),
			float64(result.MemoryUsed)/1024/1024,
			result.GCCount,
			result.AllocationsPerSec,
		)
	}
}

func main() {
	fmt.Println("커널 메모리 관리 개념 기반 Go 할당자 성능 비교")
	fmt.Println(strings.Repeat("=", 60))
	
	// 테스트 설정
	iterations := 100000
	concurrency := 50
	
	// 할당자들 초기화
	standard := &StandardAllocator{}
	slab := NewOptimizedSlabAllocator()
	buddy := NewBuddySystemAllocator()
	
	// 성능 테스트
	results := []BenchmarkResult{
		benchmarkAllocator("표준 할당자", standard, iterations, concurrency),
		benchmarkAllocator("슬랩 할당자 (SLAB)", slab, iterations, concurrency),
		benchmarkAllocator("버디 시스템 (Buddy)", buddy, iterations, concurrency),
	}
	
	// 결과 출력
	fmt.Printf("\n%-20s %10s %12s %8s %15s\n", 
		"할당자", "시간", "메모리(MB)", "GC횟수", "할당/초")
	fmt.Println(strings.Repeat("-", 70))
	
	baseTime := results[0].Duration
	baseMem := results[0].MemoryUsed
	
	for _, result := range results {
		speedup := float64(baseTime) / float64(result.Duration)
		memReduction := float64(baseMem) / float64(result.MemoryUsed)
		
		fmt.Printf("%-20s %10v %12.2f %8d %15.0f (%.2fx faster, %.2fx less mem)\n",
			result.Name,
			result.Duration,
			float64(result.MemoryUsed)/1024/1024,
			result.GCCount,
			result.AllocationsPerSec,
			speedup,
			memReduction,
		)
	}
	
	// 메모리 단편화 분석
	testFragmentation()
	
	// 실무 시나리오 테스트
	simulateRealWorldScenario(standard, "표준할당자")
	simulateRealWorldScenario(slab, "슬랩할당자")
	simulateRealWorldScenario(buddy, "버디시스템")
	
	fmt.Println("\n=== 결론 ===")
	fmt.Println("• 슬랩 할당자: 특정 크기 객체에 최적화, 단편화 최소")
	fmt.Println("• 버디 시스템: 2의 거듭제곱 할당, 병합 효율적")
	fmt.Println("• 표준 할당자: 단순하지만 GC 압박 높음")
	fmt.Println("• 실무 선택: 사용 패턴에 따라 적절한 할당자 선택 필요")
}