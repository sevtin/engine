package engine

import (
	"gitlab.com/around25/products/matching-engine/utils"
)

/**
Market Orders
=============

This file defines how market orders are processed by the matching engine.
A market order tells the exchange to fill the order at any price available in the market.
Therefore a market order matches can not be matched by other market orders.

Therefore in regards to market orders we have the following scenarios:

1. DONE: New Market Order, existing limit orders
2. DONE: New Market Order, empty order book
3. DONE: New Market Order, only existing market orders
4. New Limit Order, existing market orders

Let's see how each one should be handled individually and how they are treated by the engine.

1. New Market Order, existing limit orders

In the case when there are existing limit orders in the order book and a new market order is added,
the engine will match the market order with the existing orders based on the priority of those orders
and decrease the amount looking to fill with the limit orders one by one until the entire market order
has been filled or the amount available as funds has been reached.

In case the amount can not be reached by the available limit orders then the remaining amount will be
stored in the pending buy/sell market array until another limit order is added.

2. New Market Order, empty order book

If a market order is added in the order book before there are any available limit orders to fill it
then it will be placed in the pending buy/sell market array and wait for another order that can fillfil it.

This is cause by the requirement that a market order should always be considered as a "taker" in a market.
This means that the market order will always take an existing limit order and will not be taken by another order.
In other words since it does not specify a price at which it will fill it can not "make" a market.

3. New Market Order, only existing market orders

If there are no limit orders in the order book and there are already pending market orders added then
the new market order will be added to the list of pending buy/sell market orders until it can match with
a limit orders.

The pending market orders are filled in the order in which they were added.

4. New Limit Order, existing market orders

If there are no limit orders and only market orders are pending on the other side of the order book and
a new limit order is added, then the limit order will not be added to the order book until all pending
market orders have been filled at the price set by the limit order.
If the entire amount of the limit order is filled then the remaining unfilled market orders will not be
removed from the pending list and since the limit order was filled it would not be added to the orde book.

*/

// Process a new market buy order and return the list of trades that matched
// This method automatically adds the remaining amount needed to be fill the market order to the list of
// pending market buy orders.
// @todo CH: Refactor this to support processing already added market orders that have not been fully filled
func (book *orderBook) processMarketBuy(order Order, trades *[]Trade) Order {
	if book.LowestAsk != 0 {
		iterator := book.SellEntries.Seek(book.LowestAsk)

		// traverse orders to find a matching one based on the sell order list
		if iterator != nil {
			for order.Amount > 0 && order.Funds > 0 {
				pricePoint := iterator.Value()
				complete := false
				// calculate how much we could afford at this price
				amountAffordable := utils.Divide(order.Funds, iterator.Key(), book.PricePrecision, book.PricePrecision, book.VolumePrecision)
				for index := 0; index < len(pricePoint.Entries); index++ {
					sellEntry := &pricePoint.Entries[index]

					amount := utils.Min(order.Amount, amountAffordable)

					// if we can fill the amount instantly and we have the necessary funds then fill the order and return trade
					// if we can fill the amount instantly, but we don't have the necessary funds then fill as much as we can afford and return the trade
					if sellEntry.Amount >= amount {
						// funds := utils.Multiply(amount, sellEntry.Price, book.VolumePrecision, book.PricePrecision, book.PricePrecision)
						*trades = append(*trades, NewTrade(book.MarketID, MarketSide_Sell, sellEntry.ID, order.ID, sellEntry.OwnerID, order.OwnerID, amount, sellEntry.Price))
						sellEntry.Amount -= amount
						order.Amount = 0
						order.SetStatus(OrderStatus_Filled)
						if sellEntry.Amount == 0 {
							book.removeSellBookEntry(sellEntry, pricePoint, index)
						}
						complete = true
						break
					}

					// if the sell order has a lower amount than what the buy order is then we fill only what we can from the sell order,
					// we complete the sell order and we move to the next order
					if sellEntry.Amount < amount {
						// @todo CH: check for overflow issues
						funds := utils.Multiply(amount, sellEntry.Price, book.VolumePrecision, book.PricePrecision, book.PricePrecision)
						*trades = append(*trades, NewTrade(book.MarketID, MarketSide_Sell, sellEntry.ID, order.ID, sellEntry.OwnerID, order.OwnerID, sellEntry.Amount, sellEntry.Price))
						order.Amount -= sellEntry.Amount
						amountAffordable -= sellEntry.Amount
						order.SetStatus(OrderStatus_PartiallyFilled)
						order.Funds -= funds
						book.removeSellBookEntry(sellEntry, pricePoint, index)
						index--
						continue
					}
				}

				if complete {
					if len(pricePoint.Entries) != 0 {
						iterator.Close()
						return order
					}
					if ok := iterator.Next(); ok {
						book.LowestAsk = iterator.Key()
						iterator.Close()
						return order
					}
					book.LowestAsk = 0
					iterator.Close()
					return order
				}

				if ok := iterator.Next(); ok {
					book.LowestAsk = iterator.Key()
				} else {
					book.LowestAsk = 0
					break
				}
			}
			iterator.Close()
		}
	}

	return order
}

// Process a new market sell order and return the list of trades that matched
// This method automatically adds the remaining amount needed to be fill the market order to the list of
// pending market sell orders.
// @todo CH: Refactor this to support processing already added market orders that have not been fully filled
func (book *orderBook) processMarketSell(order Order, trades *[]Trade) Order {
	if book.HighestBid != 0 {
		iterator := book.BuyEntries.Seek(book.HighestBid)

		// traverse orders to find a matching one based on the sell order list
		if iterator != nil {
			for order.Amount > 0 {
				pricePoint := iterator.Value()
				complete := false
				for index := 0; index < len(pricePoint.Entries); index++ {
					buyEntry := &pricePoint.Entries[index]
					// if we can fill the trade instantly then we add the trade and complete the order
					if buyEntry.Amount >= order.Amount {
						// funds := utils.Multiply(order.Amount, buyEntry.Price, book.VolumePrecision, book.PricePrecision, book.PricePrecision)
						*trades = append(*trades, NewTrade(book.MarketID, MarketSide_Buy, order.ID, buyEntry.ID, order.OwnerID, buyEntry.OwnerID, order.Amount, buyEntry.Price))
						buyEntry.Amount -= order.Amount
						order.Amount = 0
						order.SetStatus(OrderStatus_Filled)
						if buyEntry.Amount == 0 {
							book.removeBuyBookEntry(buyEntry, pricePoint, index)
						}
						complete = true
						break
					}

					// if the sell order has a lower amount than what the buy order is then we fill only what we can from the sell order,
					// we complete the sell order and we move to the next order
					if buyEntry.Amount < order.Amount {
						// funds := utils.Multiply(buyEntry.Amount, buyEntry.Price, book.VolumePrecision, book.PricePrecision, book.PricePrecision)
						*trades = append(*trades, NewTrade(book.MarketID, MarketSide_Buy, order.ID, buyEntry.ID, order.OwnerID, buyEntry.OwnerID, buyEntry.Amount, buyEntry.Price))
						order.Amount -= buyEntry.Amount
						order.SetStatus(OrderStatus_PartiallyFilled)
						book.removeBuyBookEntry(buyEntry, pricePoint, index)
						index--
						continue
					}
				}

				if complete {
					if len(pricePoint.Entries) != 0 {
						iterator.Close()
						return order
					}
					if ok := iterator.Previous(); ok {
						book.HighestBid = iterator.Key()
						iterator.Close()
						return order
					}
					book.HighestBid = 0
					iterator.Close()
					return order
				}

				if ok := iterator.Previous(); ok {
					book.HighestBid = iterator.Key()
				} else {
					book.HighestBid = 0
					break
				}
			}
			iterator.Close()
		}
	}

	return order
}