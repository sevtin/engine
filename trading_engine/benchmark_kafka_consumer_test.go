package trading_engine_test

import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/around25/products/matching-engine/net"
	"gitlab.com/around25/products/matching-engine/trading_engine"
)

func BenchmarkKafkaConsumer(benchmark *testing.B) {
	engine := trading_engine.NewTradingEngine()

	generateOrdersInKafka(benchmark.N)

	kafkaBroker := "kafka:9092"
	kafkaOrderTopic := "trading.order.btc.eth"
	kafkaOrderConsumer := "benchmark_kafka_consumer_test"
	ordersCompleted := 0
	tradesCompleted := 0

	consumer := net.NewKafkaPartitionConsumer(kafkaOrderConsumer, []string{kafkaBroker}, []string{kafkaOrderTopic})
	consumer.Start()
	defer consumer.Close()

	orders := make(chan trading_engine.Order, 10000)
	defer close(orders)

	messages := make(chan []byte, 10000)
	defer close(messages)

	done := make(chan bool)
	defer close(done)

	jsonDecode := func(messages <-chan []byte, orders chan<- trading_engine.Order) {
		for {
			msg, more := <-messages
			if !more {
				return
			}
			var order trading_engine.Order
			order.FromJSON(msg)
			orders <- order
		}
	}

	receiveMessages := func(messages chan<- []byte, n int) {
		msgChan := consumer.GetMessageChan()
		for j := 0; j < n; j++ {
			msg := <-msgChan
			// consumer.MarkOffset(msg, "")
			messages <- msg.Value
		}
	}

	startTime := time.Now().UnixNano()
	benchmark.ResetTimer()

	go receiveMessages(messages, benchmark.N)
	go jsonDecode(messages, orders)
	go func(engine trading_engine.TradingEngine, orders <-chan trading_engine.Order, n int) {
		for {
			order := <-orders
			trades := engine.Process(order)
			ordersCompleted++
			tradesCompleted += len(trades)
			if ordersCompleted >= n {
				done <- true
				return
			}
		}
	}(engine, orders, benchmark.N)

	<-done
	endTime := time.Now().UnixNano()
	timeout := (float64)(float64(time.Nanosecond) * float64(endTime-startTime) / float64(time.Second))
	fmt.Printf(
		"Total Orders: %d\n"+
			"Total Trades: %d\n"+
			"Orders/second: %f\n"+
			"Trades/second: %f\n"+
			"Pending Buy: %d\n"+
			"Lowest Ask: %d\n"+
			"Pending Sell: %d\n"+
			"Highest Bid: %d\n"+
			"Duration (seconds): %f\n\n",
		ordersCompleted,
		tradesCompleted,
		float64(ordersCompleted)/timeout,
		float64(tradesCompleted)/timeout,
		engine.GetOrderBook().GetMarket()[0].Len(),
		engine.GetOrderBook().GetLowestAsk(),
		engine.GetOrderBook().GetMarket()[1].Len(),
		engine.GetOrderBook().GetHighestBid(),
		timeout,
	)
}
