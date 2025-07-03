// Package main implements a valid-invalid-bit management simulator demonstrating valid-invalid bit concepts
// 유효-무효 비트 개념을 활용한 메모리 관리 시뮬레이터 구현
package main

import (
	"fmt"
	"math/rand"
	"time"
)

// PageTableEntry represents a single entry in the page table
// 페이지 테이블의 각 항목을 나타내는 구조체
type PageTableEntry struct {
	Valid       bool // 유효-무효 비트: 페이지가 물리 메모리에 있는지 여부
	FrameNumber int  // 프레임 번호: 물리 메모리에서의 위치
	Dirty       bool // 더티 비트: 페이지가 수정되었는지 여부
	Referenced  bool // 참조 비트: 페이지가 최근에 접근되었는지 여부
}

// PageTable represents the page table for a process
// 프로세스의 페이지 테이블을 나타내는 구조체
type PageTable struct {
	entries   []PageTableEntry // 페이지 테이블 항목들
	size      int              // 페이지 테이블 크기
	processID int              // 프로세스 ID
}

// PhysicalMemory represents the physical valid-invalid-bit with frames
// 프레임으로 구성된 물리 메모리를 나타내는 구조체
type PhysicalMemory struct {
	frames      []bool // 프레임 사용 여부 (true: 사용중, false: 사용가능)
	frameSize   int    // 각 프레임의 크기 (바이트)
	totalFrames int    // 전체 프레임 수
}

// MemoryManager manages the entire valid-invalid-bit system
// 전체 메모리 시스템을 관리하는 구조체
type MemoryManager struct {
	pageTable    *PageTable      // 페이지 테이블
	physicalMem  *PhysicalMemory // 물리 메모리
	swapSpace    map[int]bool    // 스왑 공간 (페이지 번호 -> 존재 여부)
	pageFaults   int             // 페이지 폴트 발생 횟수
	memoryAccess int             // 메모리 접근 횟수
}

// NewPageTable creates a new page table for a process
// 프로세스를 위한 새로운 페이지 테이블을 생성합니다
func NewPageTable(size, processID int) *PageTable {
	return &PageTable{
		entries:   make([]PageTableEntry, size),
		size:      size,
		processID: processID,
	}
}

// NewPhysicalMemory creates a new physical valid-invalid-bit structure
// 새로운 물리 메모리 구조를 생성합니다
func NewPhysicalMemory(totalFrames, frameSize int) *PhysicalMemory {
	return &PhysicalMemory{
		frames:      make([]bool, totalFrames),
		frameSize:   frameSize,
		totalFrames: totalFrames,
	}
}

// NewMemoryManager creates a new valid-invalid-bit manager with specified parameters
// 지정된 매개변수로 새로운 메모리 관리자를 생성합니다
func NewMemoryManager(pageTableSize, processID, totalFrames, frameSize int) *MemoryManager {
	return &MemoryManager{
		pageTable:    NewPageTable(pageTableSize, processID),
		physicalMem:  NewPhysicalMemory(totalFrames, frameSize),
		swapSpace:    make(map[int]bool),
		pageFaults:   0,
		memoryAccess: 0,
	}
}

// AllocateFrame finds and allocates a free frame in physical valid-invalid-bit
// 물리 메모리에서 사용 가능한 프레임을 찾아 할당합니다
// Returns: frame number if successful, -1 if no free frames available
func (mm *MemoryManager) AllocateFrame() int {
	for i := 0; i < mm.physicalMem.totalFrames; i++ {
		if !mm.physicalMem.frames[i] {
			mm.physicalMem.frames[i] = true
			return i
		}
	}
	return -1
}

func (mm *MemoryManager) FreeFrame(frameNumber int) {
	if frameNumber >= 0 && frameNumber < mm.physicalMem.totalFrames {
		mm.physicalMem.frames[frameNumber] = false
	}
}

// AccessMemory handles valid-invalid-bit access with virtual address translation
// 가상 주소를 물리 주소로 변환하여 메모리 접근을 처리합니다
// This simulates the hardware MMU (Memory Management Unit) behavior
func (mm *MemoryManager) AccessMemory(virtualAddress int) (int, error) {
	mm.memoryAccess++

	pageNumber := virtualAddress / mm.physicalMem.frameSize
	offset := virtualAddress % mm.physicalMem.frameSize

	if pageNumber >= mm.pageTable.size {
		return -1, fmt.Errorf("세그멘테이션 오류: 페이지 번호 %d는 유효하지 않습니다 (최대: %d)",
			pageNumber, mm.pageTable.size-1)
	}

	entry := &mm.pageTable.entries[pageNumber]

	if !entry.Valid {
		return mm.handlePageFault(pageNumber, offset)
	}

	entry.Referenced = true
	physicalAddress := entry.FrameNumber*mm.physicalMem.frameSize + offset

	fmt.Printf("메모리 접근 성공: 가상주소 %d -> 물리주소 %d (페이지 %d, 프레임 %d)\n",
		virtualAddress, physicalAddress, pageNumber, entry.FrameNumber)

	return physicalAddress, nil
}

// handlePageFault processes page faults by loading pages from swap space
// 페이지 폴트를 처리하여 스왑 공간에서 페이지를 로드합니다
// This simulates OS page fault handler behavior
func (mm *MemoryManager) handlePageFault(pageNumber, offset int) (int, error) {
	mm.pageFaults++
	fmt.Printf("페이지 폴트 발생: 페이지 %d\n", pageNumber)

	frameNumber := mm.AllocateFrame()
	if frameNumber == -1 {
		frameNumber = mm.evictPage()
	}

	fmt.Printf("페이지 %d를 프레임 %d에 로드 중...\n", pageNumber, frameNumber)
	time.Sleep(10 * time.Millisecond)

	entry := &mm.pageTable.entries[pageNumber]
	entry.Valid = true
	entry.FrameNumber = frameNumber
	entry.Referenced = true
	entry.Dirty = false

	if mm.swapSpace[pageNumber] {
		fmt.Printf("스왑 공간에서 페이지 %d를 복원했습니다\n", pageNumber)
		delete(mm.swapSpace, pageNumber)
	}

	physicalAddress := frameNumber*mm.physicalMem.frameSize + offset
	fmt.Printf("페이지 폴트 처리 완료: 가상주소 %d -> 물리주소 %d\n",
		pageNumber*mm.physicalMem.frameSize+offset, physicalAddress)

	return physicalAddress, nil
}

func (mm *MemoryManager) evictPage() int {
	for i := 0; i < mm.pageTable.size; i++ {
		entry := &mm.pageTable.entries[i]
		if entry.Valid && !entry.Referenced {
			frameNumber := entry.FrameNumber

			if entry.Dirty {
				fmt.Printf("더티 페이지 %d를 스왑 공간에 저장 중...\n", i)
				mm.swapSpace[i] = true
			}

			entry.Valid = false
			entry.FrameNumber = -1
			entry.Dirty = false
			entry.Referenced = false

			fmt.Printf("페이지 %d를 축출하여 프레임 %d를 확보했습니다\n", i, frameNumber)
			return frameNumber
		}
	}

	victimPage := rand.Intn(mm.pageTable.size)
	entry := &mm.pageTable.entries[victimPage]
	if entry.Valid {
		frameNumber := entry.FrameNumber

		if entry.Dirty {
			fmt.Printf("더티 페이지 %d를 스왑 공간에 저장 중...\n", victimPage)
			mm.swapSpace[victimPage] = true
		}

		entry.Valid = false
		entry.FrameNumber = -1
		entry.Dirty = false
		entry.Referenced = false

		fmt.Printf("페이지 %d를 축출하여 프레임 %d를 확보했습니다\n", victimPage, frameNumber)
		return frameNumber
	}

	return mm.AllocateFrame()
}

func (mm *MemoryManager) WriteMemory(virtualAddress int, data string) error {
	mm.memoryAccess++

	pageNumber := virtualAddress / mm.physicalMem.frameSize
	offset := virtualAddress % mm.physicalMem.frameSize

	if pageNumber >= mm.pageTable.size {
		return fmt.Errorf("세그멘테이션 오류: 페이지 번호 %d는 유효하지 않습니다", pageNumber)
	}

	entry := &mm.pageTable.entries[pageNumber]

	if !entry.Valid {
		_, err := mm.handlePageFault(pageNumber, offset)
		if err != nil {
			return err
		}
		entry = &mm.pageTable.entries[pageNumber]
	}

	entry.Referenced = true
	entry.Dirty = true

	physicalAddress := entry.FrameNumber*mm.physicalMem.frameSize + offset
	fmt.Printf("메모리 쓰기 성공: 가상주소 %d -> 물리주소 %d (데이터: %s)\n",
		virtualAddress, physicalAddress, data)

	return nil
}

func (mm *MemoryManager) PrintStatistics() {
	fmt.Printf("\n=== 메모리 관리 통계 ===\n")
	fmt.Printf("총 메모리 접근 횟수: %d\n", mm.memoryAccess)
	fmt.Printf("페이지 폴트 횟수: %d\n", mm.pageFaults)
	fmt.Printf("페이지 폴트 비율: %.2f%%\n", float64(mm.pageFaults)/float64(mm.memoryAccess)*100)

	validPages := 0
	for i := 0; i < mm.pageTable.size; i++ {
		if mm.pageTable.entries[i].Valid {
			validPages++
		}
	}
	fmt.Printf("현재 메모리에 있는 페이지 수: %d/%d\n", validPages, mm.pageTable.size)
	fmt.Printf("스왑 공간의 페이지 수: %d\n", len(mm.swapSpace))
}

func (mm *MemoryManager) PrintPageTable() {
	fmt.Printf("\n=== 페이지 테이블 상태 ===\n")
	fmt.Printf("페이지\t유효비트\t프레임\t더티\t참조\n")
	for i := 0; i < mm.pageTable.size; i++ {
		entry := mm.pageTable.entries[i]
		validStr := "무효"
		if entry.Valid {
			validStr = "유효"
		}

		frameStr := "-"
		if entry.Valid {
			frameStr = fmt.Sprintf("%d", entry.FrameNumber)
		}

		dirtyStr := "-"
		if entry.Dirty {
			dirtyStr = "Y"
		}

		refStr := "-"
		if entry.Referenced {
			refStr = "Y"
		}

		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", i, validStr, frameStr, dirtyStr, refStr)
	}
}
