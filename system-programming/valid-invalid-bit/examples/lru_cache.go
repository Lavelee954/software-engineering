package main

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// Item 캐시 항목 구조체
type Item struct {
	key     string
	value   interface{}
	element *list.Element
	valid   bool // 유효-무효 비트와 유사한 역할
}

// LRUCache LRU 캐시 구현 (페이지 교체 알고리즘 적용)
type LRUCache struct {
	capacity  int
	items     map[string]*Item
	evictList *list.List // 최근 사용된 순서 관리
	mu        sync.RWMutex
	hits      int64
	misses    int64
}

// NewLRUCache LRU 캐시 생성
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity:  capacity,
		items:     make(map[string]*Item),
		evictList: list.New(),
	}
}

// Get 데이터 가져오기 (페이지 접근과 유사)
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	item, exists := c.items[key]
	if !exists || !item.valid {
		c.misses++
		fmt.Printf("캐시 미스: %s (페이지 폴트와 유사)\n", key)
		return nil, false
	}
	
	// 최근 사용된 항목으로 이동 (참조 비트 업데이트와 유사)
	c.evictList.MoveToFront(item.element)
	c.hits++
	
	fmt.Printf("캐시 히트: %s\n", key)
	return item.value, true
}

// Set 데이터 설정 (페이지 로드와 유사)
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// 이미 존재하는 경우 업데이트
	if item, exists := c.items[key]; exists {
		item.value = value
		item.valid = true
		c.evictList.MoveToFront(item.element)
		fmt.Printf("캐시 업데이트: %s\n", key)
		return
	}
	
	// 용량 초과 시 가장 오래된 항목 제거 (페이지 교체와 유사)
	if c.evictList.Len() >= c.capacity {
		c.evictOldest()
	}
	
	// 새 항목 추가
	item := &Item{
		key:   key,
		value: value,
		valid: true,
	}
	
	element := c.evictList.PushFront(item)
	item.element = element
	c.items[key] = item
	
	fmt.Printf("캐시 추가: %s\n", key)
}

// evictOldest 가장 오래된 항목 제거 (LRU 페이지 교체)
func (c *LRUCache) evictOldest() {
	element := c.evictList.Back()
	if element != nil {
		c.evictList.Remove(element)
		item := element.Value.(*Item)
		delete(c.items, item.key)
		fmt.Printf("캐시 축출: %s (LRU 페이지 교체와 유사)\n", item.key)
	}
}

// Invalidate 항목 무효화
func (c *LRUCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if item, exists := c.items[key]; exists {
		item.valid = false
		fmt.Printf("캐시 무효화: %s\n", key)
	}
}

// Remove 항목 제거
func (c *LRUCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if item, exists := c.items[key]; exists {
		c.evictList.Remove(item.element)
		delete(c.items, key)
		fmt.Printf("캐시 제거: %s\n", key)
	}
}

// Stats 캐시 통계
func (c *LRUCache) Stats() (hits, misses int64, hitRate float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	total := c.hits + c.misses
	if total == 0 {
		return c.hits, c.misses, 0.0
	}
	
	return c.hits, c.misses, float64(c.hits)/float64(total)*100
}

// Size 현재 캐시 크기
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// APICache API 응답 캐시
type APICache struct {
	cache *LRUCache
	ttl   time.Duration
}

// NewAPICache API 캐시 생성
func NewAPICache(capacity int, ttl time.Duration) *APICache {
	return &APICache{
		cache: NewLRUCache(capacity),
		ttl:   ttl,
	}
}

// CachedAPIResponse 캐시된 API 응답
type CachedAPIResponse struct {
	Data      interface{}
	Timestamp time.Time
}

// GetUserData 사용자 데이터 가져오기 (캐시 적용)
func (api *APICache) GetUserData(userID string) (map[string]interface{}, error) {
	// 캐시에서 조회
	if cached, exists := api.cache.Get("user:" + userID); exists {
		response := cached.(*CachedAPIResponse)
		
		// TTL 확인
		if time.Since(response.Timestamp) < api.ttl {
			fmt.Printf("캐시에서 사용자 데이터 반환: %s\n", userID)
			return response.Data.(map[string]interface{}), nil
		} else {
			// 만료된 데이터 무효화
			api.cache.Invalidate("user:" + userID)
			fmt.Printf("캐시 데이터 만료: %s\n", userID)
		}
	}
	
	// API 호출 시뮬레이션
	fmt.Printf("외부 API 호출: %s\n", userID)
	time.Sleep(100 * time.Millisecond) // 네트워크 지연 시뮬레이션
	
	userData := map[string]interface{}{
		"id":    userID,
		"name":  "사용자" + userID,
		"email": "user" + userID + "@example.com",
	}
	
	// 캐시에 저장
	response := &CachedAPIResponse{
		Data:      userData,
		Timestamp: time.Now(),
	}
	api.cache.Set("user:"+userID, response)
	
	return userData, nil
}

// DatabaseCache 데이터베이스 쿼리 캐시
type DatabaseCache struct {
	cache *LRUCache
}

// NewDatabaseCache 데이터베이스 캐시 생성
func NewDatabaseCache(capacity int) *DatabaseCache {
	return &DatabaseCache{
		cache: NewLRUCache(capacity),
	}
}

// ExecuteQuery 쿼리 실행 (캐시 적용)
func (db *DatabaseCache) ExecuteQuery(query string) ([]map[string]interface{}, error) {
	// 캐시에서 조회
	if cached, exists := db.cache.Get(query); exists {
		fmt.Printf("캐시에서 쿼리 결과 반환: %s\n", query)
		return cached.([]map[string]interface{}), nil
	}
	
	// 데이터베이스 쿼리 실행 시뮬레이션
	fmt.Printf("데이터베이스 쿼리 실행: %s\n", query)
	time.Sleep(50 * time.Millisecond) // DB 쿼리 시간 시뮬레이션
	
	// 시뮬레이션 결과
	result := []map[string]interface{}{
		{"id": 1, "name": "김철수", "email": "kim@example.com"},
		{"id": 2, "name": "이영희", "email": "lee@example.com"},
	}
	
	// 캐시에 저장
	db.cache.Set(query, result)
	
	return result, nil
}

func main() {
	fmt.Println("=== LRU Cache 예제 (페이지 교체 알고리즘 적용) ===")
	
	// 예제 1: 기본 LRU 캐시 동작
	fmt.Println("\n1. 기본 LRU 캐시 동작:")
	cache := NewLRUCache(3) // 용량 3인 캐시
	
	// 데이터 추가
	cache.Set("A", "데이터A")
	cache.Set("B", "데이터B")
	cache.Set("C", "데이터C")
	
	// 캐시 조회
	cache.Get("A") // A를 최근 사용으로 이동
	cache.Get("B") // B를 최근 사용으로 이동
	
	// 용량 초과 시 가장 오래된 항목(C) 제거
	cache.Set("D", "데이터D")
	
	// 제거된 항목 조회 시도
	if _, exists := cache.Get("C"); !exists {
		fmt.Println("예상대로 C가 제거되었습니다")
	}
	
	// 예제 2: API 응답 캐시
	fmt.Println("\n2. API 응답 캐시:")
	apiCache := NewAPICache(10, 5*time.Second)
	
	// 첫 번째 요청 (API 호출)
	userData1, _ := apiCache.GetUserData("123")
	fmt.Printf("첫 번째 결과: %v\n", userData1)
	
	// 두 번째 요청 (캐시에서 반환)
	userData2, _ := apiCache.GetUserData("123")
	fmt.Printf("두 번째 결과: %v\n", userData2)
	
	// 다른 사용자 요청
	userData3, _ := apiCache.GetUserData("456")
	fmt.Printf("다른 사용자 결과: %v\n", userData3)
	
	// 예제 3: 데이터베이스 쿼리 캐시
	fmt.Println("\n3. 데이터베이스 쿼리 캐시:")
	dbCache := NewDatabaseCache(5)
	
	// 첫 번째 쿼리 실행
	result1, _ := dbCache.ExecuteQuery("SELECT * FROM users")
	fmt.Printf("첫 번째 쿼리 결과: %d개 행\n", len(result1))
	
	// 같은 쿼리 재실행 (캐시에서 반환)
	result2, _ := dbCache.ExecuteQuery("SELECT * FROM users")
	fmt.Printf("두 번째 쿼리 결과: %d개 행\n", len(result2))
	
	// 예제 4: 캐시 통계
	fmt.Println("\n4. 캐시 통계:")
	hits, misses, hitRate := cache.Stats()
	fmt.Printf("캐시 히트: %d, 미스: %d, 히트율: %.2f%%\n", hits, misses, hitRate)
	fmt.Printf("현재 캐시 크기: %d\n", cache.Size())
	
	// 예제 5: 캐시 무효화
	fmt.Println("\n5. 캐시 무효화:")
	cache.Invalidate("A")
	if _, exists := cache.Get("A"); !exists {
		fmt.Println("A가 무효화되어 조회할 수 없습니다")
	}
}