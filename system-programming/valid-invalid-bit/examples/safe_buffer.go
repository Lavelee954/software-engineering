package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// SafeBuffer 안전한 버퍼 구조체 (유효-무효 비트 개념 적용)
type SafeBuffer struct {
	data  []byte
	size  int
	valid bool // 유효-무효 비트 역할
}

// NewSafeBuffer 안전한 버퍼 생성
func NewSafeBuffer(size int) *SafeBuffer {
	return &SafeBuffer{
		data:  make([]byte, size),
		size:  size,
		valid: true, // 초기에는 유효한 상태
	}
}

// Write 데이터 쓰기 (경계 검사 포함)
func (s *SafeBuffer) Write(offset int, data []byte) error {
	// 유효성 검사 (무효 상태 접근 시 오류)
	if !s.valid {
		return errors.New("메모리 보호: 무효한 버퍼에 접근 시도")
	}
	
	// 경계 검사 (버퍼 오버플로우 방지)
	if offset < 0 || offset+len(data) > s.size {
		return fmt.Errorf("메모리 보호: 버퍼 오버플로우 (offset: %d, size: %d, buffer_size: %d)", 
			offset, len(data), s.size)
	}
	
	copy(s.data[offset:], data)
	return nil
}

// Read 데이터 읽기
func (s *SafeBuffer) Read(offset, length int) ([]byte, error) {
	if !s.valid {
		return nil, errors.New("메모리 보호: 무효한 버퍼에서 읽기 시도")
	}
	
	if offset < 0 || offset+length > s.size {
		return nil, fmt.Errorf("메모리 보호: 읽기 범위 초과 (offset: %d, length: %d, buffer_size: %d)", 
			offset, length, s.size)
	}
	
	return s.data[offset:offset+length], nil
}

// Invalidate 버퍼 무효화 (메모리 해제 시뮬레이션)
func (s *SafeBuffer) Invalidate() {
	s.valid = false
	fmt.Println("버퍼가 무효화되었습니다 (메모리 해제)")
}

// IsValid 유효성 확인
func (s *SafeBuffer) IsValid() bool {
	return s.valid
}

// 실무 적용 예제 1: 웹 업로드 처리
func HandleFileUpload(w http.ResponseWriter, r *http.Request) {
	// 1MB 제한 버퍼 생성
	buffer := NewSafeBuffer(1024 * 1024)
	
	// 요청 본문 읽기
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "파일 읽기 오류", http.StatusBadRequest)
		return
	}
	
	// 안전한 버퍼에 쓰기
	if err := buffer.Write(0, data); err != nil {
		http.Error(w, fmt.Sprintf("업로드 오류: %v", err), http.StatusBadRequest)
		return
	}
	
	// 처리 완료 후 버퍼 무효화
	defer buffer.Invalidate()
	
	fmt.Fprintf(w, "파일 업로드 성공: %d 바이트", len(data))
}

// 실무 적용 예제 2: 데이터 파싱
func ParseProtocolData(rawData []byte) error {
	buffer := NewSafeBuffer(len(rawData))
	
	// 데이터 복사
	if err := buffer.Write(0, rawData); err != nil {
		return fmt.Errorf("데이터 파싱 오류: %v", err)
	}
	
	// 헤더 읽기 (처음 8바이트)
	header, err := buffer.Read(0, 8)
	if err != nil {
		return fmt.Errorf("헤더 읽기 오류: %v", err)
	}
	
	fmt.Printf("프로토콜 헤더: %x\n", header)
	
	// 페이로드 읽기 (나머지 데이터)
	if len(rawData) > 8 {
		payload, err := buffer.Read(8, len(rawData)-8)
		if err != nil {
			return fmt.Errorf("페이로드 읽기 오류: %v", err)
		}
		fmt.Printf("페이로드 크기: %d 바이트\n", len(payload))
	}
	
	return nil
}

func main() {
	fmt.Println("=== SafeBuffer 예제 (유효-무효 비트 개념 적용) ===")
	
	// 예제 1: 정상적인 사용
	fmt.Println("\n1. 정상적인 버퍼 사용:")
	buffer := NewSafeBuffer(100)
	
	data := []byte("Hello, World!")
	if err := buffer.Write(0, data); err != nil {
		fmt.Printf("쓰기 오류: %v\n", err)
	} else {
		fmt.Println("데이터 쓰기 성공")
	}
	
	readData, err := buffer.Read(0, len(data))
	if err != nil {
		fmt.Printf("읽기 오류: %v\n", err)
	} else {
		fmt.Printf("읽은 데이터: %s\n", string(readData))
	}
	
	// 예제 2: 버퍼 오버플로우 시도
	fmt.Println("\n2. 버퍼 오버플로우 시도:")
	largeData := make([]byte, 200) // 버퍼 크기(100)보다 큰 데이터
	if err := buffer.Write(0, largeData); err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 예제 3: 무효화된 버퍼 접근
	fmt.Println("\n3. 무효화된 버퍼 접근:")
	buffer.Invalidate()
	
	if err := buffer.Write(0, data); err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 예제 4: 프로토콜 데이터 파싱
	fmt.Println("\n4. 프로토콜 데이터 파싱:")
	protocolData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x41, 0x42, 0x43}
	if err := ParseProtocolData(protocolData); err != nil {
		fmt.Printf("파싱 오류: %v\n", err)
	}
}