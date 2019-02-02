package engine

// OrderBook interface
// Defines what constitudes an order book and how we can interact with it
type OrderBook interface {
	Process(Order) []Trade
	Cancel(order Order) bool
	GetHighestBid() uint64
	GetLowestAsk() uint64
	Load(MarketBackup) error
	Backup() MarketBackup
	GetMarket() []*SkipList
}

type orderBook struct {
	BuyEntries        *SkipList
	SellEntries       *SkipList
	LowestAsk         uint64
	HighestBid        uint64
	BuyMarketEntries  []Order
	SellMarketEntries []Order
}

// NewOrderBook Creates a new empty order book for the trading engine
func NewOrderBook() OrderBook {
	return &orderBook{
		LowestAsk:         0,
		HighestBid:        0,
		BuyEntries:        NewPricePoints(),
		SellEntries:       NewPricePoints(),
		BuyMarketEntries:  make([]Order, 0, 0),
		SellMarketEntries: make([]Order, 0, 0),
	}
}

// GetHighestBid returns the highest bid of the current market
func (book orderBook) GetHighestBid() uint64 {
	return book.HighestBid
}

// GetLowestAsk returns the lowest ask of the current market
func (book orderBook) GetLowestAsk() uint64 {
	return book.LowestAsk
}

// GetMarket returns the list of price points for the market
// @todo Add sell entries
func (book orderBook) GetMarket() []*SkipList {
	return []*SkipList{book.BuyEntries, book.SellEntries}
}

// Process a new received order and return a list of trades make
func (book *orderBook) Process(order Order) []Trade {
	switch order.EventType {
	case CommandType_NewOrder:
		return book.processOrder(order)
		// case EventTypeCancelOrder: return book.Cancel(order.ID);
		// case EventTypeBackupMarket: return book.Backup();
	}
	return []Trade{}
}

// Process a new order and return a list of trades resulted from the exchange
func (book *orderBook) processOrder(order Order) []Trade {
	// for limit orders first process the limit order with the orderbook since you
	// can't have a pending market order and not have an empty order book
	if order.Type == OrderType_Limit && order.Side == MarketSide_Buy {
		trades := book.processLimitBuy(order)
		// if there are sell market orders pending then keep loading the first one until there are
		// no more pending market orders or the limit order was filled.
		sellMarketCount := len(book.SellMarketEntries)
		if sellMarketCount == 0 {
			return trades
		}
		for i := 0; i < sellMarketCount; i++ {
			mko := book.popMarketSellOrder()
			mkOrder, mkTrades := book.processMarketSell(*mko)
			trades = append(trades, mkTrades...)
			if mkOrder.Status != OrderStatus_Filled {
				book.lpushMarketSellOrder(mkOrder)
				break
			}
		}
		return trades
	}
	// similar to the above for sell limit orders
	if order.Type == OrderType_Limit && order.Side == MarketSide_Sell {
		trades := book.processLimitSell(order)
		buyMarketCount := len(book.BuyMarketEntries)
		if buyMarketCount == 0 {
			return trades
		}
		for i := 0; i < buyMarketCount; i++ {
			mko := book.popMarketBuyOrder()
			mkOrder, mkTrades := book.processMarketBuy(*mko)
			trades = append(trades, mkTrades...)
			if mkOrder.Status != OrderStatus_Filled {
				book.lpushMarketBuyOrder(mkOrder)
				break
			}
		}
		return trades
	}

	// market order either get filly filled or they get added to the pending market order list
	if order.Type == OrderType_Market && order.Side == MarketSide_Buy {
		if len(book.BuyMarketEntries) > 0 {
			book.pushMarketBuyOrder(order)
		} else {
			order, trades := book.processMarketBuy(order)
			if order.Status != OrderStatus_Filled {
				book.pushMarketBuyOrder(order)
			}
			return trades
		}
	}
	// exactly the same for sell market orders
	if order.Type == OrderType_Market && order.Side == MarketSide_Sell {
		if len(book.SellMarketEntries) > 0 {
			book.pushMarketSellOrder(order)
		} else {
			order, trades := book.processMarketSell(order)
			if order.Status != OrderStatus_Filled {
				book.pushMarketSellOrder(order)
			}
			return trades
		}
	}

	return []Trade{}
}

// Cancel an order from the order book based on the order price and ID
//
// @todo This method does not properly implement the cancelation of an order
// - Improve this by also calculating the new LowestAsk and HighestBid after the order is removed
func (book *orderBook) Cancel(order Order) bool {
	switch order.Type {
	case OrderType_Limit:
		return book.cancelLimitOrder(order)
	case OrderType_Market:
		return book.cancelMarketOrder(order)
	default:
		return false
	}
}

// Cancel a limit order based in order attributes
func (book *orderBook) cancelLimitOrder(order Order) bool {
	if order.Side == MarketSide_Buy {
		if pricePoint, ok := book.BuyEntries.Get(order.Price); ok {
			for i := 0; i < len(pricePoint.Entries); i++ {
				if pricePoint.Entries[i].Order.ID == order.ID {
					bookEntry := pricePoint.Entries[i]
					book.removeBuyBookEntry(&bookEntry, pricePoint, i)
					return true
				}
			}
		}
		return false
	}
	if pricePoint, ok := book.SellEntries.Get(order.Price); ok {
		for i := 0; i < len(pricePoint.Entries); i++ {
			if pricePoint.Entries[i].Order.ID == order.ID {
				bookEntry := pricePoint.Entries[i]
				book.removeSellBookEntry(&bookEntry, pricePoint, i)
				return true
			}
		}
	}
	return false
}

func (book *orderBook) cancelMarketOrder(order Order) bool {
	if order.Side == MarketSide_Buy {
		for i := 0; i < len(book.BuyMarketEntries); i++ {
			if order.ID == book.BuyMarketEntries[i].ID {
				book.BuyMarketEntries = append(book.BuyMarketEntries[:i], book.BuyMarketEntries[i+1:]...)
				return true
			}
		}
		return false
	}
	for i := 0; i < len(book.SellMarketEntries); i++ {
		if order.ID == book.SellMarketEntries[i].ID {
			book.SellMarketEntries = append(book.SellMarketEntries[:i], book.SellMarketEntries[i+1:]...)
			return true
		}
	}
	return false
}

// Add a new book entry in the order book
// If the price point already exists then the book entry is simply added at the end of the pricepoint
// If the price point does not exist yet it will be created

func (book *orderBook) addBuyBookEntry(bookEntry BookEntry) {
	price := bookEntry.Price
	if pricePoint, ok := book.BuyEntries.Get(price); ok {
		pricePoint.Entries = append(pricePoint.Entries, bookEntry)
		return
	}
	book.BuyEntries.Set(price, &PricePoint{
		Entries: []BookEntry{bookEntry},
	})
}

func (book *orderBook) addSellBookEntry(bookEntry BookEntry) {
	price := bookEntry.Price
	if pricePoint, ok := book.SellEntries.Get(price); ok {
		pricePoint.Entries = append(pricePoint.Entries, bookEntry)
		return
	}
	book.SellEntries.Set(price, &PricePoint{
		Entries: []BookEntry{bookEntry},
	})
}

// Remove a book entry from the order book
// The method will also remove the price point entry if both book entry lists are empty

func (book *orderBook) removeBuyBookEntry(bookEntry *BookEntry, pricePoint *PricePoint, index int) {
	pricePoint.Entries = append(pricePoint.Entries[:index], pricePoint.Entries[index+1:]...)
	if len(pricePoint.Entries) == 0 {
		book.BuyEntries.Delete(bookEntry.Price)
	}
}

func (book *orderBook) removeSellBookEntry(bookEntry *BookEntry, pricePoint *PricePoint, index int) {
	pricePoint.Entries = append(pricePoint.Entries[:index], pricePoint.Entries[index+1:]...)
	if len(pricePoint.Entries) == 0 {
		book.SellEntries.Delete(bookEntry.Price)
	}
}

func (book *orderBook) pushMarketBuyOrder(order Order) {
	book.BuyMarketEntries = append(book.BuyMarketEntries, order)
}

func (book *orderBook) pushMarketSellOrder(order Order) {
	book.SellMarketEntries = append(book.SellMarketEntries, order)
}

func (book *orderBook) lpushMarketBuyOrder(order Order) {
	book.BuyMarketEntries = append([]Order{order}, book.BuyMarketEntries...)
}

func (book *orderBook) lpushMarketSellOrder(order Order) {
	book.SellMarketEntries = append([]Order{order}, book.SellMarketEntries...)
}

func (book *orderBook) popMarketSellOrder() *Order {
	if len(book.SellMarketEntries) == 0 {
		return nil
	}
	index := 0
	order := book.SellMarketEntries[0]
	book.SellMarketEntries = append(book.SellMarketEntries[:index], book.SellMarketEntries[index+1:]...)
	return &order
}

func (book *orderBook) popMarketBuyOrder() *Order {
	if len(book.BuyMarketEntries) == 0 {
		return nil
	}
	index := 0
	order := book.BuyMarketEntries[0]
	book.BuyMarketEntries = append(book.BuyMarketEntries[:index], book.BuyMarketEntries[index+1:]...)
	return &order
}
