package engine

import(
	"gitlab.com/around25/products/matching-engine/model"
)

// TradingEngine contains the current order book and information about the service since it was created
type TradingEngine interface {
	Process(order model.Order, trades *[]model.Trade)
	GetOrderBook() OrderBook
	BackupMarket() model.MarketBackup
	LoadMarket(model.MarketBackup) error
	CancelOrder(order model.Order) bool
	ProcessEvent(order model.Order, trades *[]model.Trade) interface{}
}

type tradingEngine struct {
	OrderBook OrderBook
	Symbol    string
}

// NewTradingEngine creates a new trading engine that contains an empty order book and can start receving requests
func NewTradingEngine(marketID string, pricePrecision, volumePrecision int) TradingEngine {
	orderBook := NewOrderBook(marketID, pricePrecision, volumePrecision)
	return &tradingEngine{
		OrderBook: orderBook,
	}
}

// Process a single order and returned all the trades that can be satisfied instantly
func (ngin *tradingEngine) Process(order model.Order, trades *[]model.Trade) {
	ngin.OrderBook.Process(order, trades)
}

func (ngin *tradingEngine) CancelOrder(order model.Order) bool {
	return ngin.OrderBook.Cancel(order)
}

func (ngin *tradingEngine) LoadMarket(market model.MarketBackup) error {
	return ngin.GetOrderBook().Load(market)
}

func (ngin *tradingEngine) BackupMarket() model.MarketBackup {
	return ngin.GetOrderBook().Backup()
}

func (ngin *tradingEngine) ProcessEvent(order model.Order, trades *[]model.Trade) interface{} {
	switch order.EventType {
	case model.CommandType_NewOrder:
		ngin.Process(order, trades)
	case model.CommandType_CancelOrder:
		return ngin.CancelOrder(order)
	default:
		return nil
	}
	return nil
}

func (ngin tradingEngine) GetOrderBook() OrderBook {
	return ngin.OrderBook
}
