package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// SlabAllocator는 커널의 SLAB 할당자 개념을 구현
type SlabAllocator struct {
	small  sync.Pool // 1KB 미만 요청
	medium sync.Pool // 4KB 미만 요청  
	large  sync.Pool // 16KB 미만 요청
}

func NewSlabAllocator() *SlabAllocator {
	return &SlabAllocator{
		small: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
		medium: sync.Pool{
			New: func() interface{} {
				return make([]byte, 4096)
			},
		},
		large: sync.Pool{
			New: func() interface{} {
				return make([]byte, 16384)
			},
		},
	}
}

func (sa *SlabAllocator) Get(size int) []byte {
	switch {
	case size <= 1024:
		return sa.small.Get().([]byte)[:size]
	case size <= 4096:
		return sa.medium.Get().([]byte)[:size]
	case size <= 16384:
		return sa.large.Get().([]byte)[:size]
	default:
		return make([]byte, size)
	}
}

func (sa *SlabAllocator) Put(buf []byte) {
	switch cap(buf) {
	case 1024:
		sa.small.Put(buf[:1024])
	case 4096:
		sa.medium.Put(buf[:4096])
	case 16384:
		sa.large.Put(buf[:16384])
	}
}

// ResponseBufferPool은 HTTP 응답 버퍼 관리
type ResponseBufferPool struct {
	allocator *SlabAllocator
}

func NewResponseBufferPool() *ResponseBufferPool {
	return &ResponseBufferPool{
		allocator: NewSlabAllocator(),
	}
}

func (rbp *ResponseBufferPool) GetBuffer(size int) []byte {
	return rbp.allocator.Get(size)
}

func (rbp *ResponseBufferPool) PutBuffer(buf []byte) {
	rbp.allocator.Put(buf)
}

// 실무용 HTTP 서버
type HighPerformanceServer struct {
	bufferPool *ResponseBufferPool
	metrics    *ServerMetrics
}

type ServerMetrics struct {
	requestCount    int64
	allocatedMemory int64
	gcCount         uint32
	mutex           sync.RWMutex
}

func (sm *ServerMetrics) IncrementRequest() {
	sm.mutex.Lock()
	sm.requestCount++
	sm.mutex.Unlock()
}

func (sm *ServerMetrics) UpdateMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	sm.mutex.Lock()
	sm.allocatedMemory = int64(m.Alloc)
	sm.gcCount = m.NumGC
	sm.mutex.Unlock()
}

func (sm *ServerMetrics) GetStats() (int64, int64, uint32) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.requestCount, sm.allocatedMemory, sm.gcCount
}

func NewHighPerformanceServer() *HighPerformanceServer {
	return &HighPerformanceServer{
		bufferPool: NewResponseBufferPool(),
		metrics:    &ServerMetrics{},
	}
}

// JSON 응답 생성 (메모리 풀 사용)
func (hps *HighPerformanceServer) writeJSONResponse(w http.ResponseWriter, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	// 메모리 풀에서 버퍼 가져오기
	bufferSize := len(jsonData) + 100 // 헤더용 여유 공간
	buffer := hps.bufferPool.GetBuffer(bufferSize)
	defer hps.bufferPool.PutBuffer(buffer)
	
	// 버퍼에 응답 작성
	copy(buffer, jsonData)
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(buffer[:len(jsonData)])
	
	return nil
}

// API 핸들러들
func (hps *HighPerformanceServer) handleUserData(w http.ResponseWriter, r *http.Request) {
	hps.metrics.IncrementRequest()
	
	// 실무에서 흔한 사용자 데이터 응답
	userData := map[string]interface{}{
		"id":       12345,
		"username": "john_doe",
		"email":    "john@example.com",
		"profile": map[string]interface{}{
			"name":     "John Doe",
			"age":      30,
			"location": "Seoul, Korea",
			"bio":      "Software Engineer with 5+ years experience",
		},
		"preferences": map[string]interface{}{
			"theme":    "dark",
			"language": "ko",
			"timezone": "Asia/Seoul",
		},
		"timestamp": time.Now().Unix(),
	}
	
	hps.writeJSONResponse(w, userData)
}

func (hps *HighPerformanceServer) handleBulkData(w http.ResponseWriter, r *http.Request) {
	hps.metrics.IncrementRequest()
	
	// 대용량 데이터 시뮬레이션
	var items []map[string]interface{}
	for i := 0; i < 1000; i++ {
		items = append(items, map[string]interface{}{
			"id":          i,
			"title":       fmt.Sprintf("Item %d", i),
			"description": "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
			"price":       float64(i * 10),
			"category":    "electronics",
		})
	}
	
	response := map[string]interface{}{
		"total_count": len(items),
		"items":       items,
		"page":        1,
		"per_page":    1000,
	}
	
	hps.writeJSONResponse(w, response)
}

func (hps *HighPerformanceServer) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	hps.metrics.IncrementRequest()
	
	// 파일 업로드 처리 (메모리 풀 사용)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "파일을 읽을 수 없습니다", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	// 파일 크기만큼 버퍼 할당
	buffer := hps.bufferPool.GetBuffer(int(header.Size))
	defer hps.bufferPool.PutBuffer(buffer)
	
	// 파일 내용을 버퍼로 읽기
	bytesRead, err := io.ReadFull(file, buffer[:header.Size])
	if err != nil && err != io.ErrUnexpectedEOF {
		http.Error(w, "파일 읽기 오류", http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"filename":   header.Filename,
		"size":       bytesRead,
		"status":     "uploaded",
		"timestamp":  time.Now().Unix(),
	}
	
	hps.writeJSONResponse(w, response)
}

func (hps *HighPerformanceServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	hps.metrics.UpdateMemoryStats()
	requests, memory, gcCount := hps.metrics.GetStats()
	
	metrics := map[string]interface{}{
		"total_requests":    requests,
		"allocated_memory":  memory,
		"gc_count":         gcCount,
		"timestamp":        time.Now().Unix(),
		"memory_mb":        float64(memory) / 1024 / 1024,
	}
	
	hps.writeJSONResponse(w, metrics)
}

func main() {
	server := NewHighPerformanceServer()
	
	// 라우트 설정
	http.HandleFunc("/api/user", server.handleUserData)
	http.HandleFunc("/api/bulk", server.handleBulkData)
	http.HandleFunc("/api/upload", server.handleFileUpload)
	http.HandleFunc("/api/metrics", server.handleMetrics)
	
	// 메모리 모니터링 고루틴
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			server.metrics.UpdateMemoryStats()
			requests, memory, gcCount := server.metrics.GetStats()
			log.Printf("Requests: %d, Memory: %.2f MB, GC: %d", 
				requests, float64(memory)/1024/1024, gcCount)
		}
	}()
	
	log.Println("슬랩 할당자 기반 고성능 HTTP 서버 시작 - :8080")
	log.Println("테스트 URL:")
	log.Println("  GET  /api/user     - 사용자 데이터")
	log.Println("  GET  /api/bulk     - 대용량 데이터")
	log.Println("  POST /api/upload   - 파일 업로드")
	log.Println("  GET  /api/metrics  - 서버 메트릭")
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("서버 시작 실패:", err)
	}
}