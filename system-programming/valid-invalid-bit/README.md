# 유효-무효 비트 (Valid-Invalid Bit) 메모리 관리 시뮬레이터

## 개념 설명

### 유효-무효 비트란?
유효-무효 비트는 운영체제의 메모리 관리에서 **메모리 보호**와 **요구 페이징**을 구현하는 핵심 메커니즘입니다.

### 주요 역할

#### 1. 메모리 보호
- **유효(Valid)**: 해당 페이지가 프로세스의 합법적인 주소 공간에 있음
- **무효(Invalid)**: 불법적인 주소이거나 현재 물리 메모리에 없는 상태
- 무효 페이지 접근 시 → **하드웨어 트랩 발생** → 메모리 보호 위반 감지

#### 2. 요구 페이징 (Demand Paging)
- 필요한 페이지만 물리 메모리에 로드
- 무효 페이지 접근 시 → **페이지 폴트 발생** → OS가 페이지를 메모리로 로드
- 사용자에게는 투명한 메모리 관리 제공

#### 3. 내부 단편화 관리
- 페이지 크기로 인한 메모리 낭비 최소화
- 효율적인 메모리 할당 및 해제

## 프로젝트 구조

```
valid-invalid-bit/
├── README.md           # 프로젝트 설명서
├── work.md            # 이론적 배경 설명
├── go.mod             # Go 모듈 설정
├── page_table.go      # 페이지 테이블 및 메모리 관리 구현
├── main.go            # 시뮬레이션 실행 코드
└── examples/          # 실무 적용 예제들
    ├── safe_buffer.go     # 메모리 안전성 패턴
    ├── lazy_resource.go   # 지연 로딩 패턴
    ├── lru_cache.go       # LRU 캐시 구현
    ├── memory_pool.go     # 메모리 풀링
    └── boundary_check.go  # 경계 검사 패턴
```

## 실행 방법

```bash
# 시뮬레이터 실행
go run .

# 특정 예제 실행
go run examples/safe_buffer.go
```

## 시뮬레이션 시나리오

### 시나리오 1: 기본 메모리 접근
- 초기 페이지 폴트 발생
- 페이지 테이블 업데이트
- 물리 메모리 할당

### 시나리오 2: 메모리 쓰기 및 더티 비트
- 페이지 수정 시 더티 비트 설정
- 스왑 공간 활용

### 시나리오 3: 메모리 부족 상황
- 페이지 교체 알고리즘 (LRU)
- 스왑 공간 관리
- 세그멘테이션 오류 처리

## 실무 적용 패턴

### 1. 메모리 안전성 보장
```go
// 버퍼 오버플로우 방지
type SafeBuffer struct {
    data []byte
    valid bool
}
```

### 2. 지연 로딩 (Lazy Loading)
```go
// 필요할 때만 리소스 로드
type LazyResource struct {
    loaded bool
    loader func() ([]byte, error)
}
```

### 3. 캐시 시스템
```go
// LRU 기반 캐시 구현
type LRUCache struct {
    capacity int
    items map[string]*Item
}
```

### 4. 메모리 풀링
```go
// 메모리 재사용으로 GC 부담 감소
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}
```

### 5. 경계 검사
```go
// 배열/슬라이스 안전한 접근
func SafeSliceAccess(slice []int, index int) (int, error) {
    if index < 0 || index >= len(slice) {
        return 0, fmt.Errorf("index out of bounds")
    }
    return slice[index], nil
}
```

## 실무 적용 상황

| 패턴 | 적용 상황 | 예시 |
|------|-----------|------|
| SafeBuffer | 네트워크/파일 처리 | 웹 업로드, 데이터 파싱 |
| LazyResource | 대용량 데이터 로딩 | DB 연결, 설정 파일 |
| LRUCache | API 응답 캐싱 | 외부 API 호출 결과 |
| 메모리 풀 | 고성능 서버 | JSON 파싱, 버퍼 관리 |
| 경계 검사 | 사용자 입력 처리 | 배열 접근, 데이터 검증 |

## 성능 지표

시뮬레이션 실행 시 다음 지표들을 확인할 수 있습니다:

- **페이지 폴트 비율**: 메모리 효율성 측정
- **메모리 사용률**: 현재 로드된 페이지 수
- **스왑 공간 사용량**: 디스크 스왑 활용도
- **캐시 히트율**: 캐시 효율성 (예제 코드에서)

## 학습 목표

1. 운영체제의 메모리 관리 메커니즘 이해
2. 실무에서 메모리 안전성 패턴 적용
3. 성능 최적화 기법 학습
4. Go 언어를 통한 시스템 프로그래밍 경험

## 참고 자료

- `contents.md`: 유효-무효 비트의 이론적 배경
- `examples/`: 실무 적용 예제 코드들
- Go 공식 문서: https://golang.org/doc/