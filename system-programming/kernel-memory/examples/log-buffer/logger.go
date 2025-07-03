package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel 타입 정의
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// SlabBufferAllocator는 로그 버퍼를 위한 슬랩 할당자
type SlabBufferAllocator struct {
	small  sync.Pool // 512바이트 - 일반 로그
	medium sync.Pool // 2KB - 상세 로그
	large  sync.Pool // 8KB - 스택 트레이스 등
}

func NewSlabBufferAllocator() *SlabBufferAllocator {
	return &SlabBufferAllocator{
		small: sync.Pool{
			New: func() interface{} {
				return make([]byte, 512)
			},
		},
		medium: sync.Pool{
			New: func() interface{} {
				return make([]byte, 2048)
			},
		},
		large: sync.Pool{
			New: func() interface{} {
				return make([]byte, 8192)
			},
		},
	}
}

func (sba *SlabBufferAllocator) GetBuffer(size int) []byte {
	switch {
	case size <= 512:
		return sba.small.Get().([]byte)[:size]
	case size <= 2048:
		return sba.medium.Get().([]byte)[:size]
	case size <= 8192:
		return sba.large.Get().([]byte)[:size]
	default:
		return make([]byte, size)
	}
}

func (sba *SlabBufferAllocator) PutBuffer(buf []byte) {
	switch cap(buf) {
	case 512:
		sba.small.Put(buf[:512])
	case 2048:
		sba.medium.Put(buf[:2048])
	case 8192:
		sba.large.Put(buf[:8192])
	}
}

// LogEntry는 개별 로그 항목
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	File      string
	Line      int
	Goroutine int
	Buffer    []byte
}

// HighPerformanceLogger는 커널 메모리 관리 개념을 적용한 로거
type HighPerformanceLogger struct {
	allocator    *SlabBufferAllocator
	logChannel   chan *LogEntry
	writers      []io.Writer
	level        LogLevel
	bufferSize   int
	maxGoroutines int
	wg           sync.WaitGroup
	done         chan bool
	stats        *LoggerStats
}

type LoggerStats struct {
	TotalLogs     int64
	DroppedLogs   int64
	BufferReuse   int64
	BufferAlloc   int64
	mutex         sync.RWMutex
}

func (ls *LoggerStats) IncrementTotal() {
	ls.mutex.Lock()
	ls.TotalLogs++
	ls.mutex.Unlock()
}

func (ls *LoggerStats) IncrementDropped() {
	ls.mutex.Lock()
	ls.DroppedLogs++
	ls.mutex.Unlock()
}

func (ls *LoggerStats) IncrementReuse() {
	ls.mutex.Lock()
	ls.BufferReuse++
	ls.mutex.Unlock()
}

func (ls *LoggerStats) IncrementAlloc() {
	ls.mutex.Lock()
	ls.BufferAlloc++
	ls.mutex.Unlock()
}

func (ls *LoggerStats) GetStats() (int64, int64, int64, int64) {
	ls.mutex.RLock()
	defer ls.mutex.RUnlock()
	return ls.TotalLogs, ls.DroppedLogs, ls.BufferReuse, ls.BufferAlloc
}

func NewHighPerformanceLogger(writers []io.Writer, level LogLevel) *HighPerformanceLogger {
	logger := &HighPerformanceLogger{
		allocator:     NewSlabBufferAllocator(),
		logChannel:    make(chan *LogEntry, 10000), // 비동기 버퍼
		writers:       writers,
		level:         level,
		bufferSize:    1000,
		maxGoroutines: runtime.NumCPU(),
		done:          make(chan bool),
		stats:         &LoggerStats{},
	}
	
	// 로그 처리 워커 시작
	for i := 0; i < logger.maxGoroutines; i++ {
		logger.wg.Add(1)
		go logger.worker()
	}
	
	return logger
}

func (hpl *HighPerformanceLogger) worker() {
	defer hpl.wg.Done()
	
	for {
		select {
		case entry := <-hpl.logChannel:
			hpl.processLogEntry(entry)
		case <-hpl.done:
			return
		}
	}
}

func (hpl *HighPerformanceLogger) processLogEntry(entry *LogEntry) {
	defer func() {
		// 버퍼를 풀로 반환
		if entry.Buffer != nil {
			hpl.allocator.PutBuffer(entry.Buffer)
			hpl.stats.IncrementReuse()
		}
	}()
	
	// 로그 포맷팅
	logMessage := fmt.Sprintf("[%s] %s %s:%d [G%d] %s\n",
		entry.Timestamp.Format("2006-01-02 15:04:05.000"),
		entry.Level.String(),
		entry.File,
		entry.Line,
		entry.Goroutine,
		entry.Message,
	)
	
	// 모든 writer에 출력
	for _, writer := range hpl.writers {
		writer.Write([]byte(logMessage))
	}
}

func (hpl *HighPerformanceLogger) log(level LogLevel, format string, args ...interface{}) {
	if level < hpl.level {
		return
	}
	
	hpl.stats.IncrementTotal()
	
	// 호출자 정보 획득
	_, file, line, _ := runtime.Caller(2)
	goroutineID := runtime.NumGoroutine()
	
	// 메시지 포맷팅
	message := fmt.Sprintf(format, args...)
	
	// 예상 버퍼 크기 계산
	estimatedSize := len(message) + 100 // 타임스탬프, 레벨 등 추가 정보
	
	// 슬랩 할당자에서 버퍼 가져오기
	buffer := hpl.allocator.GetBuffer(estimatedSize)
	hpl.stats.IncrementAlloc()
	
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		File:      file,
		Line:      line,
		Goroutine: goroutineID,
		Buffer:    buffer,
	}
	
	// 비동기 처리
	select {
	case hpl.logChannel <- entry:
		// 성공적으로 큐에 추가
	default:
		// 큐가 가득 찬 경우 드롭
		hpl.stats.IncrementDropped()
		hpl.allocator.PutBuffer(buffer)
	}
}

// 공개 로깅 메서드들
func (hpl *HighPerformanceLogger) Debug(format string, args ...interface{}) {
	hpl.log(DEBUG, format, args...)
}

func (hpl *HighPerformanceLogger) Info(format string, args ...interface{}) {
	hpl.log(INFO, format, args...)
}

func (hpl *HighPerformanceLogger) Warn(format string, args ...interface{}) {
	hpl.log(WARN, format, args...)
}

func (hpl *HighPerformanceLogger) Error(format string, args ...interface{}) {
	hpl.log(ERROR, format, args...)
}

func (hpl *HighPerformanceLogger) Fatal(format string, args ...interface{}) {
	hpl.log(FATAL, format, args...)
	hpl.Close()
	os.Exit(1)
}

func (hpl *HighPerformanceLogger) Close() {
	close(hpl.done)
	hpl.wg.Wait()
	close(hpl.logChannel)
	
	// 남은 로그 처리
	for entry := range hpl.logChannel {
		hpl.processLogEntry(entry)
	}
}

func (hpl *HighPerformanceLogger) GetStats() (int64, int64, int64, int64) {
	return hpl.stats.GetStats()
}

// 실무 시나리오: 웹 서버 로깅
type WebServer struct {
	logger *HighPerformanceLogger
}

func NewWebServer() *WebServer {
	// 파일과 콘솔에 동시 출력
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("로그 파일 생성 실패:", err)
	}
	
	writers := []io.Writer{os.Stdout, logFile}
	logger := NewHighPerformanceLogger(writers, INFO)
	
	return &WebServer{logger: logger}
}

func (ws *WebServer) HandleRequest(userID int, endpoint string, duration time.Duration) {
	ws.logger.Info("Request: user=%d endpoint=%s duration=%v", userID, endpoint, duration)
	
	// 에러 시뮬레이션
	if userID%100 == 0 {
		ws.logger.Error("Database connection failed for user %d", userID)
	}
	
	// 디버그 정보 (높은 로그 레벨에서만 출력)
	ws.logger.Debug("Processing details: user=%d internal_state=processing", userID)
}

func (ws *WebServer) SimulateTraffic(requestCount int, concurrency int) {
	var wg sync.WaitGroup
	
	start := time.Now()
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			requestsPerWorker := requestCount / concurrency
			for j := 0; j < requestsPerWorker; j++ {
				userID := workerID*requestsPerWorker + j
				endpoint := fmt.Sprintf("/api/users/%d", userID%10)
				duration := time.Duration(j%100) * time.Millisecond
				
				ws.HandleRequest(userID, endpoint, duration)
				
				// CPU 부하 시뮬레이션
				if j%1000 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	ws.logger.Info("Traffic simulation completed: %d requests in %v (%.2f req/sec)",
		requestCount, duration, float64(requestCount)/duration.Seconds())
}

func (ws *WebServer) PrintStats() {
	total, dropped, reuse, alloc := ws.logger.GetStats()
	
	fmt.Printf("\n=== 로거 통계 ===\n")
	fmt.Printf("총 로그: %d\n", total)
	fmt.Printf("드롭된 로그: %d\n", dropped)
	fmt.Printf("버퍼 재사용: %d\n", reuse)
	fmt.Printf("버퍼 할당: %d\n", alloc)
	if alloc > 0 {
		fmt.Printf("재사용률: %.2f%%\n", float64(reuse)/float64(alloc)*100)
	}
	if total > 0 {
		fmt.Printf("드롭률: %.2f%%\n", float64(dropped)/float64(total)*100)
	}
}

func (ws *WebServer) Close() {
	ws.logger.Close()
}

// 성능 비교용 일반 로거
type SimpleLogger struct {
	writers []io.Writer
	level   LogLevel
	mutex   sync.Mutex
	stats   struct {
		totalLogs int64
		mutex     sync.RWMutex
	}
}

func NewSimpleLogger(writers []io.Writer, level LogLevel) *SimpleLogger {
	return &SimpleLogger{
		writers: writers,
		level:   level,
	}
}

func (sl *SimpleLogger) log(level LogLevel, format string, args ...interface{}) {
	if level < sl.level {
		return
	}
	
	sl.stats.mutex.Lock()
	sl.stats.totalLogs++
	sl.stats.mutex.Unlock()
	
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	
	_, file, line, _ := runtime.Caller(2)
	message := fmt.Sprintf(format, args...)
	
	logMessage := fmt.Sprintf("[%s] %s %s:%d %s\n",
		time.Now().Format("2006-01-02 15:04:05.000"),
		level.String(),
		file,
		line,
		message,
	)
	
	for _, writer := range sl.writers {
		writer.Write([]byte(logMessage))
	}
}

func (sl *SimpleLogger) Info(format string, args ...interface{}) {
	sl.log(INFO, format, args...)
}

func main() {
	// 1. 고성능 로거 테스트
	fmt.Println("=== 고성능 로거 (커널 메모리 관리 개념 적용) ===")
	
	webServer := NewWebServer()
	defer webServer.Close()
	
	// 트래픽 시뮬레이션
	start := time.Now()
	webServer.SimulateTraffic(10000, 50)
	highPerfDuration := time.Since(start)
	
	webServer.PrintStats()
	fmt.Printf("고성능 로거 소요시간: %v\n", highPerfDuration)
	
	// 2. 일반 로거와 비교
	fmt.Println("\n=== 일반 로거 (비교용) ===")
	
	simpleLogger := NewSimpleLogger([]io.Writer{io.Discard}, INFO)
	
	start = time.Now()
	var wg sync.WaitGroup
	
	requestCount := 10000
	concurrency := 50
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			requestsPerWorker := requestCount / concurrency
			for j := 0; j < requestsPerWorker; j++ {
				userID := workerID*requestsPerWorker + j
				endpoint := fmt.Sprintf("/api/users/%d", userID%10)
				duration := time.Duration(j%100) * time.Millisecond
				
				simpleLogger.Info("Request: user=%d endpoint=%s duration=%v", userID, endpoint, duration)
			}
		}(i)
	}
	
	wg.Wait()
	simpleDuration := time.Since(start)
	
	fmt.Printf("일반 로거 소요시간: %v\n", simpleDuration)
	fmt.Printf("성능 향상: %.2fx\n", float64(simpleDuration)/float64(highPerfDuration))
}