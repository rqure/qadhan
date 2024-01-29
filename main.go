package main

import (
	"os"
	"os/signal"
	"strconv"
	"time"

	qmq "github.com/rqure/qmq/src"
)

func main() {
	app := qmq.NewQMQApplication("prayer")
	app.Initialize()
	defer app.Deinitialize()

	app.AddProducer("prayer:time-exchange").Initialize(500)
	app.AddProducer("prayer:adhan-exchange").Initialize(1)
	app.AddConsumer("prayer:time-queue").Initialize()

	tickRateMs, err := strconv.Atoi(os.Getenv("TICK_RATE_MS"))
	if err != nil {
		tickRateMs = 100
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	ticker := time.NewTicker(time.Duration(tickRateMs) * time.Millisecond)
	for {
		select {
		case <-sigint:
			app.Logger().Advise("SIGINT received")
			return
		case <-ticker.C:

		}
	}
}
