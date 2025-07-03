package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// MemoryPool 메모리 풀 구조체 (내부 단편화 방지)
type MemoryPool struct {
	pool     sync.Pool
	size     int
	allocated int64
	reused   int64
	mu       sync.RWMutex
}

// NewMemoryPool 메모리 풀 생성
func NewMemoryPool(size int) *MemoryPool {
	return &MemoryPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get 메모리 블록 가져오기
func (mp *MemoryPool) Get() []byte {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	buffer := mp.pool.Get().([]byte)
	
	// 풀에서 재사용된 경우
	if len(buffer) == mp.size {
		mp.reused++
		fmt.Printf("메모리 재사용: %d 바이트\n", mp.size)
	} else {
		mp.allocated++
		fmt.Printf("새 메모리 할당: %d 바이트\n", mp.size)
	}
	
	// 버퍼 초기화
	for i := range buffer {
		buffer[i] = 0
	}
	
	return buffer
}

// Put 메모리 블록 반환
func (mp *MemoryPool) Put(buffer []byte) {
	if len(buffer) != mp.size {
		fmt.Printf("잘못된 크기의 버퍼 반환: %d (예상: %d)\n", len(buffer), mp.size)
		return
	}
	
	mp.pool.Put(buffer)
	fmt.Printf("메모리 반환: %d 바이트\n", mp.size)
}

// Stats 메모리 풀 통계
func (mp *MemoryPool) Stats() (allocated, reused int64, reuseRate float64) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	
	total := mp.allocated + mp.reused
	if total == 0 {
		return mp.allocated, mp.reused, 0.0
	}
	
	return mp.allocated, mp.reused, float64(mp.reused)/float64(total)*100
}

// JSONProcessor JSON 처리기 (메모리 풀 사용)
type JSONProcessor struct {
	bufferPool *MemoryPool
}

// NewJSONProcessor JSON 처리기 생성
func NewJSONProcessor(bufferSize int) *JSONProcessor {
	return &JSONProcessor{
		bufferPool: NewMemoryPool(bufferSize),
	}
}

// ProcessJSON JSON 데이터 처리
func (jp *JSONProcessor) ProcessJSON(data []byte) ([]byte, error) {
	// 메모리 풀에서 버퍼 가져오기
	buffer := jp.bufferPool.Get()
	defer jp.bufferPool.Put(buffer)
	
	// JSON 처리 시뮬레이션
	fmt.Printf("JSON 처리 중: %d 바이트\n", len(data))
	time.Sleep(10 * time.Millisecond)
	
	// 처리된 데이터 복사
	processedData := make([]byte, len(data))
	copy(processedData, data)
	
	return processedData, nil
}

// HTTPServer HTTP 서버 (메모리 풀 사용)
type HTTPServer struct {
	requestPool  *MemoryPool
	responsePool *MemoryPool
}

// NewHTTPServer HTTP 서버 생성
func NewHTTPServer(requestSize, responseSize int) *HTTPServer {
	return &HTTPServer{
		requestPool:  NewMemoryPool(requestSize),
		responsePool: NewMemoryPool(responseSize),
	}
}

// HandleRequest 요청 처리
func (hs *HTTPServer) HandleRequest(requestData []byte) []byte {
	// 요청 버퍼 가져오기
	reqBuffer := hs.requestPool.Get()
	defer hs.requestPool.Put(reqBuffer)
	
	// 응답 버퍼 가져오기
	respBuffer := hs.responsePool.Get()
	defer hs.responsePool.Put(respBuffer)
	
	// 요청 데이터 처리
	fmt.Printf("HTTP 요청 처리: %d 바이트\n", len(requestData))
	
	// 요청 데이터 복사
	copy(reqBuffer, requestData)
	
	// 응답 데이터 생성
	response := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"status\":\"success\"}")
	copy(respBuffer, response)
	
	// 실제 응답 데이터 반환
	result := make([]byte, len(response))
	copy(result, response)
	
	return result
}

// DatabaseConnector 데이터베이스 연결자 (메모리 풀 사용)
type DatabaseConnector struct {
	queryPool  *MemoryPool
	resultPool *MemoryPool
}

// NewDatabaseConnector 데이터베이스 연결자 생성
func NewDatabaseConnector(querySize, resultSize int) *DatabaseConnector {
	return &DatabaseConnector{
		queryPool:  NewMemoryPool(querySize),
		resultPool: NewMemoryPool(resultSize),
	}
}

// ExecuteQuery 쿼리 실행
func (dc *DatabaseConnector) ExecuteQuery(query string) ([]byte, error) {
	// 쿼리 버퍼 가져오기
	queryBuffer := dc.queryPool.Get()
	defer dc.queryPool.Put(queryBuffer)
	
	// 결과 버퍼 가져오기
	resultBuffer := dc.resultPool.Get()
	defer dc.resultPool.Put(resultBuffer)
	
	// 쿼리 실행 시뮬레이션
	fmt.Printf("데이터베이스 쿼리 실행: %s\n", query)
	time.Sleep(20 * time.Millisecond)
	
	// 쿼리 데이터 복사
	copy(queryBuffer, []byte(query))
	
	// 결과 데이터 생성
	result := []byte(`[{"id":1,"name":"김철수"},{"id":2,"name":"이영희"}]`)
	copy(resultBuffer, result)
	
	// 실제 결과 반환
	actualResult := make([]byte, len(result))
	copy(actualResult, result)
	
	return actualResult, nil
}

// benchmarkMemoryUsage 메모리 사용량 벤치마크
func benchmarkMemoryUsage() {
	fmt.Println("\n=== 메모리 사용량 벤치마크 ===")
	
	// GC 실행 및 초기 메모리 상태 확인
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	fmt.Printf("초기 메모리: %d KB\n", m1.Alloc/1024)
	
	// 메모리 풀 사용
	fmt.Println("\n1. 메모리 풀 사용:")
	pool := NewMemoryPool(4096)
	
	start := time.Now()
	for i := 0; i < 1000; i++ {
		buffer := pool.Get()
		// 버퍼 사용 시뮬레이션
		buffer[0] = byte(i % 256)
		pool.Put(buffer)
	}
	poolTime := time.Since(start)
	
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	fmt.Printf("메모리 풀 사용 후: %d KB\n", m2.Alloc/1024)
	fmt.Printf("메모리 풀 실행 시간: %v\n", poolTime)
	
	// 직접 할당 방식
	fmt.Println("\n2. 직접 할당 방식:")
	start = time.Now()
	for i := 0; i < 1000; i++ {
		buffer := make([]byte, 4096)
		// 버퍼 사용 시뮬레이션
		buffer[0] = byte(i % 256)
		// 자동으로 GC 대상이 됨
	}
	directTime := time.Since(start)
	
	runtime.GC()
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)
	
	fmt.Printf("직접 할당 사용 후: %d KB\n", m3.Alloc/1024)
	fmt.Printf("직접 할당 실행 시간: %v\n", directTime)
	
	// 성능 비교
	fmt.Printf("\n성능 개선: %.2fx\n", float64(directTime)/float64(poolTime))
	
	// 메모리 풀 통계
	allocated, reused, reuseRate := pool.Stats()
	fmt.Printf("메모리 풀 통계 - 할당: %d, 재사용: %d, 재사용율: %.2f%%\n", allocated, reused, reuseRate)
}

func main() {
	fmt.Println("=== Memory Pool 예제 (내부 단편화 방지) ===")
	
	// 예제 1: JSON 처리기
	fmt.Println("\n1. JSON 처리기:")
	jsonProcessor := NewJSONProcessor(8192)
	
	jsonData := []byte(`{"name":"김철수","age":30,"email":"kim@example.com"}`)
	
	// 여러 번 JSON 처리
	for i := 0; i < 3; i++ {
		processedData, err := jsonProcessor.ProcessJSON(jsonData)
		if err != nil {
			fmt.Printf("JSON 처리 오류: %v\n", err)
		} else {
			fmt.Printf("처리 완료: %d 바이트\n", len(processedData))
		}
	}
	
	// 예제 2: HTTP 서버
	fmt.Println("\n2. HTTP 서버:")
	httpServer := NewHTTPServer(4096, 2048)
	
	requestData := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	
	// 여러 번 요청 처리
	for i := 0; i < 3; i++ {
		response := httpServer.HandleRequest(requestData)
		fmt.Printf("응답 생성: %d 바이트\n", len(response))
	}
	
	// 예제 3: 데이터베이스 연결자
	fmt.Println("\n3. 데이터베이스 연결자:")
	dbConnector := NewDatabaseConnector(1024, 4096)
	
	queries := []string{
		"SELECT * FROM users",
		"SELECT * FROM products",
		"SELECT * FROM orders",
	}
	
	for _, query := range queries {
		result, err := dbConnector.ExecuteQuery(query)
		if err != nil {
			fmt.Printf("쿼리 실행 오류: %v\n", err)
		} else {
			fmt.Printf("쿼리 결과: %d 바이트\n", len(result))
		}
	}
	
	// 예제 4: 메모리 사용량 벤치마크
	benchmarkMemoryUsage()
	
	// 예제 5: 메모리 풀 통계
	fmt.Println("\n4. 메모리 풀 통계:")
	allocated, reused, reuseRate := jsonProcessor.bufferPool.Stats()
	fmt.Printf("JSON 처리기 - 할당: %d, 재사용: %d, 재사용율: %.2f%%\n", allocated, reused, reuseRate)
	
	allocated, reused, reuseRate = httpServer.requestPool.Stats()
	fmt.Printf("HTTP 서버 (요청) - 할당: %d, 재사용: %d, 재사용율: %.2f%%\n", allocated, reused, reuseRate)
	
	allocated, reused, reuseRate = dbConnector.queryPool.Stats()
	fmt.Printf("DB 연결자 (쿼리) - 할당: %d, 재사용: %d, 재사용율: %.2f%%\n", allocated, reused, reuseRate)
}