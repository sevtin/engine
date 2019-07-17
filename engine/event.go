package engine

import (
	"github.com/segmentio/kafka-go"
	"gitlab.com/around25/products/matching-engine/model"
)

// Event structure for order execution
type Event struct {
	Msg    kafka.Message
	Order  model.Order
	Trades []model.Trade
}

// NewEvent Create a new event
func NewEvent(msg kafka.Message) Event {
	return Event{Msg: msg}
}

// Decode the contained message into a proper order
func (event *Event) Decode() {
	event.Order.FromBinary(event.Msg.Value)
}

// SetTrades - sets the generated trades from that order on the current market
func (event *Event) SetTrades(trades []model.Trade) {
	event.Trades = trades
}

// HasTrades checks if there are any trades generated by the order
func (event Event) HasTrades() bool {
	return len(event.Trades) > 0
}
