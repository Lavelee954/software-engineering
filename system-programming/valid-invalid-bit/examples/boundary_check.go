package main

import (
	"fmt"
	"strings"
)

// SafeArray 안전한 배열 구조체 (메모리 보호 개념 적용)
type SafeArray struct {
	data   []interface{}
	size   int
	valid  bool // 유효-무효 비트
	bounds struct {
		min, max int
	}
}

// NewSafeArray 안전한 배열 생성
func NewSafeArray(size int) *SafeArray {
	return &SafeArray{
		data:  make([]interface{}, size),
		size:  size,
		valid: true,
		bounds: struct{ min, max int }{
			min: 0,
			max: size - 1,
		},
	}
}

// Get 안전한 인덱스 접근
func (sa *SafeArray) Get(index int) (interface{}, error) {
	if !sa.valid {
		return nil, fmt.Errorf("메모리 보호: 무효한 배열에 접근 시도")
	}
	
	if index < sa.bounds.min || index > sa.bounds.max {
		return nil, fmt.Errorf("메모리 보호: 인덱스 범위 초과 (index: %d, 범위: %d-%d)", 
			index, sa.bounds.min, sa.bounds.max)
	}
	
	return sa.data[index], nil
}

// Set 안전한 값 설정
func (sa *SafeArray) Set(index int, value interface{}) error {
	if !sa.valid {
		return fmt.Errorf("메모리 보호: 무효한 배열에 쓰기 시도")
	}
	
	if index < sa.bounds.min || index > sa.bounds.max {
		return fmt.Errorf("메모리 보호: 인덱스 범위 초과 (index: %d, 범위: %d-%d)", 
			index, sa.bounds.min, sa.bounds.max)
	}
	
	sa.data[index] = value
	return nil
}

// SafeSlice 안전한 슬라이스 접근 함수
func SafeSliceAccess(slice []int, index int) (int, error) {
	if slice == nil {
		return 0, fmt.Errorf("메모리 보호: nil 슬라이스에 접근 시도")
	}
	
	if index < 0 || index >= len(slice) {
		return 0, fmt.Errorf("메모리 보호: 슬라이스 범위 초과 (index: %d, 길이: %d)", 
			index, len(slice))
	}
	
	return slice[index], nil
}

// SafeSliceWrite 안전한 슬라이스 쓰기 함수
func SafeSliceWrite(slice []int, index, value int) error {
	if slice == nil {
		return fmt.Errorf("메모리 보호: nil 슬라이스에 쓰기 시도")
	}
	
	if index < 0 || index >= len(slice) {
		return fmt.Errorf("메모리 보호: 슬라이스 범위 초과 (index: %d, 길이: %d)", 
			index, len(slice))
	}
	
	slice[index] = value
	return nil
}

// SafeString 안전한 문자열 접근 함수
func SafeStringAccess(str string, index int) (byte, error) {
	if index < 0 || index >= len(str) {
		return 0, fmt.Errorf("메모리 보호: 문자열 범위 초과 (index: %d, 길이: %d)", 
			index, len(str))
	}
	
	return str[index], nil
}

// SafeSubstring 안전한 부분 문자열 추출
func SafeSubstring(str string, start, end int) (string, error) {
	if start < 0 || start > len(str) {
		return "", fmt.Errorf("메모리 보호: 시작 인덱스 범위 초과 (start: %d, 길이: %d)", 
			start, len(str))
	}
	
	if end < start || end > len(str) {
		return "", fmt.Errorf("메모리 보호: 종료 인덱스 범위 초과 (end: %d, 길이: %d)", 
			end, len(str))
	}
	
	return str[start:end], nil
}

// UserInputValidator 사용자 입력 검증기
type UserInputValidator struct {
	maxLength int
	allowedChars string
}

// NewUserInputValidator 사용자 입력 검증기 생성
func NewUserInputValidator(maxLength int, allowedChars string) *UserInputValidator {
	return &UserInputValidator{
		maxLength: maxLength,
		allowedChars: allowedChars,
	}
}

// ValidateInput 입력 검증
func (v *UserInputValidator) ValidateInput(input string) error {
	// 길이 검증
	if len(input) > v.maxLength {
		return fmt.Errorf("입력 길이 초과: %d (최대: %d)", len(input), v.maxLength)
	}
	
	// 허용된 문자 검증
	for i, char := range input {
		if !strings.ContainsRune(v.allowedChars, char) {
			return fmt.Errorf("허용되지 않은 문자: '%c' (위치: %d)", char, i)
		}
	}
	
	return nil
}

// ProductManager 상품 관리자 (경계 검사 적용)
type ProductManager struct {
	products []Product
}

// Product 상품 구조체
type Product struct {
	ID    int
	Name  string
	Price float64
}

// NewProductManager 상품 관리자 생성
func NewProductManager() *ProductManager {
	return &ProductManager{
		products: []Product{
			{ID: 1, Name: "노트북", Price: 1500000},
			{ID: 2, Name: "마우스", Price: 50000},
			{ID: 3, Name: "키보드", Price: 100000},
		},
	}
}

// GetProduct 상품 조회 (안전한 접근)
func (pm *ProductManager) GetProduct(index int) (*Product, error) {
	if index < 0 || index >= len(pm.products) {
		return nil, fmt.Errorf("상품을 찾을 수 없습니다: 인덱스 %d (범위: 0-%d)", 
			index, len(pm.products)-1)
	}
	
	return &pm.products[index], nil
}

// GetProductByID 상품 ID로 조회
func (pm *ProductManager) GetProductByID(id int) (*Product, error) {
	for i, product := range pm.products {
		if product.ID == id {
			return &pm.products[i], nil
		}
	}
	
	return nil, fmt.Errorf("상품을 찾을 수 없습니다: ID %d", id)
}

// CSVParser CSV 파서 (경계 검사 적용)
type CSVParser struct {
	data [][]string
}

// NewCSVParser CSV 파서 생성
func NewCSVParser(csvData string) *CSVParser {
	lines := strings.Split(csvData, "\n")
	data := make([][]string, 0, len(lines))
	
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			fields := strings.Split(line, ",")
			data = append(data, fields)
		}
	}
	
	return &CSVParser{data: data}
}

// GetCell 셀 값 가져오기
func (csv *CSVParser) GetCell(row, col int) (string, error) {
	if row < 0 || row >= len(csv.data) {
		return "", fmt.Errorf("행 인덱스 범위 초과: %d (범위: 0-%d)", 
			row, len(csv.data)-1)
	}
	
	if col < 0 || col >= len(csv.data[row]) {
		return "", fmt.Errorf("열 인덱스 범위 초과: %d (범위: 0-%d)", 
			col, len(csv.data[row])-1)
	}
	
	return csv.data[row][col], nil
}

// GetRow 행 데이터 가져오기
func (csv *CSVParser) GetRow(row int) ([]string, error) {
	if row < 0 || row >= len(csv.data) {
		return nil, fmt.Errorf("행 인덱스 범위 초과: %d (범위: 0-%d)", 
			row, len(csv.data)-1)
	}
	
	return csv.data[row], nil
}

// APIRequestHandler API 요청 처리기
type APIRequestHandler struct {
	maxRequestSize int
	maxParams      int
}

// NewAPIRequestHandler API 요청 처리기 생성
func NewAPIRequestHandler(maxRequestSize, maxParams int) *APIRequestHandler {
	return &APIRequestHandler{
		maxRequestSize: maxRequestSize,
		maxParams:      maxParams,
	}
}

// ProcessRequest 요청 처리
func (handler *APIRequestHandler) ProcessRequest(requestData []byte, params map[string]string) (string, error) {
	// 요청 크기 검증
	if len(requestData) > handler.maxRequestSize {
		return "", fmt.Errorf("요청 크기 초과: %d 바이트 (최대: %d)", 
			len(requestData), handler.maxRequestSize)
	}
	
	// 파라미터 개수 검증
	if len(params) > handler.maxParams {
		return "", fmt.Errorf("파라미터 개수 초과: %d (최대: %d)", 
			len(params), handler.maxParams)
	}
	
	// 파라미터 검증
	for key, value := range params {
		if len(key) > 100 {
			return "", fmt.Errorf("파라미터 키 길이 초과: %s", key)
		}
		if len(value) > 1000 {
			return "", fmt.Errorf("파라미터 값 길이 초과: %s", key)
		}
	}
	
	return "요청 처리 완료", nil
}

func main() {
	fmt.Println("=== Boundary Check 예제 (메모리 보호 개념 적용) ===")
	
	// 예제 1: 안전한 배열 사용
	fmt.Println("\n1. 안전한 배열 사용:")
	safeArray := NewSafeArray(5)
	
	// 정상적인 접근
	safeArray.Set(0, "첫 번째 항목")
	safeArray.Set(1, "두 번째 항목")
	
	value, err := safeArray.Get(0)
	if err != nil {
		fmt.Printf("오류: %v\n", err)
	} else {
		fmt.Printf("값: %v\n", value)
	}
	
	// 범위 초과 접근 시도
	_, err = safeArray.Get(10)
	if err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 예제 2: 안전한 슬라이스 접근
	fmt.Println("\n2. 안전한 슬라이스 접근:")
	numbers := []int{10, 20, 30, 40, 50}
	
	// 정상적인 접근
	value, err = SafeSliceAccess(numbers, 2)
	if err != nil {
		fmt.Printf("오류: %v\n", err)
	} else {
		fmt.Printf("값: %d\n", value)
	}
	
	// 범위 초과 접근 시도
	_, err = SafeSliceAccess(numbers, -1)
	if err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 예제 3: 사용자 입력 검증
	fmt.Println("\n3. 사용자 입력 검증:")
	validator := NewUserInputValidator(10, "abcdefghijklmnopqrstuvwxyz0123456789")
	
	// 유효한 입력
	err = validator.ValidateInput("abc123")
	if err == nil {
		fmt.Println("유효한 입력: abc123")
	}
	
	// 무효한 입력 (길이 초과)
	err = validator.ValidateInput("verylonginput123")
	if err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 무효한 입력 (허용되지 않은 문자)
	err = validator.ValidateInput("abc@123")
	if err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 예제 4: 상품 관리자
	fmt.Println("\n4. 상품 관리자:")
	productManager := NewProductManager()
	
	// 정상적인 상품 조회
	product, err := productManager.GetProduct(1)
	if err != nil {
		fmt.Printf("오류: %v\n", err)
	} else {
		fmt.Printf("상품: %s, 가격: %.0f원\n", product.Name, product.Price)
	}
	
	// 범위 초과 상품 조회
	_, err = productManager.GetProduct(10)
	if err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 예제 5: CSV 파서
	fmt.Println("\n5. CSV 파서:")
	csvData := `이름,나이,이메일
김철수,30,kim@example.com
이영희,25,lee@example.com
박민수,35,park@example.com`
	
	csvParser := NewCSVParser(csvData)
	
	// 정상적인 셀 접근
	cell, err := csvParser.GetCell(1, 1)
	if err != nil {
		fmt.Printf("오류: %v\n", err)
	} else {
		fmt.Printf("셀 값: %s\n", cell)
	}
	
	// 범위 초과 셀 접근
	_, err = csvParser.GetCell(10, 1)
	if err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 예제 6: API 요청 처리
	fmt.Println("\n6. API 요청 처리:")
	apiHandler := NewAPIRequestHandler(1024, 10)
	
	// 정상적인 요청
	requestData := []byte(`{"action":"get_user","user_id":"123"}`)
	params := map[string]string{
		"format": "json",
		"lang":   "ko",
	}
	
	result, err := apiHandler.ProcessRequest(requestData, params)
	if err != nil {
		fmt.Printf("오류: %v\n", err)
	} else {
		fmt.Printf("결과: %s\n", result)
	}
	
	// 요청 크기 초과
	largeRequest := make([]byte, 2048)
	_, err = apiHandler.ProcessRequest(largeRequest, params)
	if err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
	
	// 예제 7: 안전한 문자열 접근
	fmt.Println("\n7. 안전한 문자열 접근:")
	testString := "Hello, World!"
	
	// 정상적인 접근
	char, err := SafeStringAccess(testString, 7)
	if err != nil {
		fmt.Printf("오류: %v\n", err)
	} else {
		fmt.Printf("문자: %c\n", char)
	}
	
	// 안전한 부분 문자열 추출
	substr, err := SafeSubstring(testString, 0, 5)
	if err != nil {
		fmt.Printf("오류: %v\n", err)
	} else {
		fmt.Printf("부분 문자열: %s\n", substr)
	}
	
	// 범위 초과 접근
	_, err = SafeStringAccess(testString, 100)
	if err != nil {
		fmt.Printf("예상된 오류: %v\n", err)
	}
}