package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
	
	_ "github.com/mattn/go-sqlite3"
)

// BuddyConnectionPool은 버디 시스템 개념을 활용한 커넥션 풀
type BuddyConnectionPool struct {
	pools    map[int]*sync.Pool // 2의 거듭제곱 크기별 풀
	dbSource string
	mutex    sync.RWMutex
	stats    *PoolStats
}

type PoolStats struct {
	ActiveConnections int64
	TotalRequests     int64
	PoolHits          int64
	PoolMisses        int64
	mutex             sync.RWMutex
}

func (ps *PoolStats) IncrementRequest() {
	ps.mutex.Lock()
	ps.TotalRequests++
	ps.mutex.Unlock()
}

func (ps *PoolStats) IncrementHit() {
	ps.mutex.Lock()
	ps.PoolHits++
	ps.mutex.Unlock()
}

func (ps *PoolStats) IncrementMiss() {
	ps.mutex.Lock()
	ps.PoolMisses++
	ps.mutex.Unlock()
}

func (ps *PoolStats) GetStats() (int64, int64, int64, int64) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.ActiveConnections, ps.TotalRequests, ps.PoolHits, ps.PoolMisses
}

// 커넥션 래퍼 (슬랙 할당자의 객체 개념)
type PooledConnection struct {
	*sql.DB
	poolSize   int
	lastUsed   time.Time
	inUse      bool
	created    time.Time
}

func NewBuddyConnectionPool(dbSource string) *BuddyConnectionPool {
	bcp := &BuddyConnectionPool{
		pools:    make(map[int]*sync.Pool),
		dbSource: dbSource,
		stats:    &PoolStats{},
	}
	
	// 2의 거듭제곱 크기별 풀 초기화 (1, 2, 4, 8, 16, 32 커넥션)
	for size := 1; size <= 32; size *= 2 {
		currentSize := size
		bcp.pools[size] = &sync.Pool{
			New: func() interface{} {
				return bcp.createConnectionGroup(currentSize)
			},
		}
	}
	
	return bcp
}

func (bcp *BuddyConnectionPool) createConnectionGroup(size int) []*PooledConnection {
	connections := make([]*PooledConnection, size)
	
	for i := 0; i < size; i++ {
		db, err := sql.Open("sqlite3", bcp.dbSource)
		if err != nil {
			log.Printf("커넥션 생성 실패: %v", err)
			continue
		}
		
		connections[i] = &PooledConnection{
			DB:       db,
			poolSize: size,
			created:  time.Now(),
			lastUsed: time.Now(),
		}
	}
	
	return connections
}

func (bcp *BuddyConnectionPool) nextPowerOf2(size int) int {
	power := 1
	for power < size {
		power *= 2
	}
	if power > 32 {
		return 32
	}
	return power
}

func (bcp *BuddyConnectionPool) GetConnection(requestedSize int) *PooledConnection {
	bcp.stats.IncrementRequest()
	
	poolSize := bcp.nextPowerOf2(requestedSize)
	
	bcp.mutex.RLock()
	pool, exists := bcp.pools[poolSize]
	bcp.mutex.RUnlock()
	
	if !exists {
		bcp.stats.IncrementMiss()
		// 직접 생성
		db, err := sql.Open("sqlite3", bcp.dbSource)
		if err != nil {
			log.Printf("직접 커넥션 생성 실패: %v", err)
			return nil
		}
		
		return &PooledConnection{
			DB:       db,
			poolSize: poolSize,
			created:  time.Now(),
			lastUsed: time.Now(),
			inUse:    true,
		}
	}
	
	// 풀에서 커넥션 그룹 가져오기
	connectionGroup := pool.Get().([]*PooledConnection)
	if len(connectionGroup) > 0 {
		bcp.stats.IncrementHit()
		conn := connectionGroup[0]
		conn.inUse = true
		conn.lastUsed = time.Now()
		
		bcp.stats.mutex.Lock()
		bcp.stats.ActiveConnections++
		bcp.stats.mutex.Unlock()
		
		return conn
	}
	
	bcp.stats.IncrementMiss()
	return nil
}

func (bcp *BuddyConnectionPool) PutConnection(conn *PooledConnection) {
	if conn == nil {
		return
	}
	
	conn.inUse = false
	conn.lastUsed = time.Now()
	
	bcp.stats.mutex.Lock()
	bcp.stats.ActiveConnections--
	bcp.stats.mutex.Unlock()
	
	// 커넥션이 너무 오래된 경우 폐기
	if time.Since(conn.created) > 30*time.Minute {
		conn.Close()
		return
	}
	
	// 풀로 반환
	bcp.mutex.RLock()
	pool, exists := bcp.pools[conn.poolSize]
	bcp.mutex.RUnlock()
	
	if exists {
		pool.Put([]*PooledConnection{conn})
	} else {
		conn.Close()
	}
}

// 실무용 데이터베이스 서비스
type UserService struct {
	pool *BuddyConnectionPool
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Created  string `json:"created"`
}

func NewUserService(dbSource string) *UserService {
	pool := NewBuddyConnectionPool(dbSource)
	
	// 테이블 초기화
	conn := pool.GetConnection(1)
	if conn != nil {
		conn.Exec(`
			CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT UNIQUE NOT NULL,
				email TEXT UNIQUE NOT NULL,
				created DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		pool.PutConnection(conn)
	}
	
	return &UserService{pool: pool}
}

func (us *UserService) CreateUser(username, email string) (*User, error) {
	// 단일 커넥션 요청 (크기 1)
	conn := us.pool.GetConnection(1)
	if conn == nil {
		return nil, fmt.Errorf("커넥션을 가져올 수 없습니다")
	}
	defer us.pool.PutConnection(conn)
	
	result, err := conn.Exec(
		"INSERT INTO users (username, email) VALUES (?, ?)",
		username, email,
	)
	if err != nil {
		return nil, err
	}
	
	id, _ := result.LastInsertId()
	
	return &User{
		ID:       int(id),
		Username: username,
		Email:    email,
		Created:  time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (us *UserService) GetUser(id int) (*User, error) {
	conn := us.pool.GetConnection(1)
	if conn == nil {
		return nil, fmt.Errorf("커넥션을 가져올 수 없습니다")
	}
	defer us.pool.PutConnection(conn)
	
	var user User
	err := conn.QueryRow(
		"SELECT id, username, email, created FROM users WHERE id = ?", id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Created)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

func (us *UserService) GetAllUsers() ([]User, error) {
	// 대량 조회를 위해 더 큰 커넥션 풀 요청 (크기 4)
	conn := us.pool.GetConnection(4)
	if conn == nil {
		return nil, fmt.Errorf("커넥션을 가져올 수 없습니다")
	}
	defer us.pool.PutConnection(conn)
	
	rows, err := conn.Query("SELECT id, username, email, created FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Created)
		if err != nil {
			continue
		}
		users = append(users, user)
	}
	
	return users, nil
}

func (us *UserService) BulkCreateUsers(userCount int) error {
	// 대량 작업을 위해 큰 커넥션 풀 요청 (크기 8)
	conn := us.pool.GetConnection(8)
	if conn == nil {
		return fmt.Errorf("커넥션을 가져올 수 없습니다")
	}
	defer us.pool.PutConnection(conn)
	
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare("INSERT INTO users (username, email) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	for i := 0; i < userCount; i++ {
		username := fmt.Sprintf("user_%d", i)
		email := fmt.Sprintf("user_%d@example.com", i)
		
		_, err := stmt.Exec(username, email)
		if err != nil {
			log.Printf("사용자 %d 생성 실패: %v", i, err)
		}
	}
	
	return tx.Commit()
}

func (us *UserService) GetPoolStats() (int64, int64, int64, int64) {
	return us.pool.stats.GetStats()
}

// 실무 시나리오 테스트
func main() {
	userService := NewUserService(":memory:")
	
	// 1. 단일 사용자 생성
	log.Println("=== 단일 사용자 생성 ===")
	user, err := userService.CreateUser("john_doe", "john@example.com")
	if err != nil {
		log.Printf("사용자 생성 실패: %v", err)
	} else {
		log.Printf("생성된 사용자: %+v", user)
	}
	
	// 2. 대량 사용자 생성 (실무에서 자주 발생)
	log.Println("\n=== 대량 사용자 생성 ===")
	start := time.Now()
	err = userService.BulkCreateUsers(1000)
	if err != nil {
		log.Printf("대량 생성 실패: %v", err)
	} else {
		log.Printf("1000명 사용자 생성 완료 (소요시간: %v)", time.Since(start))
	}
	
	// 3. 사용자 조회
	log.Println("\n=== 사용자 조회 ===")
	user, err = userService.GetUser(1)
	if err != nil {
		log.Printf("사용자 조회 실패: %v", err)
	} else {
		log.Printf("조회된 사용자: %+v", user)
	}
	
	// 4. 전체 사용자 조회
	log.Println("\n=== 전체 사용자 조회 ===")
	start = time.Now()
	users, err := userService.GetAllUsers()
	if err != nil {
		log.Printf("전체 조회 실패: %v", err)
	} else {
		log.Printf("총 %d명 사용자 조회 완료 (소요시간: %v)", len(users), time.Since(start))
	}
	
	// 5. 풀 통계 출력
	log.Println("\n=== 커넥션 풀 통계 ===")
	active, total, hits, misses := userService.GetPoolStats()
	log.Printf("활성 커넥션: %d", active)
	log.Printf("총 요청: %d", total)
	log.Printf("풀 적중: %d", hits)
	log.Printf("풀 실패: %d", misses)
	if total > 0 {
		log.Printf("적중률: %.2f%%", float64(hits)/float64(total)*100)
	}
	
	// 6. 동시성 테스트
	log.Println("\n=== 동시성 테스트 ===")
	var wg sync.WaitGroup
	concurrency := 50
	requestsPerGoroutine := 20
	
	start = time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerGoroutine; j++ {
				// 랜덤하게 조회와 생성 수행
				if j%2 == 0 {
					userService.GetUser(j + 1)
				} else {
					userService.CreateUser(
						fmt.Sprintf("concurrent_user_%d_%d", id, j),
						fmt.Sprintf("user_%d_%d@test.com", id, j),
					)
				}
			}
		}(i)
	}
	
	wg.Wait()
	log.Printf("동시성 테스트 완료 (소요시간: %v)", time.Since(start))
	
	// 최종 통계
	active, total, hits, misses = userService.GetPoolStats()
	log.Printf("\n=== 최종 통계 ===")
	log.Printf("총 요청: %d", total)
	log.Printf("풀 적중률: %.2f%%", float64(hits)/float64(total)*100)
}