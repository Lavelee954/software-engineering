package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	// 캐시 라인 크기 (대부분의 현대 CPU에서 64바이트)
	CacheLineSize = 64
	// L1 캐시 크기 고려한 블록 크기
	OptimalBlockSize = 64
)

// =====================================================
// 1. 블록킹/타일링을 위한 구조체와 함수들
// =====================================================

// PaddedMatrix - 캐시 라인 정렬을 고려한 매트릭스
type PaddedMatrix struct {
	data [][]int64 // int64 사용으로 더 명확한 성능 차이
	rows int
	cols int
	// 패딩을 통한 false sharing 방지
	_ [CacheLineSize - unsafe.Sizeof([][]int64{}) - 2*unsafe.Sizeof(int(0))]byte
}

// NewPaddedMatrix 캐시 라인 정렬된 매트릭스 생성
func NewPaddedMatrix(rows, cols int) *PaddedMatrix {
	// 각 행을 캐시 라인 경계에 정렬
	paddedCols := ((cols*8 + CacheLineSize - 1) / CacheLineSize) * CacheLineSize / 8

	data := make([][]int64, rows)
	for i := range data {
		data[i] = make([]int64, paddedCols)
	}

	return &PaddedMatrix{
		data: data,
		rows: rows,
		cols: cols,
	}
}

// RandomFill 랜덤 값으로 채우기
func (m *PaddedMatrix) RandomFill() {
	for i := 0; i < m.rows; i++ {
		for j := 0; j < m.cols; j++ {
			m.data[i][j] = int64(rand.Intn(100))
		}
	}
}

// Get 값 조회
func (m *PaddedMatrix) Get(i, j int) int64 {
	return m.data[i][j]
}

// Set 값 설정
func (m *PaddedMatrix) Set(i, j int, value int64) {
	m.data[i][j] = value
}

// =====================================================
// 2. 캐시 충돌 회피를 위한 데이터 패딩 예제
// =====================================================

// NoPaddingStruct - 패딩 없는 구조체 (캐시 충돌 발생 가능)
type NoPaddingStruct struct {
	counter1 int64
	counter2 int64
	counter3 int64
	counter4 int64
}

// PaddedStruct - 패딩 있는 구조체 (캐시 충돌 방지)
type PaddedStruct struct {
	counter1 int64
	_        [CacheLineSize - 8]byte // 패딩으로 캐시 라인 분리
	counter2 int64
	_        [CacheLineSize - 8]byte
	counter3 int64
	_        [CacheLineSize - 8]byte
	counter4 int64
	_        [CacheLineSize - 8]byte
}

// FalseSharing 테스트 - 패딩 없는 경우
func BenchmarkFalseSharing(iterations int) time.Duration {
	data := &NoPaddingStruct{}
	numGoroutines := runtime.NumCPU()

	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 각 고루틴이 다른 카운터를 수정하지만 같은 캐시 라인에 있음
				switch id % 4 {
				case 0:
					data.counter1++
				case 1:
					data.counter2++
				case 2:
					data.counter3++
				case 3:
					data.counter4++
				}
			}
		}(i)
	}

	wg.Wait()
	return time.Since(start)
}

// 캐시 충돌 회피 테스트 - 패딩 있는 경우
func BenchmarkCacheOptimized(iterations int) time.Duration {
	data := &PaddedStruct{}
	numGoroutines := runtime.NumCPU()

	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 각 카운터가 별도 캐시 라인에 있어 false sharing 방지
				switch id % 4 {
				case 0:
					data.counter1++
				case 1:
					data.counter2++
				case 2:
					data.counter3++
				case 3:
					data.counter4++
				}
			}
		}(i)
	}

	wg.Wait()
	return time.Since(start)
}

// =====================================================
// 3. 스트라이드 접근 패턴과 캐시 충돌
// =====================================================

// BadStrideAccess - 큰 스트라이드로 인한 캐시 충돌
func BadStrideAccess(arr []int64, size, stride int) int64 {
	sum := int64(0)
	for i := 0; i < size; i += stride {
		if i < len(arr) {
			sum += arr[i] // 큰 스트라이드로 캐시 미스 빈발
		}
	}
	return sum
}

// GoodSequentialAccess - 순차 접근으로 캐시 효율성 극대화
func GoodSequentialAccess(arr []int64, size int) int64 {
	sum := int64(0)
	for i := 0; i < size && i < len(arr); i++ {
		sum += arr[i] // 순차 접근으로 캐시 적중률 높음
	}
	return sum
}

// =====================================================
// 4. 블록킹과 패딩을 모두 적용한 매트릭스 곱셈
// =====================================================

// 패딩 없는 일반 매트릭스 곱셈
func NaivePaddedMatrixMultiply(A, B *PaddedMatrix) *PaddedMatrix {
	C := NewPaddedMatrix(A.rows, B.cols)

	for i := 0; i < A.rows; i++ {
		for j := 0; j < B.cols; j++ {
			sum := int64(0)
			for k := 0; k < A.cols; k++ {
				sum += A.Get(i, k) * B.Get(k, j) // 캐시 미스 빈발
			}
			C.Set(i, j, sum)
		}
	}

	return C
}

// 블록킹과 패딩을 모두 적용한 최적화된 매트릭스 곱셈
func OptimizedMatrixMultiply(A, B *PaddedMatrix, blockSize int) *PaddedMatrix {
	C := NewPaddedMatrix(A.rows, B.cols)

	// 블록킹으로 캐시 지역성 극대화
	for ii := 0; ii < A.rows; ii += blockSize {
		for jj := 0; jj < B.cols; jj += blockSize {
			for kk := 0; kk < A.cols; kk += blockSize {

				iEnd := ii + blockSize
				if iEnd > A.rows {
					iEnd = A.rows
				}
				jEnd := jj + blockSize
				if jEnd > B.cols {
					jEnd = B.cols
				}
				kEnd := kk + blockSize
				if kEnd > A.cols {
					kEnd = A.cols
				}

				// 블록 내에서 최적화된 접근 패턴
				for i := ii; i < iEnd; i++ {
					for k := kk; k < kEnd; k++ {
						aik := A.Get(i, k) // 시간적 지역성
						for j := jj; j < jEnd; j++ {
							// 공간적 지역성 + 패딩으로 캐시 충돌 방지
							C.Set(i, j, C.Get(i, j)+aik*B.Get(k, j))
						}
					}
				}
			}
		}
	}

	return C
}

// =====================================================
// 5. 종합 벤치마크 함수들
// =====================================================

// 캐시 충돌 회피 테스트
func BenchmarkCacheConflictAvoidance() {
	fmt.Println("\n=== Cache Conflict Avoidance Test ===")
	fmt.Printf("CPU Count: %d, Cache Line Size: %d bytes\n", runtime.NumCPU(), CacheLineSize)

	iterations := 1000000

	// 1. False Sharing 테스트 (패딩 없음)
	fmt.Println("\n1. False Sharing (No Padding):")
	fmt.Println("   - Multiple goroutines modify different variables")
	fmt.Println("   - Variables share same cache line")
	fmt.Println("   - High cache coherency overhead")
	falseTime := BenchmarkFalseSharing(iterations)
	fmt.Printf("   - Time: %v\n", falseTime)

	// 2. 캐시 최적화 테스트 (패딩 있음)
	fmt.Println("\n2. Cache Optimized (With Padding):")
	fmt.Println("   - Each variable in separate cache line")
	fmt.Println("   - No false sharing")
	fmt.Println("   - Minimal cache coherency overhead")
	optimizedTime := BenchmarkCacheOptimized(iterations)
	fmt.Printf("   - Time: %v\n", optimizedTime)

	speedup := float64(falseTime) / float64(optimizedTime)
	fmt.Printf("   - Speedup: %.2fx\n", speedup)
}

// 스트라이드 접근 패턴 테스트
func BenchmarkStrideAccess() {
	fmt.Println("\n=== Stride Access Pattern Test ===")

	size := 1024 * 1024 // 1M elements
	arr := make([]int64, size)

	// 배열 초기화
	for i := range arr {
		arr[i] = int64(i)
	}

	// 1. 나쁜 스트라이드 접근 (캐시 충돌)
	fmt.Println("\n1. Bad Stride Access (Cache Conflicts):")
	stride := 1024 // 큰 스트라이드로 캐시 세트 충돌 유발
	fmt.Printf("   - Stride: %d (likely cache conflicts)\n", stride)
	fmt.Println("   - Access pattern: arr[0], arr[1024], arr[2048], ...")
	start := time.Now()
	sum1 := BadStrideAccess(arr, size, stride)
	strideTime := time.Since(start)
	fmt.Printf("   - Time: %v, Sum: %d\n", strideTime, sum1)

	// 2. 좋은 순차 접근
	fmt.Println("\n2. Good Sequential Access:")
	fmt.Println("   - Access pattern: arr[0], arr[1], arr[2], ...")
	fmt.Println("   - Optimal cache line utilization")
	start = time.Now()
	sum2 := GoodSequentialAccess(arr, size/stride)
	seqTime := time.Since(start)
	fmt.Printf("   - Time: %v, Sum: %d\n", seqTime, sum2)

	speedup := float64(strideTime) / float64(seqTime)
	fmt.Printf("   - Sequential vs Stride speedup: %.2fx\n", speedup)
}

// 매트릭스 곱셈 종합 테스트
func BenchmarkComprehensiveMatrixMultiply(size int) {
	fmt.Printf("\n=== Comprehensive Matrix Multiplication (%dx%d) ===\n", size, size)

	// 테스트용 매트릭스 생성
	A := NewPaddedMatrix(size, size)
	B := NewPaddedMatrix(size, size)
	A.RandomFill()
	B.RandomFill()

	// 1. 순진한 방법 (최적화 없음)
	fmt.Println("\n1. Naive Method (No Optimization):")
	fmt.Println("   - No blocking, no padding consideration")
	fmt.Println("   - High cache misses")
	start := time.Now()
	result1 := NaivePaddedMatrixMultiply(A, B)
	naiveTime := time.Since(start)
	fmt.Printf("   - Time: %v\n", naiveTime)

	// 2. 완전 최적화 방법 (블록킹 + 패딩)
	fmt.Printf("\n2. Fully Optimized (Blocking + Padding):")
	fmt.Printf("   - Block size: %d (fits in L1 cache)\n", OptimalBlockSize)
	fmt.Println("   - Cache-aligned data structures")
	fmt.Println("   - Optimal memory access patterns")
	start = time.Now()
	result2 := OptimizedMatrixMultiply(A, B, OptimalBlockSize)
	optimizedTime := time.Since(start)
	fmt.Printf("   - Time: %v\n", optimizedTime)

	speedup := float64(naiveTime) / float64(optimizedTime)
	fmt.Printf("   - Total speedup: %.2fx\n", speedup)

	// 결과 검증
	fmt.Println("\n3. Result Verification:")
	equal := true
	maxCheck := size
	if maxCheck > 100 {
		maxCheck = 100
	}
	for i := 0; i < maxCheck; i++ {
		for j := 0; j < maxCheck; j++ {
			if result1.Get(i, j) != result2.Get(i, j) {
				equal = false
				break
			}
		}
		if !equal {
			break
		}
	}
	fmt.Printf("   - Results match: %t\n", equal)
}

// 메인 실행 함수
func RunCacheOptimizationDemo() {
	fmt.Println("Advanced Cache Optimization Techniques Demo")
	fmt.Println("===========================================")

	// 1. 캐시 충돌 회피 테스트
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PART 1: CACHE CONFLICT AVOIDANCE (DATA PADDING)")
	fmt.Println(strings.Repeat("=", 60))
	BenchmarkCacheConflictAvoidance()

	// 2. 스트라이드 접근 패턴 테스트
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PART 2: STRIDE ACCESS PATTERNS")
	fmt.Println(strings.Repeat("=", 60))
	BenchmarkStrideAccess()

	// 3. 종합 매트릭스 곱셈 테스트
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PART 3: COMPREHENSIVE MATRIX MULTIPLICATION")
	fmt.Println(strings.Repeat("=", 60))

	sizes := []int{256, 512}
	for _, size := range sizes {
		BenchmarkComprehensiveMatrixMultiply(size)
		fmt.Println("\n" + strings.Repeat("-", 40))
	}

	// 4. 요약 및 결론
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("SUMMARY: ADVANCED CACHE OPTIMIZATION TECHNIQUES")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n=== 1. 블록킹/타일링 (Blocking/Tiling) ===")
	fmt.Println("목적: 캐시에 올린 데이터의 최대한 재사용")
	fmt.Println("원리:")
	fmt.Println("  - 큰 데이터를 캐시 크기에 맞는 작은 블록으로 분할")
	fmt.Println("  - 각 블록이 캐시에서 완전히 처리된 후 다음 블록으로 이동")
	fmt.Println("  - 시간적 지역성: 블록 내 데이터 반복 사용")
	fmt.Println("  - 공간적 지역성: 연속된 메모리 접근")
	fmt.Println("효과:")
	fmt.Println("  - 캐시 미스 대폭 감소")
	fmt.Println("  - 2-10배 성능 향상 가능")

	fmt.Println("\n=== 2. 데이터 패딩 (Data Padding) ===")
	fmt.Println("목적: 캐시 충돌 회피")
	fmt.Println("원리:")
	fmt.Println("  - False Sharing 방지: 서로 다른 변수를 다른 캐시 라인에 배치")
	fmt.Println("  - Cache Line Alignment: 데이터를 캐시 라인 경계에 정렬")
	fmt.Println("  - Stride Conflicts 회피: 큰 스트라이드 접근 패턴 최적화")
	fmt.Println("효과:")
	fmt.Println("  - 멀티쓰레드 환경에서 특히 효과적")
	fmt.Println("  - 캐시 코히어런시 오버헤드 감소")
	fmt.Println("  - 2-5배 성능 향상 가능")

	fmt.Println("\n=== 3. 결합 효과 ===")
	fmt.Println("두 기법을 함께 사용할 때:")
	fmt.Println("  - 시너지 효과로 더 큰 성능 향상")
	fmt.Println("  - 다양한 캐시 계층에서 최적 성능")
	fmt.Println("  - 현대 멀티코어 시스템에서 필수적")

	fmt.Println("\n=== 4. 실무 적용 가이드라인 ===")
	fmt.Println("언제 사용할까:")
	fmt.Println("  - 대용량 데이터 처리")
	fmt.Println("  - 반복적인 메모리 접근 패턴")
	fmt.Println("  - 멀티쓰레드 고성능 어플리케이션")
	fmt.Println("  - 실시간 시스템")
	fmt.Println("주의사항:")
	fmt.Println("  - 메모리 사용량 증가 (패딩으로 인한)")
	fmt.Println("  - 코드 복잡성 증가")
	fmt.Println("  - 하드웨어별 최적화 필요")
}
