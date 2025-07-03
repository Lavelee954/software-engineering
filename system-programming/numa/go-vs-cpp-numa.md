# Go vs C/C++: NUMA 최적화 프로그래밍 비교

## 개요

NUMA(Non-Uniform Memory Access) 시스템에서의 메모리 관리 최적화를 Go와 C/C++로 구현할 때의 차이점과 한계를 분석합니다.

## 주요 차이점

### 1. 메모리 관리 제어 수준

#### C/C++
```c
// 직접적인 NUMA 노드별 메모리 할당
void* ptr = numa_alloc_onnode(size, node_id);
numa_free(ptr, size);

// CPU 친화도 직접 설정
cpu_set_t mask;
CPU_ZERO(&mask);
CPU_SET(cpu_id, &mask);
sched_setaffinity(pid, sizeof(mask), &mask);
```

#### Go
```go
// 메모리 할당 위치 제어 불가능
slice := make([]int, 1000000)  // 어떤 NUMA 노드에 할당될지 모름

// 간접적인 OS 스레드 고정만 가능
runtime.LockOSThread()
```

### 2. 시스템 콜 접근성

#### C/C++
- `numa_alloc_onnode()`: 특정 NUMA 노드에 메모리 할당
- `sched_setaffinity()`: CPU 친화도 설정
- `mbind()`: 메모리 정책 바인딩
- 직접적인 libnuma API 사용

#### Go
```go
/*
#cgo LDFLAGS: -lnuma
#include <numa.h>
*/
import "C"

// cgo를 통한 간접 접근만 가능
// syscall 패키지를 통한 제한적 접근
```

### 3. 성능 최적화 가능성

#### C/C++
- 메모리 레이아웃 완전 제어
- CPU-메모리 친화도 최적화
- 캐시 적중률 최대화
- 메모리 접근 패턴 세밀 조정

#### Go
- 런타임 스케줄러가 고루틴 관리
- NUMA 인식 제한적
- 가비지 컬렉터의 예측 불가능한 메모리 재배치

## Go에서 NUMA 최적화가 어려운 이유

### 1. 가비지 컬렉터의 메모리 재배치
```go
// 메모리가 특정 NUMA 노드에 할당되어도
obj := make([]byte, 1024)
// GC가 언제든지 다른 노드로 이동시킬 수 있음
// 개발자가 제어할 수 없음
```

### 2. 고루틴 스케줄러의 NUMA 무시
```go
// 고루틴이 어떤 OS 스레드/CPU에서 실행될지 불확실
go func() {
    // 이 코드가 어떤 NUMA 노드에서 실행될지 예측 불가
    data := heavyComputation()
}()
```

### 3. 메모리 할당자의 추상화
- Go의 메모리 할당은 NUMA 노드를 지정할 수 없음
- 런타임이 내부적으로 메모리 관리
- 개발자의 세밀한 제어 불가능

### 4. 런타임의 투명한 관리
- Go 런타임이 성능을 위해 메모리/스레드를 자동 관리
- 개발자가 저수준 제어 불가능
- 추상화 계층이 NUMA 최적화를 방해

### 5. 추상화 계층의 격차
```
응용프로그램 → Go 런타임 → OS → 하드웨어 (NUMA)
     ↑              ↑
  제어 불가      투명한 관리
```

## Go에서의 제한적 NUMA 접근 방법

### 1. cgo를 통한 libnuma 사용
```go
/*
#cgo LDFLAGS: -lnuma
#include <numa.h>
*/
import "C"

func allocOnNode(size int, node int) unsafe.Pointer {
    return C.numa_alloc_onnode(C.size_t(size), C.int(node))
}
```

### 2. OS 스레드 고정
```go
func workerOnCPU() {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    
    // 이 고루틴은 특정 OS 스레드에서만 실행
    // 하지만 여전히 NUMA 노드 제어는 제한적
}
```

### 3. syscall을 통한 CPU 친화도 설정
```go
import "syscall"

func setCPUAffinity(pid int, cpus []int) error {
    // 복잡한 syscall 조작 필요
    // 플랫폼 의존적
    return syscall.Syscall(syscall.SYS_SCHED_SETAFFINITY, ...)
}
```

## 설계 철학의 차이

### Go의 철학
- **개발자 편의성 우선**: 런타임이 모든 것을 관리
- **안전성**: 메모리 안전성과 동시성 안전성
- **단순성**: 복잡한 저수준 제어 숨김

### NUMA 최적화의 요구사항
- **하드웨어 특성에 맞춘 세밀한 제어**
- **메모리 할당 위치의 정확한 제어**
- **CPU-메모리 친화도 최적화**

## 적용 분야별 권장사항

### Go 적합 분야
- 웹 서버
- 마이크로서비스
- 일반 애플리케이션
- 네트워크 프로그래밍

### C/C++ 필요 분야
- 시스템 프로그래밍
- 고성능 컴퓨팅 (HPC)
- OS 커널 개발
- 임베디드 시스템
- **NUMA 최적화가 중요한 시스템**

## 결론

Go는 NUMA 최적화가 **구조적으로 부적합**합니다:

1. **언어 설계 자체가 저수준 최적화를 막음**
2. **런타임의 추상화가 NUMA 제어를 방해**
3. **설계 철학의 차이**: 편의성 vs 성능 최적화

NUMA 성능이 중요한 시스템 프로그래밍에서는 **C/C++를 사용하는 것이 필수**입니다.

---

*이 문서는 `contents.md`의 NUMA 메모리 관리 내용을 바탕으로 작성되었습니다.*