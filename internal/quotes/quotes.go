package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"microcoin/internal/models"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// Service handles real-time quotes
type Service struct {
	redisClient *redis.Client
	quotes      map[models.Symbol]*models.Quote
	mutex       sync.RWMutex
	subscribers map[models.Symbol][]chan *models.Quote
	subMutex    sync.RWMutex
}

// NewService creates a new quotes service
func NewService(redisClient *redis.Client) *Service {
	return &Service{
		redisClient: redisClient,
		quotes:      make(map[models.Symbol]*models.Quote),
		subscribers: make(map[models.Symbol][]chan *models.Quote),
	}
}

// Start starts the quotes service
func (s *Service) Start(ctx context.Context) error {
	// Subscribe to Redis channels for quotes
	go s.subscribeToQuotes(ctx)
	
	// Start mock quote generator (in production, this would connect to real data feeds)
	go s.generateMockQuotes(ctx)
	
	return nil
}

// GetQuote returns the latest quote for a symbol
func (s *Service) GetQuote(symbol models.Symbol) (*models.Quote, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	quote, exists := s.quotes[symbol]
	if !exists {
		return nil, fmt.Errorf("no quote available for symbol %s", symbol)
	}
	
	return quote, nil
}

// Subscribe subscribes to quote updates for a symbol
func (s *Service) Subscribe(symbol models.Symbol) <-chan *models.Quote {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()
	
	ch := make(chan *models.Quote, 10)
	s.subscribers[symbol] = append(s.subscribers[symbol], ch)
	
	return ch
}

// Unsubscribe unsubscribes from quote updates
func (s *Service) Unsubscribe(symbol models.Symbol, ch <-chan *models.Quote) {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()
	
	subscribers := s.subscribers[symbol]
	for i, subscriber := range subscribers {
		if subscriber == ch {
			s.subscribers[symbol] = append(subscribers[:i], subscribers[i+1:]...)
			close(subscriber)
			break
		}
	}
}

// subscribeToQuotes subscribes to Redis channels for quote updates
func (s *Service) subscribeToQuotes(ctx context.Context) {
	pubsub := s.redisClient.Subscribe(ctx, "quotes:BTC-USD", "quotes:ETH-USD")
	defer pubsub.Close()
	
	ch := pubsub.Channel()
	
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			var quote models.Quote
			if err := json.Unmarshal([]byte(msg.Payload), &quote); err != nil {
				log.Printf("Failed to unmarshal quote: %v", err)
				continue
			}
			
			s.updateQuote(&quote)
		}
	}
}

// updateQuote updates the latest quote and notifies subscribers
func (s *Service) updateQuote(quote *models.Quote) {
	s.mutex.Lock()
	s.quotes[quote.Symbol] = quote
	s.mutex.Unlock()
	
	// Notify subscribers
	s.subMutex.RLock()
	subscribers := s.subscribers[quote.Symbol]
	s.subMutex.RUnlock()
	
	for _, ch := range subscribers {
		select {
		case ch <- quote:
		default:
			// Channel is full, skip this subscriber
		}
	}
}

// generateMockQuotes generates mock quotes for testing
func (s *Service) generateMockQuotes(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	// Initial prices
	btcPrice := decimal.NewFromFloat(60000.0)
	ethPrice := decimal.NewFromFloat(3000.0)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Generate random price movements
			btcChange := decimal.NewFromFloat(0.001).Mul(decimal.NewFromFloat(float64(time.Now().UnixNano()%100 - 50)))
			ethChange := decimal.NewFromFloat(0.001).Mul(decimal.NewFromFloat(float64(time.Now().UnixNano()%100 - 50)))
			
			btcPrice = btcPrice.Add(btcChange)
			ethPrice = ethPrice.Add(ethChange)
			
			// Ensure prices don't go negative
			if btcPrice.LessThan(decimal.Zero) {
				btcPrice = decimal.NewFromFloat(60000.0)
			}
			if ethPrice.LessThan(decimal.Zero) {
				ethPrice = decimal.NewFromFloat(3000.0)
			}
			
			// Create quotes with bid/ask spread
			spread := decimal.NewFromFloat(0.0001) // 0.01% spread
			
			btcQuote := &models.Quote{
				Symbol: models.SymbolBTCUSD,
				Bid:    btcPrice.Sub(btcPrice.Mul(spread)),
				Ask:    btcPrice.Add(btcPrice.Mul(spread)),
				TS:     time.Now(),
			}
			
			ethQuote := &models.Quote{
				Symbol: models.SymbolETHUSD,
				Bid:    ethPrice.Sub(ethPrice.Mul(spread)),
				Ask:    ethPrice.Add(ethPrice.Mul(spread)),
				TS:     time.Now(),
			}
			
			// Publish to Redis
			s.publishQuote(btcQuote)
			s.publishQuote(ethQuote)
		}
	}
}

// publishQuote publishes a quote to Redis
func (s *Service) publishQuote(quote *models.Quote) {
	channel := fmt.Sprintf("quotes:%s", quote.Symbol)
	
	data, err := json.Marshal(quote)
	if err != nil {
		log.Printf("Failed to marshal quote: %v", err)
		return
	}
	
	if err := s.redisClient.Publish(context.Background(), channel, data).Err(); err != nil {
		log.Printf("Failed to publish quote: %v", err)
	}
}
