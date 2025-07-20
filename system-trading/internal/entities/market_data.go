package entities

import (
	"time"
)

type MarketData struct {
	Symbol    Symbol    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Open      float64   `json:"open"`
	Timestamp time.Time `json:"timestamp"`
}

type NewsArticle struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Source      string    `json:"source"`
	Symbols     []Symbol  `json:"symbols,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Sentiment   float64   `json:"sentiment,omitempty"`
	Relevance   float64   `json:"relevance,omitempty"`
	Impact      float64   `json:"impact,omitempty"`
}

type MacroIndicator struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Country   string    `json:"country"`
	Period    string    `json:"period"`
	Timestamp time.Time `json:"timestamp"`
	Impact    string    `json:"impact"`
}

type TechnicalIndicator struct {
	Symbol     Symbol             `json:"symbol"`
	Indicators map[string]float64 `json:"indicators"`
	Timestamp  time.Time          `json:"timestamp"`
}

type SentimentScore struct {
	Symbol    Symbol    `json:"symbol"`
	Score     float64   `json:"score"`
	Confidence float64  `json:"confidence"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}