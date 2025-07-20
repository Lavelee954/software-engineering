package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/system-trading/core/internal/entities"
)

type Validator struct {
	emailRegex *regexp.Regexp
}

func NewValidator() *Validator {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return &Validator{
		emailRegex: emailRegex,
	}
}

func (v *Validator) ValidateOrder(order *entities.Order) error {
	if order == nil {
		return fmt.Errorf("order cannot be nil")
	}

	if order.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if !v.isValidSymbol(string(order.Symbol)) {
		return fmt.Errorf("invalid symbol format: %s", order.Symbol)
	}

	if order.Side != entities.OrderSideBuy && order.Side != entities.OrderSideSell {
		return fmt.Errorf("invalid order side: %s", order.Side)
	}

	if order.Type != entities.OrderTypeMarket && 
	   order.Type != entities.OrderTypeLimit && 
	   order.Type != entities.OrderTypeStop {
		return fmt.Errorf("invalid order type: %s", order.Type)
	}

	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive, got: %f", order.Quantity)
	}

	if order.Type == entities.OrderTypeLimit || order.Type == entities.OrderTypeStop {
		if order.Price == nil {
			return fmt.Errorf("price is required for %s orders", order.Type)
		}
		if *order.Price <= 0 {
			return fmt.Errorf("price must be positive, got: %f", *order.Price)
		}
	}

	return nil
}

func (v *Validator) ValidateMarketData(data *entities.MarketData) error {
	if data == nil {
		return fmt.Errorf("market data cannot be nil")
	}

	if data.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if !v.isValidSymbol(string(data.Symbol)) {
		return fmt.Errorf("invalid symbol format: %s", data.Symbol)
	}

	if data.Price <= 0 {
		return fmt.Errorf("price must be positive, got: %f", data.Price)
	}

	if data.Volume < 0 {
		return fmt.Errorf("volume cannot be negative, got: %f", data.Volume)
	}

	if data.Bid <= 0 || data.Ask <= 0 {
		return fmt.Errorf("bid and ask prices must be positive, got bid: %f, ask: %f", data.Bid, data.Ask)
	}

	if data.Bid >= data.Ask {
		return fmt.Errorf("bid price (%f) must be less than ask price (%f)", data.Bid, data.Ask)
	}

	if data.High < data.Low {
		return fmt.Errorf("high price (%f) cannot be less than low price (%f)", data.High, data.Low)
	}

	if data.Price < data.Low || data.Price > data.High {
		return fmt.Errorf("current price (%f) must be between low (%f) and high (%f)", data.Price, data.Low, data.High)
	}

	return nil
}

func (v *Validator) ValidatePortfolio(portfolio *entities.Portfolio) error {
	if portfolio == nil {
		return fmt.Errorf("portfolio cannot be nil")
	}

	if portfolio.ID == "" {
		return fmt.Errorf("portfolio ID is required")
	}

	if portfolio.Cash < 0 {
		return fmt.Errorf("cash balance cannot be negative, got: %f", portfolio.Cash)
	}

	if portfolio.TotalValue < 0 {
		return fmt.Errorf("total value cannot be negative, got: %f", portfolio.TotalValue)
	}

	for symbol, position := range portfolio.Positions {
		if err := v.ValidatePosition(position); err != nil {
			return fmt.Errorf("invalid position for %s: %w", symbol, err)
		}
	}

	return nil
}

func (v *Validator) ValidatePosition(position *entities.Position) error {
	if position == nil {
		return fmt.Errorf("position cannot be nil")
	}

	if position.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if !v.isValidSymbol(string(position.Symbol)) {
		return fmt.Errorf("invalid symbol format: %s", position.Symbol)
	}

	if position.Quantity == 0 {
		return fmt.Errorf("position quantity cannot be zero")
	}

	if position.AveragePrice <= 0 {
		return fmt.Errorf("average price must be positive, got: %f", position.AveragePrice)
	}

	if position.CurrentPrice <= 0 {
		return fmt.Errorf("current price must be positive, got: %f", position.CurrentPrice)
	}

	expectedMarketValue := position.Quantity * position.CurrentPrice
	if abs(position.MarketValue-expectedMarketValue) > 0.01 {
		return fmt.Errorf("market value mismatch: expected %f, got %f", expectedMarketValue, position.MarketValue)
	}

	expectedUnrealizedPnL := (position.CurrentPrice - position.AveragePrice) * position.Quantity
	if abs(position.UnrealizedPnL-expectedUnrealizedPnL) > 0.01 {
		return fmt.Errorf("unrealized PnL mismatch: expected %f, got %f", expectedUnrealizedPnL, position.UnrealizedPnL)
	}

	return nil
}

func (v *Validator) ValidateNewsArticle(article *entities.NewsArticle) error {
	if article == nil {
		return fmt.Errorf("news article cannot be nil")
	}

	if article.ID == "" {
		return fmt.Errorf("article ID is required")
	}

	if strings.TrimSpace(article.Title) == "" {
		return fmt.Errorf("article title is required")
	}

	if strings.TrimSpace(article.Content) == "" {
		return fmt.Errorf("article content is required")
	}

	if strings.TrimSpace(article.Source) == "" {
		return fmt.Errorf("article source is required")
	}

	if article.Sentiment != 0 && (article.Sentiment < -1 || article.Sentiment > 1) {
		return fmt.Errorf("sentiment score must be between -1 and 1, got: %f", article.Sentiment)
	}

	if article.Relevance != 0 && (article.Relevance < 0 || article.Relevance > 1) {
		return fmt.Errorf("relevance score must be between 0 and 1, got: %f", article.Relevance)
	}

	for _, symbol := range article.Symbols {
		if !v.isValidSymbol(string(symbol)) {
			return fmt.Errorf("invalid symbol in article: %s", symbol)
		}
	}

	return nil
}

func (v *Validator) ValidateMacroIndicator(indicator *entities.MacroIndicator) error {
	if indicator == nil {
		return fmt.Errorf("macro indicator cannot be nil")
	}

	if strings.TrimSpace(indicator.Name) == "" {
		return fmt.Errorf("indicator name is required")
	}

	if strings.TrimSpace(indicator.Country) == "" {
		return fmt.Errorf("country is required")
	}

	if strings.TrimSpace(indicator.Period) == "" {
		return fmt.Errorf("period is required")
	}

	validImpacts := []string{"LOW", "MEDIUM", "HIGH"}
	if !contains(validImpacts, indicator.Impact) {
		return fmt.Errorf("invalid impact level: %s, must be one of %v", indicator.Impact, validImpacts)
	}

	return nil
}

func (v *Validator) ValidateStruct(s interface{}) error {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return fmt.Errorf("struct cannot be nil")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if tag := fieldType.Tag.Get("validate"); tag != "" {
			if err := v.validateField(field, fieldType.Name, tag); err != nil {
				return fmt.Errorf("field %s: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

func (v *Validator) validateField(field reflect.Value, fieldName, tag string) error {
	rules := strings.Split(tag, ",")

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)

		switch {
		case rule == "required":
			if v.isEmptyValue(field) {
				return fmt.Errorf("is required")
			}
		case strings.HasPrefix(rule, "min="):
			if err := v.validateMin(field, rule[4:]); err != nil {
				return err
			}
		case strings.HasPrefix(rule, "max="):
			if err := v.validateMax(field, rule[4:]); err != nil {
				return err
			}
		case rule == "email":
			if field.Kind() == reflect.String {
				if !v.emailRegex.MatchString(field.String()) {
					return fmt.Errorf("invalid email format")
				}
			}
		case rule == "symbol":
			if field.Kind() == reflect.String {
				if !v.isValidSymbol(field.String()) {
					return fmt.Errorf("invalid symbol format")
				}
			}
		}
	}

	return nil
}

func (v *Validator) isEmptyValue(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.String:
		return field.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return field.Float() == 0
	case reflect.Bool:
		return !field.Bool()
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return field.IsNil()
	}
	return false
}

func (v *Validator) validateMin(field reflect.Value, value string) error {
	// Implementation for min validation
	return nil
}

func (v *Validator) validateMax(field reflect.Value, value string) error {
	// Implementation for max validation
	return nil
}

func (v *Validator) isValidSymbol(symbol string) bool {
	if len(symbol) < 1 || len(symbol) > 10 {
		return false
	}
	
	symbolRegex := regexp.MustCompile(`^[A-Z0-9]+$`)
	return symbolRegex.MatchString(strings.ToUpper(symbol))
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}