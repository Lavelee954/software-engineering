# OpenStack Keystone과 Kubernetes IAM 통합 완전 가이드

## 목차
1. [개요](#개요)
2. [핵심 통합 아키텍처](#핵심-통합-아키텍처)
3. [k8s-keystone-auth 서버 배포 전략](#k8s-keystone-auth-서버-배포-전략)
4. [CSP 관리형 서비스 설계](#csp-관리형-서비스-설계)
5. [테넌트 구분 방법](#테넌트-구분-방법)
6. [실제 구현 예시](#실제-구현-예시)
7. [운영 가이드](#운영-가이드)

## 개요

OpenStack Keystone과 Kubernetes IAM 통합은 클라우드 네이티브 환경에서 중앙 집중식 인증 및 권한 관리를 위한 핵심 기술입니다. 이 가이드는 개발자와 DevOps 엔지니어가 실제 프로덕션 환경에서 구현할 수 있는 포괄적인 솔루션을 제공합니다.

### 주요 이점
- **중앙 집중식 인증**: OpenStack Keystone을 통한 단일 인증 소스
- **역할 기반 접근 제어**: 세밀한 권한 관리
- **멀티테넌트 지원**: 테넌트별 격리 및 리소스 관리
- **확장성**: 대규모 클러스터 환경 지원
- **보안**: 엔터프라이즈급 보안 요구사항 충족

## 핵심 통합 아키텍처

### 1. Keystone 인증 통합

```yaml
# k8s-keystone-auth 서버 설정 (keystone-auth.yaml)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-keystone-auth
  namespace: kube-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: k8s-keystone-auth
  template:
    metadata:
      labels:
        app: k8s-keystone-auth
    spec:
      containers:
      - name: k8s-keystone-auth
        image: k8scloudprovider/k8s-keystone-auth:v1.28.0
        ports:
        - containerPort: 8443
        env:
        - name: KEYSTONE_URL
          value: "https://keystone.example.com:5000/v3"
        - name: KEYSTONE_CA_FILE
          value: "/etc/ssl/certs/keystone-ca.pem"
        volumeMounts:
        - name: webhook-config
          mountPath: /etc/webhook
        - name: keystone-ca
          mountPath: /etc/ssl/certs
        command:
        - /bin/k8s-keystone-auth
        - --tls-cert-file=/etc/webhook/tls.crt
        - --tls-private-key-file=/etc/webhook/tls.key
        - --keystone-url=$(KEYSTONE_URL)
        - --keystone-ca-file=$(KEYSTONE_CA_FILE)
        - --listen=0.0.0.0:8443
        - --v=2
```

### 2. API 서버 웹훅 설정

```yaml
# /etc/kubernetes/manifests/kube-apiserver.yaml
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
  - name: kube-apiserver
    command:
    - kube-apiserver
    - --authentication-token-webhook-config-file=/etc/kubernetes/webhooks/keystone-auth.yaml
    - --authorization-webhook-config-file=/etc/kubernetes/webhooks/keystone-authz.yaml
    - --authorization-mode=Node,RBAC,Webhook
    volumeMounts:
    - name: webhook-config
      mountPath: /etc/kubernetes/webhooks
      readOnly: true
```

### 3. 클라이언트 인증 구성

```yaml
# kubectl 설정 예시
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://k8s-api.example.com:6443
    certificate-authority-data: <CA_DATA>
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: keystone-user
  name: keystone-context
current-context: keystone-context
users:
- name: keystone-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: client-keystone-auth
      args:
      - --keystone-url=https://keystone.example.com:5000/v3
      - --domain-name=default
      - --project-name=myproject
```

## k8s-keystone-auth 서버 배포 전략

### 1. 중앙 집중식 배포

**장점:**
- 단일 정책 관리 지점
- 일관성 있는 인증 정책
- 운영 복잡성 최소화

**단점:**
- 단일 실패 지점 (SPOF)
- 네트워크 지연 가능성
- 확장성 제한

```yaml
# 중앙 집중식 고가용성 설정
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-keystone-auth-central
spec:
  replicas: 5
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 2
      maxUnavailable: 1
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: k8s-keystone-auth
              topologyKey: kubernetes.io/hostname
```

### 2. 클러스터별 배포

**장점:**
- 높은 가용성
- 낮은 네트워크 지연
- 클러스터별 독립성

**단점:**
- 관리 복잡성 증가
- 정책 일관성 유지 어려움
- 리소스 사용량 증가

### 3. 지역별/환경별 배포 (권장)

```yaml
# 지역별 배포 설정
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-keystone-auth-region-a
  labels:
    region: region-a
spec:
  replicas: 3
  selector:
    matchLabels:
      app: k8s-keystone-auth
      region: region-a
---
apiVersion: v1
kind: Service
metadata:
  name: k8s-keystone-auth-region-a
spec:
  selector:
    app: k8s-keystone-auth
    region: region-a
  ports:
  - port: 8443
    targetPort: 8443
```

### 4. 배포 규모별 권장사항

- **소규모 (< 10개 클러스터)**: 중앙 집중식
- **중규모 (10-50개 클러스터)**: 지역별 배포
- **대규모 (> 50개 클러스터)**: 하이브리드 접근

## CSP 관리형 서비스 설계

### 1. 멀티테넌트 서비스 모델

```go
// pkg/tenant/manager.go
package tenant

import (
    "context"
    "fmt"
    "k8s.io/client-go/kubernetes"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

type TenantManager struct {
    client.Client
    kubeClient kubernetes.Interface
}

type TenantSpec struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Domain      string            `json:"domain"`
    Projects    []string          `json:"projects"`
    SLATier     string            `json:"slaTier"`
    Resources   ResourceQuota     `json:"resources"`
    Metadata    map[string]string `json:"metadata"`
}

type ResourceQuota struct {
    CPU     string `json:"cpu"`
    Memory  string `json:"memory"`
    Storage string `json:"storage"`
    Pods    int    `json:"pods"`
}

func (tm *TenantManager) CreateTenant(ctx context.Context, spec TenantSpec) error {
    // 1. Keystone 프로젝트 생성
    if err := tm.createKeystoneProject(ctx, spec); err != nil {
        return fmt.Errorf("failed to create keystone project: %w", err)
    }
    
    // 2. Kubernetes 네임스페이스 생성
    if err := tm.createNamespace(ctx, spec); err != nil {
        return fmt.Errorf("failed to create namespace: %w", err)
    }
    
    // 3. 리소스 쿼터 설정
    if err := tm.applyResourceQuota(ctx, spec); err != nil {
        return fmt.Errorf("failed to apply resource quota: %w", err)
    }
    
    // 4. 네트워크 정책 적용
    if err := tm.applyNetworkPolicy(ctx, spec); err != nil {
        return fmt.Errorf("failed to apply network policy: %w", err)
    }
    
    return nil
}
```

### 2. SLA 계층별 서비스 구성

```yaml
# Premium SLA - 전용 인스턴스
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-keystone-auth-premium
  labels:
    sla-tier: premium
spec:
  replicas: 5
  template:
    spec:
      nodeSelector:
        node-type: premium
      resources:
        requests:
          cpu: 1000m
          memory: 2Gi
        limits:
          cpu: 2000m
          memory: 4Gi
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                sla-tier: premium
            topologyKey: kubernetes.io/hostname
---
# Standard SLA - 공유 인스턴스
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-keystone-auth-standard
  labels:
    sla-tier: standard
spec:
  replicas: 3
  template:
    spec:
      resources:
        requests:
          cpu: 500m
          memory: 1Gi
        limits:
          cpu: 1000m
          memory: 2Gi
---
# Basic SLA - 공유 인스턴스
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-keystone-auth-basic
  labels:
    sla-tier: basic
spec:
  replicas: 2
  template:
    spec:
      resources:
        requests:
          cpu: 250m
          memory: 512Mi
        limits:
          cpu: 500m
          memory: 1Gi
```

### 3. 자동 스케일링 설정

```yaml
# HPA 설정
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: k8s-keystone-auth-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: k8s-keystone-auth-standard
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: webhook_requests_per_second
      target:
        type: AverageValue
        averageValue: "100"
```

## 테넌트 구분 방법

### 1. 통합 미들웨어 구현

```go
// pkg/middleware/tenant.go
package middleware

import (
    "context"
    "crypto/x509"
    "fmt"
    "net/http"
    "strings"
    "github.com/dgrijalva/jwt-go"
)

type TenantIdentifier struct {
    methods []TenantMethod
}

type TenantMethod interface {
    IdentifyTenant(r *http.Request) (string, error)
}

type TenantContext struct {
    ID       string
    SLATier  string
    Projects []string
    Metadata map[string]string
}

// 1. URL 경로 기반 테넌트 식별
type PathBasedIdentifier struct{}

func (p *PathBasedIdentifier) IdentifyTenant(r *http.Request) (string, error) {
    path := r.URL.Path
    parts := strings.Split(path, "/")
    
    if len(parts) >= 3 && parts[1] == "tenant" {
        return parts[2], nil
    }
    
    return "", fmt.Errorf("tenant not found in path")
}

// 2. 서브도메인 기반 테넌트 식별
type SubdomainBasedIdentifier struct{}

func (s *SubdomainBasedIdentifier) IdentifyTenant(r *http.Request) (string, error) {
    host := r.Host
    if strings.Contains(host, ".") {
        subdomain := strings.Split(host, ".")[0]
        if strings.HasPrefix(subdomain, "tenant-") {
            return strings.TrimPrefix(subdomain, "tenant-"), nil
        }
    }
    return "", fmt.Errorf("tenant not found in subdomain")
}

// 3. 클라이언트 인증서 기반 테넌트 식별
type CertBasedIdentifier struct{}

func (c *CertBasedIdentifier) IdentifyTenant(r *http.Request) (string, error) {
    if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
        return "", fmt.Errorf("no client certificate provided")
    }
    
    cert := r.TLS.PeerCertificates[0]
    
    // CN에서 테넌트 추출
    if strings.HasPrefix(cert.Subject.CommonName, "tenant-") {
        return strings.TrimPrefix(cert.Subject.CommonName, "tenant-"), nil
    }
    
    // SAN에서 테넌트 추출
    for _, san := range cert.DNSNames {
        if strings.HasPrefix(san, "tenant-") {
            return strings.TrimPrefix(san, "tenant-"), nil
        }
    }
    
    return "", fmt.Errorf("tenant not found in certificate")
}

// 4. JWT 토큰 기반 테넌트 식별
type JWTBasedIdentifier struct {
    secretKey []byte
}

func (j *JWTBasedIdentifier) IdentifyTenant(r *http.Request) (string, error) {
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        return "", fmt.Errorf("invalid authorization header")
    }
    
    tokenString := strings.TrimPrefix(authHeader, "Bearer ")
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return j.secretKey, nil
    })
    
    if err != nil || !token.Valid {
        return "", fmt.Errorf("invalid token")
    }
    
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return "", fmt.Errorf("invalid token claims")
    }
    
    if tenantID, exists := claims["tenant_id"].(string); exists {
        return tenantID, nil
    }
    
    return "", fmt.Errorf("tenant_id not found in token")
}

// 5. API 키 기반 테넌트 식별
type APIKeyBasedIdentifier struct {
    keyStore map[string]string // API Key -> Tenant ID
}

func (a *APIKeyBasedIdentifier) IdentifyTenant(r *http.Request) (string, error) {
    apiKey := r.Header.Get("X-API-Key")
    if apiKey == "" {
        return "", fmt.Errorf("API key not provided")
    }
    
    if tenantID, exists := a.keyStore[apiKey]; exists {
        return tenantID, nil
    }
    
    return "", fmt.Errorf("invalid API key")
}

// 통합 테넌트 식별 미들웨어
func (ti *TenantIdentifier) IdentifyTenant(r *http.Request) (string, error) {
    for _, method := range ti.methods {
        if tenantID, err := method.IdentifyTenant(r); err == nil {
            return tenantID, nil
        }
    }
    return "", fmt.Errorf("tenant identification failed")
}

// HTTP 미들웨어 핸들러
func (ti *TenantIdentifier) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID, err := ti.IdentifyTenant(r)
        if err != nil {
            http.Error(w, "Tenant identification failed", http.StatusUnauthorized)
            return
        }
        
        // 테넌트 컨텍스트 추가
        ctx := context.WithValue(r.Context(), "tenant_id", tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 2. Istio 기반 테넌트 라우팅

```yaml
# Istio VirtualService 설정
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: keystone-auth-routing
spec:
  hosts:
  - keystone-auth.example.com
  - "*.keystone-auth.example.com"
  http:
  # 프리미엄 테넌트 라우팅
  - match:
    - headers:
        x-tenant-sla:
          exact: premium
    route:
    - destination:
        host: k8s-keystone-auth-premium
        port:
          number: 8443
  # 서브도메인 기반 라우팅
  - match:
    - headers:
        host:
          regex: "tenant-([^.]+)\\.keystone-auth\\.example\\.com"
    route:
    - destination:
        host: k8s-keystone-auth-standard
        port:
          number: 8443
  # 기본 라우팅
  - route:
    - destination:
        host: k8s-keystone-auth-basic
        port:
          number: 8443
---
# 테넌트별 Rate Limiting
apiVersion: networking.istio.io/v1beta1
kind: EnvoyFilter
metadata:
  name: tenant-rate-limit
spec:
  configPatches:
  - applyTo: HTTP_FILTER
    match:
      context: SIDECAR_INBOUND
    patch:
      operation: INSERT_BEFORE
      value:
        name: envoy.filters.http.local_ratelimit
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
          stat_prefix: tenant_rate_limiter
          token_bucket:
            max_tokens: 1000
            tokens_per_fill: 100
            fill_interval: 60s
          filter_enabled:
            runtime_key: tenant_rate_limit_enabled
            default_value:
              numerator: 100
              denominator: HUNDRED
```

## 실제 구현 예시

### 1. 완전한 k8s-keystone-auth 서버 구성

```yaml
# keystone-auth-complete.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: keystone-auth-config
  namespace: kube-system
data:
  policy.json: |
    {
      "version": "3.0",
      "rules": [
        {
          "roles": ["admin", "k8s-admin"],
          "namespaces": ["*"],
          "resources": ["*"],
          "verbs": ["*"]
        },
        {
          "roles": ["developer"],
          "namespaces": ["development", "testing"],
          "resources": ["pods", "services", "deployments"],
          "verbs": ["get", "list", "create", "update", "delete"]
        },
        {
          "roles": ["viewer"],
          "namespaces": ["*"],
          "resources": ["*"],
          "verbs": ["get", "list", "watch"]
        }
      ]
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-keystone-auth
  namespace: kube-system
  labels:
    app: k8s-keystone-auth
spec:
  replicas: 3
  selector:
    matchLabels:
      app: k8s-keystone-auth
  template:
    metadata:
      labels:
        app: k8s-keystone-auth
    spec:
      serviceAccountName: k8s-keystone-auth
      containers:
      - name: k8s-keystone-auth
        image: k8scloudprovider/k8s-keystone-auth:v1.28.0
        ports:
        - containerPort: 8443
          name: webhook
        - containerPort: 8080
          name: metrics
        env:
        - name: KEYSTONE_URL
          value: "https://keystone.example.com:5000/v3"
        - name: KEYSTONE_CA_FILE
          value: "/etc/ssl/certs/keystone-ca.pem"
        - name: POLICY_FILE
          value: "/etc/config/policy.json"
        volumeMounts:
        - name: webhook-certs
          mountPath: /etc/webhook/certs
          readOnly: true
        - name: keystone-ca
          mountPath: /etc/ssl/certs
          readOnly: true
        - name: policy-config
          mountPath: /etc/config
          readOnly: true
        command:
        - /bin/k8s-keystone-auth
        - --tls-cert-file=/etc/webhook/certs/tls.crt
        - --tls-private-key-file=/etc/webhook/certs/tls.key
        - --keystone-url=$(KEYSTONE_URL)
        - --keystone-ca-file=$(KEYSTONE_CA_FILE)
        - --policy-file=$(POLICY_FILE)
        - --listen=0.0.0.0:8443
        - --metrics-bind-address=0.0.0.0:8080
        - --v=2
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
      volumes:
      - name: webhook-certs
        secret:
          secretName: k8s-keystone-auth-certs
      - name: keystone-ca
        secret:
          secretName: keystone-ca-cert
      - name: policy-config
        configMap:
          name: keystone-auth-config
---
apiVersion: v1
kind: Service
metadata:
  name: k8s-keystone-auth
  namespace: kube-system
spec:
  selector:
    app: k8s-keystone-auth
  ports:
  - name: webhook
    port: 8443
    targetPort: 8443
  - name: metrics
    port: 8080
    targetPort: 8080
```

### 2. 모니터링 및 알림 설정

```yaml
# monitoring-setup.yaml
apiVersion: v1
kind: ServiceMonitor
metadata:
  name: k8s-keystone-auth
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: k8s-keystone-auth
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
---
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: k8s-keystone-auth-alerts
  namespace: kube-system
spec:
  groups:
  - name: keystone-auth
    rules:
    - alert: KeystoneAuthDown
      expr: up{job="k8s-keystone-auth"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Keystone Auth service is down"
        description: "Keystone Auth service has been down for more than 5 minutes"
    
    - alert: KeystoneAuthHighLatency
      expr: histogram_quantile(0.95, rate(keystone_auth_request_duration_seconds_bucket[5m])) > 0.5
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High latency in Keystone Auth requests"
        description: "95th percentile latency is above 500ms"
    
    - alert: KeystoneAuthHighErrorRate
      expr: rate(keystone_auth_request_total{code!="200"}[5m]) / rate(keystone_auth_request_total[5m]) > 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High error rate in Keystone Auth"
        description: "Error rate is above 10%"
```

## 운영 가이드

### 1. 보안 강화 방안

```yaml
# security-hardening.yaml
apiVersion: v1
kind: Pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    fsGroup: 65534
  containers:
  - name: k8s-keystone-auth
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
    volumeMounts:
    - name: tmp
      mountPath: /tmp
    - name: var-cache
      mountPath: /var/cache
  volumes:
  - name: tmp
    emptyDir: {}
  - name: var-cache
    emptyDir: {}
```

### 2. 백업 및 복구 전략

```bash
#!/bin/bash
# backup-keystone-auth.sh

# 설정 백업
kubectl get configmap keystone-auth-config -n kube-system -o yaml > keystone-auth-config-backup.yaml

# 시크릿 백업
kubectl get secret k8s-keystone-auth-certs -n kube-system -o yaml > keystone-auth-certs-backup.yaml

# 정책 백업
kubectl get configmap keystone-auth-policy -n kube-system -o yaml > keystone-auth-policy-backup.yaml

# 백업을 안전한 위치에 저장
aws s3 cp keystone-auth-config-backup.yaml s3://backup-bucket/keystone-auth/$(date +%Y%m%d)/
aws s3 cp keystone-auth-certs-backup.yaml s3://backup-bucket/keystone-auth/$(date +%Y%m%d)/
aws s3 cp keystone-auth-policy-backup.yaml s3://backup-bucket/keystone-auth/$(date +%Y%m%d)/
```

### 3. 트러블슈팅 가이드

```bash
# 일반적인 문제 진단 스크립트
#!/bin/bash
# troubleshoot-keystone-auth.sh

echo "=== Keystone Auth 서비스 상태 확인 ==="
kubectl get pods -n kube-system -l app=k8s-keystone-auth

echo "=== 서비스 로그 확인 ==="
kubectl logs -n kube-system -l app=k8s-keystone-auth --tail=100

echo "=== 인증서 상태 확인 ==="
kubectl get secret k8s-keystone-auth-certs -n kube-system -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout

echo "=== 연결성 테스트 ==="
kubectl run test-pod --image=curlimages/curl --rm -i --tty -- curl -k https://k8s-keystone-auth.kube-system.svc.cluster.local:8443/healthz

echo "=== 메트릭 확인 ==="
kubectl port-forward -n kube-system svc/k8s-keystone-auth 8080:8080 &
sleep 2
curl http://localhost:8080/metrics | grep keystone_auth
pkill -f "kubectl port-forward"
```

---

이 가이드는 OpenStack Keystone과 Kubernetes IAM 통합을 위한 완전한 솔루션을 제공합니다. 각 섹션의 코드와 설정 예시를 참고하여 실제 환경에 맞게 조정하여 사용하시기 바랍니다. 