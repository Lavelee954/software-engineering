# Go Concurrency Pattern Comparison Tests

이 디렉토리는 `concurrent-eng.md`에서 설명된 Go 동시성 패턴들을 "좋은 방법"과 "나쁜 방법"으로 비교하는 테스트 코드들을 포함합니다.

## 파일 구조

- `concurrent-eng.md` - Go 동시성 패턴 가이드 (원본 문서)
- `comparison_test.go` - 기본적인 패턴 비교 테스트
- `examples.go` - 구체적인 패턴 구현 예제
- `examples_test.go` - 구체적인 패턴들의 테스트
- `demo/main.go` - 패턴 비교 요약 데모 프로그램
- `go.mod` - Go 모듈 파일
- `README.md` - 이 파일

## 비교되는 패턴들

### 1. 기본 Goroutine 관리
- **BAD**: 동기화 없이 goroutine 실행
- **GOOD**: 채널을 사용한 동기화

### 2. 검색 서비스 (Google Search 패턴)
- **BAD**: 순차적 검색 (느림)
- **GOOD**: 병렬 검색
- **BETTER**: 타임아웃이 있는 병렬 검색

### 3. Worker Pool 패턴
- **BAD**: 작업마다 goroutine 생성 (리소스 낭비)
- **GOOD**: 제한된 수의 worker로 처리

### 4. Context 사용
- **BAD**: 취소 지원 없음
- **GOOD**: Context를 통한 취소 지원

### 5. 채널 버퍼링
- **BAD**: 데드락 가능성이 있는 unbuffered 채널
- **GOOD**: 적절한 버퍼 크기의 채널

### 6. Generator 패턴
- **BAD**: 단순한 goroutine
- **GOOD**: 채널을 반환하는 generator

### 7. Fan-In 패턴
- **BAD**: 여러 채널을 개별적으로 처리
- **GOOD**: 여러 채널을 하나로 합치는 fan-in

### 8. Timeout 패턴
- **BAD**: 타임아웃 없음 (무한 대기 가능)
- **GOOD**: select와 time.After를 사용한 타임아웃

### 9. Quit 채널 패턴
- **BAD**: 종료 신호 없음
- **GOOD**: quit 채널을 통한 graceful shutdown

### 10. Ping-Pong 패턴
- **BAD**: 공유 메모리와 mutex 사용
- **GOOD**: 채널 기반 통신

## 실행 방법

### 데모 프로그램 실행
```bash
# 패턴 비교 요약 보기
cd demo && go run main.go
```

### 테스트 실행

#### 모든 테스트 실행
```bash
go test -v
```

### 특정 패턴 테스트
```bash
# Goroutine 비교 테스트
go test -v -run TestGoroutineComparison

# 검색 비교 테스트
go test -v -run TestSearchComparison

# Worker Pool 비교 테스트
go test -v -run TestWorkerPoolComparison

# Context 비교 테스트
go test -v -run TestContextComparison
```

### 벤치마크 테스트
```bash
# 모든 벤치마크 실행
go test -bench .

# 특정 벤치마크 실행
go test -bench BenchmarkSequentialSearch
go test -bench BenchmarkParallelSearch
```

### Race Condition 테스트
```bash
# Race detector와 함께 실행
go test -race -v

# 특정 race condition 테스트
go test -race -v -run TestRaceConditions
```

### 구체적인 패턴 테스트
```bash
# Boring 패턴 테스트
go test -v -run TestBoringPatterns

# Fan-In 패턴 테스트
go test -v -run TestFanInPatterns

# Timeout 패턴 테스트
go test -v -run TestTimeoutPatterns

# Google Search 진화 테스트
go test -v -run TestGoogleSearchEvolution
```

## 주요 학습 포인트

### 1. 동시성 vs 병렬성
- 동시성: 여러 작업을 동시에 다루는 능력
- 병렬성: 여러 작업을 실제로 동시에 실행하는 능력

### 2. 채널 설계 원칙
- **Unbuffered 채널**: 동기식 통신과 시그널링용
- **Buffered 채널**: 처리량 최적화와 백프레셔 제어용
- **Receive-only 채널**: generator 패턴과 API 안전성용
- **Send-only 채널**: producer 인터페이스용

### 3. Goroutine 관리
- 항상 goroutine의 생명주기를 계획하세요
- 취소를 위해 context를 사용하세요
- graceful shutdown을 구현하세요
- goroutine leak을 방지하세요

### 4. 에러 처리 패턴
- 전용 에러 채널 사용
- 타임아웃이 있는 context 사용
- 첫 번째 에러가 우선하는 패턴
- 부분적 결과를 위한 graceful degradation

### 5. 성능 고려사항
- Worker pool로 제한된 병렬성
- 생산자/소비자 비율에 따른 채널 버퍼링
- 오버플로우 보호를 위한 ring buffer
- 최적화를 위한 프로파일링 사용

## 성능 비교 예상 결과

### 순차 vs 병렬 검색
```
Sequential search: ~300ms (100ms × 3)
Parallel search: ~100ms (max of 3 parallel operations)
Parallel with timeout: ~150ms (timeout limit)
```

### Worker Pool vs 개별 Goroutine
```
Individual goroutines: 높은 메모리 사용량, 컨텍스트 스위칭 오버헤드
Worker pool: 일정한 메모리 사용량, 효율적인 작업 처리
```

### 채널 버퍼링
```
Unbuffered: 동기식, 각 send/receive가 blocking
Buffered: 비동기식, 버퍼 크기까지 non-blocking
```

## 일반적인 실수들

1. **Goroutine 누수**: quit 채널이나 context 없이 무한 루프
2. **Race condition**: 공유 변수에 대한 동기화 없는 접근
3. **Deadlock**: 채널 읽기/쓰기 불균형
4. **Context 무시**: 긴 실행 작업에서 취소 지원 없음
5. **과도한 병렬성**: 리소스 제한 없이 goroutine 생성

## 추가 학습 자료

- [Go Concurrency Patterns](https://talks.golang.org/2012/concurrency.slide)
- [Advanced Go Concurrency](https://talks.golang.org/2013/advconc.slide)
- [Go Memory Model](https://golang.org/ref/mem)
- [Effective Go - Concurrency](https://golang.org/doc/effective_go.html#concurrency)

이 테스트들을 통해 Go의 동시성 패턴을 실제로 비교하고 학습할 수 있습니다. 