// Package main demonstrates the valid-invalid bit memory management simulator
// 유효-무효 비트 메모리 관리 시뮬레이터 데모 프로그램
package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// main function demonstrates various memory management scenarios
// 다양한 메모리 관리 시나리오를 시연하는 메인 함수
func main() {
	rand.Seed(time.Now().UnixNano())
	
	fmt.Println("=== 유효-무효 비트 기반 메모리 관리 시뮬레이터 ===\n")
	
	// 시뮬레이션 파라미터 설정
	pageTableSize := 8  // 페이지 테이블 크기 (8개 페이지)
	processID := 1      // 프로세스 ID
	totalFrames := 4    // 물리 메모리 프레임 수 (메모리 부족 상황 시뮬레이션)
	frameSize := 1024   // 각 프레임 크기 (1KB)
	
	mm := NewMemoryManager(pageTableSize, processID, totalFrames, frameSize)
	
	fmt.Printf("시뮬레이터 초기화 완료\n")
	fmt.Printf("- 페이지 테이블 크기: %d 페이지\n", pageTableSize)
	fmt.Printf("- 물리 메모리: %d 프레임\n", totalFrames)
	fmt.Printf("- 프레임 크기: %d 바이트\n\n", frameSize)
	
	testScenario1(mm)
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	
	testScenario2(mm)
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	
	testScenario3(mm)
	
	mm.PrintStatistics()
}

// testScenario1 demonstrates basic memory access and page faults
// 기본적인 메모리 접근과 페이지 폴트를 시연합니다
func testScenario1(mm *MemoryManager) {
	fmt.Println("=== 시나리오 1: 기본 메모리 접근 및 페이지 폴트 ===")
	
	addresses := []int{0, 1024, 2048, 3072, 4096, 5120}
	
	for _, addr := range addresses {
		fmt.Printf("\n가상 주소 %d 접근 시도...\n", addr)
		_, err := mm.AccessMemory(addr)
		if err != nil {
			fmt.Printf("오류: %v\n", err)
		}
		
		time.Sleep(100 * time.Millisecond)
	}
	
	mm.PrintPageTable()
}

// testScenario2 demonstrates memory writes and dirty bit handling
// 메모리 쓰기와 더티 비트 처리를 시연합니다
func testScenario2(mm *MemoryManager) {
	fmt.Println("=== 시나리오 2: 메모리 쓰기 및 더티 비트 테스트 ===")
	
	writeAddresses := []int{512, 1536, 2560}
	
	for i, addr := range writeAddresses {
		fmt.Printf("\n가상 주소 %d에 데이터 쓰기 시도...\n", addr)
		err := mm.WriteMemory(addr, fmt.Sprintf("데이터_%d", i))
		if err != nil {
			fmt.Printf("오류: %v\n", err)
		}
		
		time.Sleep(100 * time.Millisecond)
	}
	
	mm.PrintPageTable()
}

// testScenario3 demonstrates memory shortage and page replacement
// 메모리 부족 상황과 페이지 교체를 시연합니다
func testScenario3(mm *MemoryManager) {
	fmt.Println("=== 시나리오 3: 메모리 부족 상황 및 페이지 교체 ===")
	
	moreAddresses := []int{6144, 7168, 8192}
	
	for _, addr := range moreAddresses {
		fmt.Printf("\n가상 주소 %d 접근 시도 (메모리 부족 상황)...\n", addr)
		_, err := mm.AccessMemory(addr)
		if err != nil {
			fmt.Printf("오류: %v\n", err)
		}
		
		time.Sleep(100 * time.Millisecond)
	}
	
	mm.PrintPageTable()
}