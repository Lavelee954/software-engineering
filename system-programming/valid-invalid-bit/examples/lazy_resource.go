package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// LazyResource 지연 로딩 리소스 (요구 페이징 개념 적용)
type LazyResource struct {
	loaded bool      // 유효-무효 비트와 유사한 역할
	data   []byte
	loader func() ([]byte, error)
	mu     sync.RWMutex
}

// NewLazyResource 지연 로딩 리소스 생성
func NewLazyResource(loader func() ([]byte, error)) *LazyResource {
	return &LazyResource{
		loaded: false, // 초기에는 로드되지 않음 (무효 상태)
		loader: loader,
	}
}

// Get 데이터 가져오기 (페이지 폴트 처리와 유사)
func (r *LazyResource) Get() ([]byte, error) {
	r.mu.RLock()
	if r.loaded {
		defer r.mu.RUnlock()
		return r.data, nil
	}
	r.mu.RUnlock()
	
	// 로드되지 않은 경우 지연 로딩 수행 (페이지 폴트 처리)
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 다시 한 번 확인 (double-checked locking)
	if r.loaded {
		return r.data, nil
	}
	
	fmt.Println("리소스 로딩 중... (페이지 폴트 처리와 유사)")
	data, err := r.loader()
	if err != nil {
		return nil, err
	}
	
	r.data = data
	r.loaded = true // 유효 상태로 변경
	fmt.Println("리소스 로딩 완료")
	
	return r.data, nil
}

// IsLoaded 로딩 상태 확인
func (r *LazyResource) IsLoaded() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.loaded
}

// Reset 리소스 재설정 (무효 상태로 변경)
func (r *LazyResource) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loaded = false
	r.data = nil
	fmt.Println("리소스가 재설정되었습니다 (무효 상태)")
}

// DatabaseConnection 데이터베이스 연결 관리
type DatabaseConnection struct {
	*LazyResource
	dsn string
	db  *sql.DB
}

// NewDatabaseConnection 데이터베이스 연결 생성
func NewDatabaseConnection(dsn string) *DatabaseConnection {
	conn := &DatabaseConnection{dsn: dsn}
	
	// 지연 로딩 함수 정의
	conn.LazyResource = NewLazyResource(func() ([]byte, error) {
		fmt.Printf("데이터베이스 연결 중: %s\n", dsn)
		time.Sleep(100 * time.Millisecond) // 연결 시뮬레이션
		
		// 실제로는 sql.Open을 사용하지만, 여기서는 시뮬레이션
		// db, err := sql.Open("mysql", dsn)
		// conn.db = db
		
		return []byte("connected"), nil
	})
	
	return conn
}

// Query 쿼리 실행
func (db *DatabaseConnection) Query(query string) ([][]string, error) {
	// 연결이 필요할 때만 로드
	_, err := db.Get()
	if err != nil {
		return nil, fmt.Errorf("데이터베이스 연결 오류: %v", err)
	}
	
	fmt.Printf("쿼리 실행: %s\n", query)
	
	// 시뮬레이션 결과 반환
	return [][]string{
		{"id", "name", "email"},
		{"1", "김철수", "kim@example.com"},
		{"2", "이영희", "lee@example.com"},
	}, nil
}

// ConfigManager 설정 파일 관리
type ConfigManager struct {
	*LazyResource
	configPath string
}

// NewConfigManager 설정 관리자 생성
func NewConfigManager(configPath string) *ConfigManager {
	manager := &ConfigManager{configPath: configPath}
	
	// 지연 로딩 함수 정의
	manager.LazyResource = NewLazyResource(func() ([]byte, error) {
		fmt.Printf("설정 파일 로딩 중: %s\n", configPath)
		time.Sleep(50 * time.Millisecond) // 파일 읽기 시뮬레이션
		
		// 실제로는 ioutil.ReadFile 사용
		// return ioutil.ReadFile(configPath)
		
		// 시뮬레이션 데이터
		return []byte(`{
			"database": {
				"host": "localhost",
				"port": 3306,
				"username": "user",
				"password": "pass"
			},
			"cache": {
				"enabled": true,
				"ttl": 300
			}
		}`), nil
	})
	
	return manager
}

// GetConfig 설정 가져오기
func (c *ConfigManager) GetConfig() (map[string]interface{}, error) {
	_, err := c.Get()
	if err != nil {
		return nil, err
	}
	
	// 실제로는 JSON 파싱
	config := map[string]interface{}{
		"database": map[string]interface{}{
			"host":     "localhost",
			"port":     3306,
			"username": "user",
			"password": "pass",
		},
		"cache": map[string]interface{}{
			"enabled": true,
			"ttl":     300,
		},
	}
	
	return config, nil
}

// APIClient API 클라이언트
type APIClient struct {
	*LazyResource
	baseURL string
	token   string
}

// NewAPIClient API 클라이언트 생성
func NewAPIClient(baseURL string) *APIClient {
	client := &APIClient{baseURL: baseURL}
	
	// 지연 로딩 함수 정의 (인증 토큰 획득)
	client.LazyResource = NewLazyResource(func() ([]byte, error) {
		fmt.Printf("API 인증 토큰 획득 중: %s\n", baseURL)
		time.Sleep(200 * time.Millisecond) // API 호출 시뮬레이션
		
		// 실제로는 HTTP 요청으로 토큰 획득
		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
		client.token = token
		
		return []byte(token), nil
	})
	
	return client
}

// GetUserData 사용자 데이터 가져오기
func (api *APIClient) GetUserData(userID string) (map[string]interface{}, error) {
	// 토큰이 필요할 때만 로드
	_, err := api.Get()
	if err != nil {
		return nil, fmt.Errorf("API 인증 오류: %v", err)
	}
	
	fmt.Printf("사용자 데이터 요청: %s (토큰: %s...)\n", userID, api.token[:10])
	
	// 시뮬레이션 결과 반환
	return map[string]interface{}{
		"id":    userID,
		"name":  "사용자" + userID,
		"email": "user" + userID + "@example.com",
	}, nil
}

func main() {
	fmt.Println("=== LazyResource 예제 (요구 페이징 개념 적용) ===")
	
	// 예제 1: 데이터베이스 연결 관리
	fmt.Println("\n1. 데이터베이스 연결 관리:")
	db := NewDatabaseConnection("mysql://user:pass@localhost:3306/mydb")
	
	fmt.Printf("연결 상태: %t\n", db.IsLoaded())
	
	// 첫 번째 쿼리 실행 (이때 연결 로드)
	result, err := db.Query("SELECT * FROM users")
	if err != nil {
		fmt.Printf("쿼리 오류: %v\n", err)
	} else {
		fmt.Printf("쿼리 결과: %v\n", result)
	}
	
	fmt.Printf("연결 상태: %t\n", db.IsLoaded())
	
	// 두 번째 쿼리 실행 (이미 로드된 연결 사용)
	result2, _ := db.Query("SELECT COUNT(*) FROM users")
	fmt.Printf("두 번째 쿼리 결과: %v\n", result2)
	
	// 예제 2: 설정 파일 관리
	fmt.Println("\n2. 설정 파일 관리:")
	configManager := NewConfigManager("/etc/app/config.json")
	
	fmt.Printf("설정 로드 상태: %t\n", configManager.IsLoaded())
	
	// 설정 가져오기 (이때 파일 로드)
	config, err := configManager.GetConfig()
	if err != nil {
		fmt.Printf("설정 로드 오류: %v\n", err)
	} else {
		fmt.Printf("데이터베이스 호스트: %v\n", config["database"].(map[string]interface{})["host"])
	}
	
	// 예제 3: API 클라이언트
	fmt.Println("\n3. API 클라이언트:")
	apiClient := NewAPIClient("https://api.example.com")
	
	fmt.Printf("토큰 로드 상태: %t\n", apiClient.IsLoaded())
	
	// 사용자 데이터 요청 (이때 토큰 획득)
	userData, err := apiClient.GetUserData("123")
	if err != nil {
		fmt.Printf("사용자 데이터 오류: %v\n", err)
	} else {
		fmt.Printf("사용자 정보: %v\n", userData)
	}
	
	// 예제 4: 리소스 재설정
	fmt.Println("\n4. 리소스 재설정:")
	configManager.Reset()
	fmt.Printf("재설정 후 로드 상태: %t\n", configManager.IsLoaded())
}