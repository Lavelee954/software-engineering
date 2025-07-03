package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"
)

// 로드 테스트 클라이언트
func loadTest() {
	const (
		numGoroutines        = 100
		requestsPerGoroutine = 100
	)

	var wg sync.WaitGroup
	start := time.Now()

	// 동시 요청 테스트
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				// 다양한 엔드포인트 테스트
				endpoints := []string{
					"http://localhost:8080/api/user",
					"http://localhost:8080/api/bulk",
					"http://localhost:8080/api/metrics",
				}

				endpoint := endpoints[j%len(endpoints)]
				resp, err := http.Get(endpoint)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					continue
				}
				resp.Body.Close()

				if j%50 == 0 {
					fmt.Printf("Goroutine %d: %d requests completed\n", id, j+1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	totalRequests := numGoroutines * requestsPerGoroutine

	fmt.Printf("\n=== 로드 테스트 결과 ===\n")
	fmt.Printf("총 요청 수: %d\n", totalRequests)
	fmt.Printf("소요 시간: %v\n", duration)
	fmt.Printf("초당 요청 수: %.2f\n", float64(totalRequests)/duration.Seconds())
}

// 파일 업로드 테스트
func testFileUpload() {
	// 테스트용 파일 생성
	testData := bytes.Repeat([]byte("테스트 데이터\n"), 1000)

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		fmt.Printf("파일 파트 생성 오류: %v\n", err)
		return
	}

	part.Write(testData)
	writer.Close()

	// 파일 업로드 요청
	resp, err := http.Post("http://localhost:8080/api/upload",
		writer.FormDataContentType(), &b)
	if err != nil {
		fmt.Printf("파일 업로드 오류: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("파일 업로드 응답: %s\n", string(body))
}

// 메트릭 모니터링
func monitorMetrics() {
	for i := 0; i < 10; i++ {
		resp, err := http.Get("http://localhost:8080/api/metrics")
		if err != nil {
			fmt.Printf("메트릭 요청 오류: %v\n", err)
			return
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		fmt.Printf("메트릭 %d: %s\n", i+1, string(body))
		time.Sleep(2 * time.Second)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("사용법: go run client.go [load|upload|metrics]")
		return
	}

	switch os.Args[1] {
	case "load":
		fmt.Println("로드 테스트 시작...")
		loadTest()
	case "upload":
		fmt.Println("파일 업로드 테스트...")
		testFileUpload()
	case "metrics":
		fmt.Println("메트릭 모니터링...")
		monitorMetrics()
	default:
		fmt.Println("알 수 없는 명령어:", os.Args[1])
	}
}
