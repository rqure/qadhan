package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	qmq "github.com/rqure/qmq/src"
)

type EngineProcessor struct {
	PrayerInfoProvider PrayerInfoProvider
	reminderFlag       atomic.Bool
}

func (e *EngineProcessor) Process(p qmq.EngineComponentProvider) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	e.PrayerInfoProvider = NewAlAdhanPrayerInfoProvider(&EnvironmentLocationProvider{}, p.WithLogger())

	AdhanFileSelector := &DefaultAdhanFileSelector{
		FajrAdhanFiles: []string{"fajr-1.mp3"},
		OtherAdhanFiles: []string{
			"adhan-0.mp3",
			"adhan-1.mp3",
			"adhan-2.mp3",
			"adhan-3.mp3",
			"adhan-5.mp3",
			"adhan-wahhab.mp3"},
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			select {
			case consumable := <-p.WithConsumer("prayer:time:queue").Pop():
				next_prayer := consumable.Data().(*qmq.Prayer)
				if time.Now().After(next_prayer.Time.AsTime()) {
					p.WithLogger().Advise(fmt.Sprintf("It is now time for: %s", next_prayer.Name))

					p.WithProducer("audio-player:tts:exchange").Push(&qmq.TextToSpeechRequest{
						Text: fmt.Sprintf("It is now time for %s", next_prayer.Name),
					})

					p.WithProducer("audio-player:file:exchange").Push(&qmq.AudioRequest{
						Filename: AdhanFileSelector.Select(next_prayer.Name),
					})

					e.reminderFlag.Store(false)

					consumable.Ack()
				} else {
					consumable.Nack()

					if e.reminderFlag.CompareAndSwap(false, true) {
						for _, reminderTimeMin := range []int{10, 20, 30, 60} {
							go func(reminderTimeMin int, prayerName string) {
								reminderTime := next_prayer.Time.AsTime().Add(-time.Duration(reminderTimeMin) * time.Minute)
								p.WithLogger().Advise(fmt.Sprintf("Scheduling reminder for: %s", reminderTime.String()))
								<-time.After(time.Until(reminderTime))

								p.WithLogger().Advise(fmt.Sprintf("It is almost time for: %s", prayerName))

								if reminderTimeMin == 60 {
									p.WithProducer("audio-player:tts:exchange").Push(&qmq.TextToSpeechRequest{
										Text: fmt.Sprintf("Reminder: %s starts in 1 hour", prayerName),
									})
								} else {
									p.WithProducer("audio-player:tts:exchange").Push(&qmq.TextToSpeechRequest{
										Text: fmt.Sprintf("Reminder: %s starts in %d minutes", prayerName, reminderTimeMin),
									})
								}
							}(reminderTimeMin, next_prayer.Name)
						}
					}
				}
			case <-time.After(1 * time.Second):
				p.WithLogger().Advise("Querying next prayer times")
				prayers := e.PrayerInfoProvider.GetNextPrayers()
				for _, prayer := range prayers {
					p.WithLogger().Debug(fmt.Sprintf("Found prayer '%s' starting at '%s'", prayer.Name, prayer.Time.AsTime().String()))
					p.WithProducer("prayer:time:queue").Push(prayer)
				}
			}
		}
	}
}
