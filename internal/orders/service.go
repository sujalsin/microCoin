package orders

import (
	"database/sql"
	"fmt"
	"time"

	"microcoin/internal/database"
	"microcoin/internal/ledger"
	"microcoin/internal/limitbook"
	"microcoin/internal/models"
	"microcoin/internal/quotes"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Service handles order business logic
type Service struct {
	db            *sql.DB
	orderRepo     *database.OrderRepository
	accountRepo   *database.AccountRepository
	ledgerService *ledger.Service
	quotesService *quotes.Service
	orderBooks    map[models.Symbol]*limitbook.OrderBook
}

// NewService creates a new order service
func NewService(db *sql.DB, quotesService *quotes.Service) *Service {
	service := &Service{
		db:            db,
		orderRepo:     database.NewOrderRepository(db),
		accountRepo:   database.NewAccountRepository(db),
		ledgerService: ledger.NewService(db),
		quotesService: quotesService,
		orderBooks:    make(map[models.Symbol]*limitbook.OrderBook),
	}

	// Initialize order books
	service.orderBooks[models.SymbolBTCUSD] = limitbook.NewOrderBook(models.SymbolBTCUSD)
	service.orderBooks[models.SymbolETHUSD] = limitbook.NewOrderBook(models.SymbolETHUSD)

	// Load existing orders into order books
	service.loadOrdersIntoBooks()

	return service
}

// CreateOrder creates a new order
func (s *Service) CreateOrder(userID uuid.UUID, req *models.CreateOrderRequest) (*models.CreateOrderResponse, error) {
	// Validate request
	if err := s.validateOrderRequest(req); err != nil {
		return nil, err
	}

	// Get current quote for market orders
	var fillPrice *decimal.Decimal
	if req.Type == models.OrderTypeMarket {
		quote, err := s.quotesService.GetQuote(req.Symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get quote: %w", err)
		}

		if req.Side == models.OrderSideBuy {
			fillPrice = &quote.Ask
		} else {
			fillPrice = &quote.Bid
		}
	}

	// Calculate required funds
	requiredAmount, err := s.calculateRequiredAmount(req, fillPrice)
	if err != nil {
		return nil, err
	}

	// Check and hold funds
	if err := s.holdFunds(userID, req, requiredAmount); err != nil {
		return nil, err
	}

	// Create order
	order := &models.Order{
		ID:        uuid.New(),
		UserID:    userID,
		Symbol:    req.Symbol,
		Side:      req.Side,
		Type:      req.Type,
		Price:     req.Price,
		Qty:       req.Qty,
		FilledQty: decimal.Zero,
		Status:    models.OrderStatusNew,
		CreatedAt: time.Now(),
	}

	// Save order to database
	if err := s.orderRepo.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Convert to limitbook order
	bookOrder := s.convertToBookOrder(order)

	// Try to match the order
	orderBook := s.orderBooks[req.Symbol]
	trades := orderBook.MatchOrder(bookOrder)

	// Process trades
	var totalFillQty decimal.Decimal
	var totalFillValue decimal.Decimal
	for _, trade := range trades {
		if err := s.processTrade(trade); err != nil {
			// Log error but continue processing other trades
			fmt.Printf("Failed to process trade: %v\n", err)
			continue
		}
		totalFillQty = totalFillQty.Add(trade.Qty)
		totalFillValue = totalFillValue.Add(trade.Price.Mul(trade.Qty))
	}

	// Update order status
	order.FilledQty = totalFillQty
	if order.FilledQty.Equal(order.Qty) {
		order.Status = models.OrderStatusFilled
	} else if order.FilledQty.GreaterThan(decimal.Zero) {
		order.Status = models.OrderStatusPartiallyFilled
	}

	// Update order in database
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.orderRepo.UpdateOrder(tx, order); err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Add to order book if not fully filled
	if order.Status == models.OrderStatusNew || order.Status == models.OrderStatusPartiallyFilled {
		orderBook.AddOrder(bookOrder)
	}

	// Calculate average fill price
	var avgFillPrice *decimal.Decimal
	if totalFillQty.GreaterThan(decimal.Zero) {
		avg := totalFillValue.Div(totalFillQty)
		avgFillPrice = &avg
	}

	return &models.CreateOrderResponse{
		OrderID:      order.ID.String(),
		Status:       order.Status,
		FilledQty:    totalFillQty,
		AvgFillPrice: avgFillPrice,
	}, nil
}

// GetOrder retrieves an order by ID
func (s *Service) GetOrder(orderID uuid.UUID) (*models.Order, error) {
	return s.orderRepo.GetOrderByID(orderID)
}

// GetOrdersByUserID retrieves orders for a user
func (s *Service) GetOrdersByUserID(userID uuid.UUID, limit, offset int) ([]models.Order, error) {
	return s.orderRepo.GetOrdersByUserID(userID, limit, offset)
}

// validateOrderRequest validates an order request
func (s *Service) validateOrderRequest(req *models.CreateOrderRequest) error {
	if req.Qty.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("quantity must be positive")
	}

	if req.Type == models.OrderTypeLimit && (req.Price == nil || req.Price.LessThanOrEqual(decimal.Zero)) {
		return fmt.Errorf("limit orders must have a positive price")
	}

	// Validate symbol
	if req.Symbol != models.SymbolBTCUSD && req.Symbol != models.SymbolETHUSD {
		return fmt.Errorf("invalid symbol: %s", req.Symbol)
	}

	return nil
}

// calculateRequiredAmount calculates the amount of funds required for an order
func (s *Service) calculateRequiredAmount(req *models.CreateOrderRequest, fillPrice *decimal.Decimal) (decimal.Decimal, error) {
	var price decimal.Decimal

	if req.Type == models.OrderTypeMarket {
		if fillPrice == nil {
			return decimal.Zero, fmt.Errorf("fill price required for market orders")
		}
		price = *fillPrice
	} else {
		price = *req.Price
	}

	if req.Side == models.OrderSideBuy {
		// Buy orders require USD
		return price.Mul(req.Qty), nil
	} else {
		// Sell orders require the base currency (BTC/ETH)
		return req.Qty, nil
	}
}

// holdFunds holds funds for an order
func (s *Service) holdFunds(userID uuid.UUID, req *models.CreateOrderRequest, amount decimal.Decimal) error {
	var currency models.Currency

	if req.Side == models.OrderSideBuy {
		currency = models.CurrencyUSD
	} else {
		// Determine base currency from symbol
		if req.Symbol == models.SymbolBTCUSD {
			currency = models.CurrencyBTC
		} else if req.Symbol == models.SymbolETHUSD {
			currency = models.CurrencyETH
		} else {
			return fmt.Errorf("invalid symbol: %s", req.Symbol)
		}
	}

	return s.ledgerService.HoldFunds(userID, currency, amount)
}

// processTrade processes a completed trade
func (s *Service) processTrade(trade *models.Trade) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get accounts
	takerUSD, err := s.accountRepo.GetAccountByUserIDAndCurrency(trade.TakerID, models.CurrencyUSD)
	if err != nil {
		return fmt.Errorf("failed to get taker USD account: %w", err)
	}

	makerUSD, err := s.accountRepo.GetAccountByUserIDAndCurrency(trade.MakerID, models.CurrencyUSD)
	if err != nil {
		return fmt.Errorf("failed to get maker USD account: %w", err)
	}

	var takerBase, makerBase *models.Account
	if trade.Symbol == models.SymbolBTCUSD {
		takerBase, err = s.accountRepo.GetAccountByUserIDAndCurrency(trade.TakerID, models.CurrencyBTC)
		if err != nil {
			return fmt.Errorf("failed to get taker BTC account: %w", err)
		}

		makerBase, err = s.accountRepo.GetAccountByUserIDAndCurrency(trade.MakerID, models.CurrencyBTC)
		if err != nil {
			return fmt.Errorf("failed to get maker BTC account: %w", err)
		}
	} else {
		takerBase, err = s.accountRepo.GetAccountByUserIDAndCurrency(trade.TakerID, models.CurrencyETH)
		if err != nil {
			return fmt.Errorf("failed to get taker ETH account: %w", err)
		}

		makerBase, err = s.accountRepo.GetAccountByUserIDAndCurrency(trade.MakerID, models.CurrencyETH)
		if err != nil {
			return fmt.Errorf("failed to get maker ETH account: %w", err)
		}
	}

	// Calculate trade value
	tradeValue := trade.Price.Mul(trade.Qty)

	// Process the trade based on side
	if trade.Side == models.OrderSideBuy {
		// Taker buys, maker sells
		// Taker: USD -> BTC/ETH
		// Maker: BTC/ETH -> USD

		// Transfer USD from taker to maker
		if err := s.ledgerService.TransferFunds(takerUSD.ID, makerUSD.ID, tradeValue, models.CurrencyUSD, "TRADE", trade.ID); err != nil {
			return fmt.Errorf("failed to transfer USD: %w", err)
		}

		// Transfer base currency from maker to taker
		if trade.Symbol == models.SymbolBTCUSD {
			if err := s.ledgerService.TransferFunds(makerBase.ID, takerBase.ID, trade.Qty, models.CurrencyBTC, "TRADE", trade.ID); err != nil {
				return fmt.Errorf("failed to transfer BTC: %w", err)
			}
		} else {
			if err := s.ledgerService.TransferFunds(makerBase.ID, takerBase.ID, trade.Qty, models.CurrencyETH, "TRADE", trade.ID); err != nil {
				return fmt.Errorf("failed to transfer ETH: %w", err)
			}
		}
	} else {
		// Taker sells, maker buys
		// Taker: BTC/ETH -> USD
		// Maker: USD -> BTC/ETH

		// Transfer base currency from taker to maker
		if trade.Symbol == models.SymbolBTCUSD {
			if err := s.ledgerService.TransferFunds(takerBase.ID, makerBase.ID, trade.Qty, models.CurrencyBTC, "TRADE", trade.ID); err != nil {
				return fmt.Errorf("failed to transfer BTC: %w", err)
			}
		} else {
			if err := s.ledgerService.TransferFunds(takerBase.ID, makerBase.ID, trade.Qty, models.CurrencyETH, "TRADE", trade.ID); err != nil {
				return fmt.Errorf("failed to transfer ETH: %w", err)
			}
		}

		// Transfer USD from maker to taker
		if err := s.ledgerService.TransferFunds(makerUSD.ID, takerUSD.ID, tradeValue, models.CurrencyUSD, "TRADE", trade.ID); err != nil {
			return fmt.Errorf("failed to transfer USD: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// convertToBookOrder converts a models.Order to a limitbook.Order
func (s *Service) convertToBookOrder(order *models.Order) *limitbook.Order {
	return &limitbook.Order{
		ID:        order.ID,
		UserID:    order.UserID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Type:      order.Type,
		Price:     order.Price,
		Qty:       order.Qty,
		FilledQty: order.FilledQty,
		Status:    order.Status,
		CreatedAt: order.CreatedAt,
	}
}

// loadOrdersIntoBooks loads existing orders into order books
func (s *Service) loadOrdersIntoBooks() {
	for symbol := range s.orderBooks {
		orders, err := s.orderRepo.GetActiveOrdersBySymbol(symbol)
		if err != nil {
			fmt.Printf("Failed to load orders for %s: %v\n", symbol, err)
			continue
		}

		for _, order := range orders {
			bookOrder := s.convertToBookOrder(&order)
			s.orderBooks[symbol].AddOrder(bookOrder)
		}
	}
}
