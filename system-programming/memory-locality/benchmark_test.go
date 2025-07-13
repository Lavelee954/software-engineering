package main

import (
	"testing"
)

// 행렬 곱셈 벤치마크 테스트
func BenchmarkNaiveMatrixMultiply(b *testing.B) {
	size := 256
	A := NewMatrix(size, size)
	B := NewMatrix(size, size)
	A.RandomFill()
	B.RandomFill()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NaiveMatrixMultiply(A, B)
	}
}

func BenchmarkImprovedMatrixMultiply(b *testing.B) {
	size := 256
	A := NewMatrix(size, size)
	B := NewMatrix(size, size)
	A.RandomFill()
	B.RandomFill()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ImprovedMatrixMultiply(A, B)
	}
}

func BenchmarkBlockedMatrixMultiply(b *testing.B) {
	size := 256
	blockSize := 64
	A := NewMatrix(size, size)
	B := NewMatrix(size, size)
	A.RandomFill()
	B.RandomFill()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BlockedMatrixMultiply(A, B, blockSize)
	}
}

// =====================================================
// 새로운 캐시 최적화 벤치마크 테스트
// =====================================================

// False Sharing 벤치마크 (패딩 없음 vs 패딩 있음)
func BenchmarkCacheConflicts(b *testing.B) {
	iterations := 10000

	b.Run("NoPadding", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = BenchmarkFalseSharing(iterations)
		}
	})

	b.Run("WithPadding", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = BenchmarkCacheOptimized(iterations)
		}
	})
}

// 스트라이드 접근 패턴 벤치마크
func BenchmarkAccessPatterns(b *testing.B) {
	size := 1024 * 1024
	arr := make([]int64, size)
	for i := range arr {
		arr[i] = int64(i)
	}

	b.Run("BadStride", func(b *testing.B) {
		stride := 1024
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			BadStrideAccess(arr, size, stride)
		}
	})

	b.Run("Sequential", func(b *testing.B) {
		stride := 1024
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			GoodSequentialAccess(arr, size/stride)
		}
	})
}

// 패딩된 매트릭스 곱셈 벤치마크
func BenchmarkPaddedMatrixMultiply(b *testing.B) {
	size := 256
	A := NewPaddedMatrix(size, size)
	B := NewPaddedMatrix(size, size)
	A.RandomFill()
	B.RandomFill()

	b.Run("NaivePadded", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NaivePaddedMatrixMultiply(A, B)
		}
	})

	b.Run("OptimizedPadded", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			OptimizedMatrixMultiply(A, B, OptimalBlockSize)
		}
	})
}

// 메모리 지역성 비교: 일반 vs 패딩된 구조체
func BenchmarkMemoryLocality(b *testing.B) {
	size := 1000000

	b.Run("NoLocalityAccess", func(b *testing.B) {
		arr := make([]int64, size)
		stride := 256 // 캐시 미스 유발
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sum := int64(0)
			for j := 0; j < size; j += stride {
				if j < len(arr) {
					sum += arr[j]
				}
			}
			_ = sum
		}
	})

	b.Run("GoodLocalityAccess", func(b *testing.B) {
		arr := make([]int64, size)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sum := int64(0)
			for j := 0; j < size/256; j++ { // 같은 수의 원소에 접근
				sum += arr[j]
			}
			_ = sum
		}
	})
}

// 캐시 라인 정렬 효과 테스트
func BenchmarkCacheLineAlignment(b *testing.B) {
	b.Run("UnalignedStruct", func(b *testing.B) {
		data := &NoPaddingStruct{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data.counter1++
			data.counter2++
			data.counter3++
			data.counter4++
		}
	})

	b.Run("AlignedStruct", func(b *testing.B) {
		data := &PaddedStruct{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data.counter1++
			data.counter2++
			data.counter3++
			data.counter4++
		}
	})
}

// 배열 순회 벤치마크 테스트
func BenchmarkRowMajorTraversal(b *testing.B) {
	size := 1024
	arr := NewArray2D(size, size)
	arr.RandomFillArray()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RowMajorTraversal(arr)
	}
}

func BenchmarkColumnMajorTraversal(b *testing.B) {
	size := 1024
	arr := NewArray2D(size, size)
	arr.RandomFillArray()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ColumnMajorTraversal(arr)
	}
}

// 다양한 크기별 성능 비교
func BenchmarkMatrixMultiply128(b *testing.B) {
	benchmarkMatrixMultiplySize(b, 128)
}

func BenchmarkMatrixMultiply256(b *testing.B) {
	benchmarkMatrixMultiplySize(b, 256)
}

func BenchmarkMatrixMultiply512(b *testing.B) {
	benchmarkMatrixMultiplySize(b, 512)
}

func benchmarkMatrixMultiplySize(b *testing.B, size int) {
	A := NewMatrix(size, size)
	B := NewMatrix(size, size)
	A.RandomFill()
	B.RandomFill()

	b.Run("Naive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NaiveMatrixMultiply(A, B)
		}
	})

	b.Run("Improved", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ImprovedMatrixMultiply(A, B)
		}
	})

	b.Run("Blocked", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BlockedMatrixMultiply(A, B, 64)
		}
	})
}

// 종합 성능 비교: 최적화 전 vs 후
func BenchmarkComprehensiveComparison(b *testing.B) {
	size := 256

	b.Run("UnoptimizedApproach", func(b *testing.B) {
		// 최적화되지 않은 접근법
		A := NewMatrix(size, size)
		B := NewMatrix(size, size)
		A.RandomFill()
		B.RandomFill()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			NaiveMatrixMultiply(A, B)
		}
	})

	b.Run("FullyOptimizedApproach", func(b *testing.B) {
		// 완전 최적화된 접근법 (블록킹 + 패딩)
		A := NewPaddedMatrix(size, size)
		B := NewPaddedMatrix(size, size)
		A.RandomFill()
		B.RandomFill()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			OptimizedMatrixMultiply(A, B, OptimalBlockSize)
		}
	})
}

// 캐시 미스 시뮬레이션을 위한 큰 배열 테스트
func BenchmarkLargeArrayRowMajor(b *testing.B) {
	size := 2048
	arr := NewArray2D(size, size)
	arr.RandomFillArray()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RowMajorTraversal(arr)
	}
}

func BenchmarkLargeArrayColumnMajor(b *testing.B) {
	size := 2048
	arr := NewArray2D(size, size)
	arr.RandomFillArray()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ColumnMajorTraversal(arr)
	}
}

// 메모리 할당 최적화 테스트
func BenchmarkMemoryAllocation(b *testing.B) {
	size := 512

	b.Run("NewMatrix", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := NewMatrix(size, size)
			_ = m
		}
	})

	b.Run("NewArray2D", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a := NewArray2D(size, size)
			_ = a
		}
	})

	b.Run("NewPaddedMatrix", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := NewPaddedMatrix(size, size)
			_ = m
		}
	})
}
