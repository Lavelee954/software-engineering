# Central Router A2A Communication Architecture

## 🚀 Overview

현재 프로젝트에 **중앙 라우터(Central Router)** 기반의 **Agent-to-Agent (A2A) 통신 시스템**을 구현했습니다. 이 시스템은 각 디렉토리의 agent들이 독립적으로 실행되면서도 지능적으로 서로 통신할 수 있도록 설계되었습니다.

## 🏗️ 아키텍처 구조

### 기존 vs 새로운 구조

**기존 구조 (Direct NATS Pub/Sub):**
```
Agent A → NATS Topic → Agent B
```

**새로운 구조 (Central Router A2A):**
```
Agent A → Central Router → Agent B
         ↕
    Service Discovery
    Load Balancing
    Health Monitoring
    Circuit Breakers
```

### 핵심 컴포넌트

1. **Central Router** (`shared/central_router.py`)
   - 에이전트 서비스 디스커버리
   - 지능적 메시지 라우팅
   - 로드 밸런싱 (Round Robin, Least Loaded, Random)
   - 헬스 모니터링 및 Circuit Breaker 패턴
   - 실시간 통계 및 모니터링

2. **Enhanced Base Agent** (`shared/base_agent.py`)
   - 자동 라우터 등록/해제
   - Heartbeat 관리
   - A2A 메시지 처리
   - 라우팅 기능 내장

3. **Agent-Specific Implementations**
   - Technical Analysis Agent with A2A capabilities
   - Router 통합 메시지 핸들러
   - Cross-agent validation 기능

## 🔧 주요 기능

### 1. 서비스 디스커버리
- 에이전트 자동 등록/발견
- 능력(Capability) 기반 라우팅
- 동적 에이전트 추가/제거

### 2. 지능적 라우팅
- **Round Robin**: 균등 분배
- **Least Loaded**: 부하 기반 라우팅
- **Random**: 랜덤 분배
- **Capability-based**: 기능 기반 라우팅

### 3. fault Tolerance
- Circuit Breaker 패턴
- 자동 failover
- 헬스 체크 및 복구

### 4. A2A 통신 패턴
- Request/Response
- Broadcast
- Validation Requests
- Configuration Updates

## 🚀 사용 방법

### 1. 시스템 시작

```bash
# 1. Central Router 시작
cd python
make start-router

# 2. 모든 에이전트 시작
make start-all-agents

# 3. 상태 확인
make status
```

### 2. 개별 에이전트 시작

```bash
# Technical Analysis Agent만 시작
make start-technical

# News Analysis Agent만 시작
make start-news

# Sentiment Analysis Agent만 시작
make start-sentiment
```

### 3. 개발 환경 설정

```bash
make dev-setup
```

## 📊 A2A 통신 예제

### Technical Analysis Agent → Strategy Agent

```python
# Technical Analysis Agent에서 강한 신호 감지시
await self._notify_strategy_agent("strong_bullish_trend", indicators)

# 내부적으로 다음과 같이 라우팅됨:
message = {
    "signal_type": "strong_bullish_trend",
    "symbol": "AAPL",
    "confidence": 0.85,
    "indicators": {...}
}

await self.route_message_to_agent(
    message=message,
    destination_type="strategy",  # Strategy Agent 타입으로 라우팅
    strategy="round_robin"        # 라운드 로빈 방식
)
```

### Cross-Agent Validation

```python
# Strategy Agent가 Technical Analysis Agent에게 검증 요청
validation_request = {
    "message_type": "validation_request",
    "symbol": "AAPL",
    "signal": {"trend": "bullish", "strength": 0.8},
    "correlation_id": "req_123"
}

await self.route_message_to_agent(
    message=validation_request,
    destination_type="technical_analysis"
)

# Technical Analysis Agent가 검증 결과 응답
validation_response = {
    "message_type": "validation_response",
    "validation_result": {
        "trend_agreement": True,
        "confidence_adjustment": 0.1,
        "technical_support": True
    }
}
```

### Broadcast Alert

```python
# 높은 거래량 감지시 모든 관련 에이전트에게 브로드캐스트
await self.broadcast_message(
    message={
        "alert_type": "high_volume",
        "symbol": "AAPL",
        "volume_ratio": 3.5
    },
    agent_types=["strategy", "risk_management", "portfolio_management"]
)
```

## 🔍 모니터링 및 디버깅

### 라우터 통계 확인

```python
# 라우터 상태 조회
router_info = router.get_agent_info()
print(f"등록된 에이전트: {router_info['total_agents']}")
print(f"라우팅된 메시지: {router_info['stats']['messages_routed']}")
```

### 에이전트 상태 확인

```bash
make status
```

출력 예시:
```
📊 Agent Status:
Central Router:
  ✅ Running (PID: 12345)
Technical Analysis Agent:
  ✅ Running (PID: 12346)
News Analysis Agent:
  ✅ Running (PID: 12347)
Sentiment Analysis Agent:
  ✅ Running (PID: 12348)
```

## 🎯 장점 및 개선사항

### ✅ 장점

1. **모듈성**: 각 에이전트가 독립적으로 개발/배포 가능
2. **확장성**: 새로운 에이전트 타입 쉽게 추가
3. **안정성**: Circuit Breaker로 장애 격리
4. **지능성**: 능력 기반 자동 라우팅
5. **가시성**: 실시간 모니터링 및 통계

### 🚀 향후 개선사항

1. **웹 대시보드**: 라우터 상태 시각화
2. **메트릭스 연동**: Prometheus/Grafana 통합
3. **API Gateway**: REST API를 통한 에이전트 제어
4. **자동 스케일링**: 부하에 따른 에이전트 인스턴스 조절
5. **메시지 큐잉**: 고부하시 메시지 버퍼링

## 🔧 설정 옵션

### Central Router 설정

```python
router = CentralRouter(
    nats_url="nats://localhost:4222"
)

# 라우팅 전략 설정
routing_rule = RoutingRule(
    source_pattern="technical_analysis",
    destination_pattern="strategy",
    strategy=RoutingStrategy.LEAST_LOADED,
    timeout=30,
    retry_count=3
)
```

### Agent 설정

```python
config = TechnicalConfig(
    agent_name="technical-analysis-1",
    capabilities=["technical_analysis", "indicator_calculation", "signal_generation"],
    nats_url="nats://localhost:4222"
)
```

## 🐛 트러블슈팅

### 일반적인 문제들

1. **에이전트가 등록되지 않는 경우**
   - NATS 서버 연결 확인
   - Central Router가 실행 중인지 확인

2. **메시지 라우팅 실패**
   - 목적지 에이전트 타입 확인
   - Circuit Breaker 상태 확인

3. **높은 지연시간**
   - 라우팅 전략을 `least_loaded`로 변경
   - 에이전트 인스턴스 추가

### 로그 확인

```bash
# Central Router 로그 (DEBUG 모드)
python run_central_router.py --log-level DEBUG

# 개별 에이전트 로그
export LOG_LEVEL=DEBUG
make start-technical
```

## 📚 코드 예제

전체 사용 예제는 다음 파일들을 참조하세요:

- `shared/central_router.py` - 중앙 라우터 구현
- `shared/base_agent.py` - 강화된 베이스 에이전트
- `agents/technical_analysis/agent.py` - A2A 통신 적용 예제
- `run_central_router.py` - 라우터 실행 스크립트
- `Makefile` - 시스템 관리 명령들

이 아키텍처는 확장 가능하고 안정적인 멀티 에이전트 시스템을 제공하며, 각 에이전트가 독립적으로 동작하면서도 효과적으로 협력할 수 있도록 합니다. 