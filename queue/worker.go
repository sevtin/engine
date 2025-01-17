package queue

// import (
// 	"gitlab.com/around25/products/matching-engine/engine"
// )

// var (
// 	// MaxWorkers represents maximum number of workers to start
// 	MaxWorkers = 10
// 	// MaxQueue contains the maximum number of elements in the queue
// 	MaxQueue = 1000
// )

// // Job structure to process
// type Job struct {
// 	Order engine.Order
// }

// // Worker contains information about the worker pool and job channel
// type Worker interface {
// 	Start()
// 	Stop()
// }

// type worker struct {
// 	Engine     engine.TradingEngine
// 	WorkerPool chan chan Job
// 	JobChannel chan Job
// 	quit       chan bool
// }

// // JobQueue is a queue of channels
// var JobQueue chan Job = make(chan Job, MaxQueue)

// // NewWorker creates a new worker for the given worker pool
// func NewWorker(ngin engine.TradingEngine, workerPool chan chan Job) Worker {
// 	return &worker{
// 		Engine:     ngin,
// 		WorkerPool: workerPool,
// 		JobChannel: make(chan Job),
// 		quit:       make(chan bool),
// 	}
// }

// // Start method starts the run loop for the worker, listening for a quit channel in
// // case we need to stop it
// func (w worker) Start() {
// 	go func() {
// 		for {
// 			// register the current worker into the worker queue.
// 			w.WorkerPool <- w.JobChannel

// 			select {
// 			case job := <-w.JobChannel:
// 				// we have received a work request.
// 				trades := make([]engine.Trade, 0, 100)
// 				w.Engine.Process(job.Order, &trades)

// 			case <-w.quit:
// 				// we have received a signal to stop
// 				return
// 			}
// 		}
// 	}()
// }

// // Stop signals the worker to stop listening for work requests.
// func (w worker) Stop() {
// 	go func() {
// 		w.quit <- true
// 	}()
// }
