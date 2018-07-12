package trading_engine

// OrderBook interface
// Defines what constitudes an order book and how we can interact with it
type OrderBook interface {
	Process(Order) []Trade
	Cancel(id string) *BookEntry
	GetHighestBid() uint64
	GetLowestAsk() uint64
	Load(Market) error
	Backup() Market
	GetMarket() []*SkipList
}

type orderBook struct {
	BuyEntries  *SkipList
	SellEntries *SkipList
	LowestAsk   uint64
	HighestBid  uint64
	OpenOrders  map[string]uint64
}

// NewOrderBook Creates a new empty order book for the trading engine
func NewOrderBook() OrderBook {
	return &orderBook{
		LowestAsk:   0,
		HighestBid:  0,
		BuyEntries:  NewPricePoints(),
		SellEntries: NewPricePoints(),
		OpenOrders:  make(map[string]uint64),
	}
}

func (book *orderBook) AddOpenOrder(id string, price uint64) {
	book.OpenOrders[id] = price
}

func (book *orderBook) RemoveOpenOrder(id string) {
	delete(book.OpenOrders, id)
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
	if order.Side == BUY {
		return book.processLimitBuy(order)
	}
	return book.processLimitSell(order)
}

// Cancel an order from the order book based on the order price and ID
//
// @todo This method does not properly implement the cancelation of an order
// - Improve this by also calculating the new LowestAsk and HighestBid after the order is removed
func (book *orderBook) Cancel(id string) *BookEntry {
	price, ok := book.OpenOrders[id]
	if !ok {
		return nil
	}

	if pricePoint, ok := book.BuyEntries.Get(price); ok {
		for i := 0; i < len(pricePoint.Entries); i++ {
			if pricePoint.Entries[i].Order.ID == id {
				bookEntry := pricePoint.Entries[i]
				book.removeBuyBookEntry(&bookEntry, pricePoint, i)
				return &bookEntry
			}
		}
	}
	if pricePoint, ok := book.SellEntries.Get(price); ok {
		for i := 0; i < len(pricePoint.Entries); i++ {
			if pricePoint.Entries[i].Order.ID == id {
				bookEntry := pricePoint.Entries[i]
				book.removeSellBookEntry(&bookEntry, pricePoint, i)
				return &bookEntry
			}
		}
	}
	return nil
}

// Add a new book entry in the order book
// If the price point already exists then the book entry is simply added at the end of the pricepoint
// If the price point does not exist yet it will be created

func (book *orderBook) addBuyBookEntry(bookEntry BookEntry) {
	price := bookEntry.Price
	if pricePoint, ok := book.BuyEntries.Get(price); ok {
		pricePoint.Entries = append(pricePoint.Entries, bookEntry)
		book.AddOpenOrder(bookEntry.Order.ID, price)
		return
	}
	book.BuyEntries.Set(price, &PricePoint{
		Entries: []BookEntry{bookEntry},
	})
	book.AddOpenOrder(bookEntry.Order.ID, price)
}

func (book *orderBook) addSellBookEntry(bookEntry BookEntry) {
	price := bookEntry.Price
	if pricePoint, ok := book.SellEntries.Get(price); ok {
		pricePoint.Entries = append(pricePoint.Entries, bookEntry)
		book.AddOpenOrder(bookEntry.Order.ID, price)
		return
	}
	book.SellEntries.Set(price, &PricePoint{
		Entries: []BookEntry{bookEntry},
	})
	book.AddOpenOrder(bookEntry.Order.ID, price)
}

// Remove a book entry from the order book
// The method will also remove the price point entry if both book entry lists are empty

func (book *orderBook) removeBuyBookEntry(bookEntry *BookEntry, pricePoint *PricePoint, index int) {
	book.RemoveOpenOrder(bookEntry.Order.ID)
	pricePoint.Entries = append(pricePoint.Entries[:index], pricePoint.Entries[index+1:]...)
	if len(pricePoint.Entries) == 0 {
		book.BuyEntries.Delete(bookEntry.Price)
	}
}

func (book *orderBook) removeSellBookEntry(bookEntry *BookEntry, pricePoint *PricePoint, index int) {
	book.RemoveOpenOrder(bookEntry.Order.ID)
	pricePoint.Entries = append(pricePoint.Entries[:index], pricePoint.Entries[index+1:]...)
	if len(pricePoint.Entries) == 0 {
		book.SellEntries.Delete(bookEntry.Price)
	}
}
