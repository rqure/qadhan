package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	qmq "github.com/rqure/qmq/src"
)

type EngineProcessor struct {
	PrayerInfoProvider PrayerInfoProvider
}

func (e *EngineProcessor) Process(p qmq.EngineComponentProvider) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	e.PrayerInfoProvider = NewAlAdhanPrayerInfoProvider(&EnvironmentLocationProvider{}, p.WithLogger())

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			select {
			case consumable := <-p.WithConsumer("prayer:time:queue").Pop():
				p.WithLogger().Trace("popped")
				next_prayer := consumable.Data().(*qmq.Prayer)
				if time.Now().After(next_prayer.Time.AsTime()) {
					p.WithLogger().Advise(fmt.Sprintf("It is now time for: %s", next_prayer.Name))

					audioFiles := []string{
						"adhan-0.mp3",
						"adhan-1.mp3",
						"adhan-2.mp3",
						"adhan-3.mp3",
						"adhan-5.mp3",
						"adhan-wahhab.mp3"}

					if next_prayer.Name == "Fajr" {
						audioFiles = []string{"fajr-1.mp3"}
					}

					randomIndex := rand.Intn(len(audioFiles))
					audioFile := audioFiles[randomIndex]

					p.WithProducer("audio-player:file:exchange").Push(&qmq.AudioRequest{
						Filename: audioFile,
					})

					consumable.Ack()
				} else {
					consumable.Nack()
				}
			case <-time.After(1 * time.Second):
				p.WithLogger().Advise("Querying next prayer times")
				// prayers := e.PrayerInfoProvider.GetNextPrayers()
				// for _, prayer := range prayers {
				// 	p.WithLogger().Debug(fmt.Sprintf("Found prayer '%s' starting at '%s'", prayer.Name, prayer.Time.AsTime().String()))
				// 	p.WithProducer("prayer:time:queue").Push(prayer)
				// }
			}
		}
	}
}
