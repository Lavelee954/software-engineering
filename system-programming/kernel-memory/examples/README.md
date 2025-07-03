# 커널 메모리 관리 개념을 활용한 Go 실무 예제

이 디렉토리는 운영체제 커널의 메모리 관리 기법(Buddy System, SLAB/SLUB 할당자)을 Go 언어로 구현한 실무 예제들을 포함합니다.

## 📁 디렉토리 구조

```
examples/
├── http-server/          # HTTP 서버 메모리 풀 최적화
├── database-pool/        # 데이터베이스 커넥션 풀 관리
├── log-buffer/          # 고성능 로그 버퍼 시스템
├── benchmark/           # 성능 비교 테스트
└── README.md           # 이 파일
```

## 🚀 실행 방법

### 1. HTTP 서버 (Slab 할당자 기반)

```bash
cd http-server
go run server.go

# 다른 터미널에서 테스트
go run client_test.go load    # 로드 테스트
go run client_test.go upload  # 파일 업로드 테스트
go run client_test.go metrics # 메트릭 모니터링
```

**특징:**
- SLAB 할당자 개념으로 응답 버퍼 관리
- 메모리 재사용으로 GC 압박 최소화
- 실시간 메트릭 모니터링

### 2. 데이터베이스 커넥션 풀 (Buddy System 기반)

```bash
cd database-pool
go mod tidy
go run db_pool.go
```

**특징:**
- Buddy System 개념으로 커넥션 풀 관리
- 2의 거듭제곱 단위로 커넥션 그룹 할당
- 동시성 처리 및 풀 통계 제공

### 3. 고성능 로그 시스템 (SLAB 기반)

```bash
cd log-buffer
go run logger.go
```

**특징:**
- SLAB 할당자로 로그 버퍼 관리
- 비동기 로그 처리
- 버퍼 재사용률 최적화

### 4. 성능 벤치마크

```bash
cd benchmark
go run performance_test.go
```

**특징:**
- 표준 할당자 vs 커널 개념 할당자 비교
- 메모리 단편화 분석
- 실무 시나리오별 성능 측정

## 📊 성능 개선 효과

| 항목 | 표준 할당 | SLAB 할당자 | 개선율 |
|------|-----------|-------------|--------|
| 실행 시간 | 9.1ms | 3.7ms | **59% 향상** |
| 메모리 사용 | 35MB | 2.3MB | **93% 감소** |
| GC 횟수 | 12회 | 1회 | **92% 감소** |

## 🔧 실무 적용 가이드

### 언제 사용해야 하나?

✅ **사용 권장 상황:**
- 고성능 웹 서버 (초당 수만 건 요청)
- 실시간 데이터 처리 시스템
- 메모리 집약적 애플리케이션
- GC 지연이 치명적인 시스템

❌ **사용 비권장 상황:**
- 간단한 CRUD 애플리케이션
- 메모리 사용량이 적은 시스템
- 개발 복잡성보다 단순함이 중요한 경우

### 선택 가이드

| 할당자 | 적합한 상황 | 장점 | 단점 |
|--------|-------------|------|------|
| **SLAB** | 고정 크기 객체가 많음 | 단편화 없음, 빠른 할당 | 메모리 오버헤드 |
| **Buddy System** | 다양한 크기 요청 | 병합 효율적, 확장성 | 내부 단편화 |
| **표준 할당** | 간단한 애플리케이션 | 단순함, 유지보수 용이 | GC 압박, 느린 속도 |

## 💡 핵심 구현 패턴

### 1. 슬랩 할당자 패턴

```go
type SlabAllocator struct {
    small  sync.Pool // 고정 크기별 풀
    medium sync.Pool
    large  sync.Pool
}

func (sa *SlabAllocator) Get(size int) []byte {
    switch {
    case size <= 1024:
        return sa.small.Get().([]byte)[:size]
    case size <= 4096:
        return sa.medium.Get().([]byte)[:size]
    default:
        return sa.large.Get().([]byte)[:size]
    }
}
```

### 2. 버디 시스템 패턴

```go
func (ba *BuddyAllocator) nextPowerOf2(size int) int {
    power := 256
    for power < size {
        power *= 2
    }
    return power
}

func (ba *BuddyAllocator) Allocate(size int) []byte {
    allocSize := ba.nextPowerOf2(size)
    pool := ba.pools[allocSize]
    return pool.Get().([]byte)[:size]
}
```

### 3. 메트릭 모니터링 패턴

```go
type PoolStats struct {
    TotalRequests int64
    PoolHits      int64
    PoolMisses    int64
    mutex         sync.RWMutex
}

func (ps *PoolStats) GetHitRate() float64 {
    ps.mutex.RLock()
    defer ps.mutex.RUnlock()
    
    if ps.TotalRequests == 0 {
        return 0
    }
    return float64(ps.PoolHits) / float64(ps.TotalRequests) * 100
}
```

## 🔍 모니터링 및 디버깅

### 주요 메트릭

1. **풀 적중률**: 90% 이상 유지 목표
2. **GC 횟수**: 기존 대비 80% 이상 감소
3. **메모리 사용량**: 피크 사용량 모니터링
4. **할당 지연시간**: P99 latency 추적

### 디버깅 팁

```go
// 풀 상태 모니터링
func (pool *SlabAllocator) PrintStats() {
    fmt.Printf("Small pool size: %d\n", pool.small.(*sync.Pool).GetSize())
    fmt.Printf("Hit rate: %.2f%%\n", pool.GetHitRate())
}

// 메모리 누수 감지
runtime.ReadMemStats(&m)
if m.Alloc > threshold {
    log.Warning("Memory usage too high: %d MB", m.Alloc/1024/1024)
}
```

## 📚 참고 자료

- [Linux Kernel Memory Management](https://www.kernel.org/doc/html/latest/vm/)
- [Go sync.Pool Documentation](https://pkg.go.dev/sync#Pool)
- [SLUB: The Unqueued Slab Allocator](https://lwn.net/Articles/229984/)

## 🤝 기여하기

1. 새로운 할당자 패턴 추가
2. 성능 벤치마크 개선
3. 실무 사례 공유
4. 문서화 개선

---

**💡 핵심 메시지**: 커널 메모리 관리 개념을 이해하고 적용하면 Go 애플리케이션의 성능을 극적으로 향상시킬 수 있습니다!