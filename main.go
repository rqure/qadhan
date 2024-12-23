package main

import (
	"os"

	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/app/workers"
	"github.com/rqure/qlib/pkg/data/store"
)

func getStoreAddress() string {
	addr := os.Getenv("Q_ADDR")
	if addr == "" {
		addr = "ws://webgateway:20000/ws"
	}

	return addr
}

func main() {
	s := store.NewWeb(store.WebConfig{
		Address: getStoreAddress(),
	})

	storeWorker := workers.NewStore(s)
	leadershipWorker := workers.NewLeadership(s)
	adhanPlayer := NewAdhanPlayer(s)
	prayerDetailsProvider := NewPrayerDetailsProvider(s)
	reminderPlayer := NewReminderPlayer(s)

	schemaValidator := leadershipWorker.GetEntityFieldValidator()

	schemaValidator.RegisterEntityFields("Root", "SchemaUpdateTrigger")
	schemaValidator.RegisterEntityFields("AdhanController", "Country", "City", "BaseURL")
	schemaValidator.RegisterEntityFields("RingBuffer", "Capacity", "CurrentIndex", "EndIndex")
	schemaValidator.RegisterEntityFields("MP3File", "Content", "Description")
	schemaValidator.RegisterEntityFields("Adhan", "AudioFile", "Enabled", "IsFajr")
	schemaValidator.RegisterEntityFields("Prayer", "PrayerName", "StartTime")
	schemaValidator.RegisterEntityFields("PrayerReminder", "MinutesBefore", "TextToSpeech", "HasPlayed", "Prayer", "Language")

	storeWorker.Connected.Connect(leadershipWorker.OnStoreConnected)
	storeWorker.Disconnected.Connect(leadershipWorker.OnStoreDisconnected)

	leadershipWorker.BecameLeader().Connect(prayerDetailsProvider.OnBecameLeader)
	leadershipWorker.LosingLeadership().Connect(prayerDetailsProvider.OnLostLeadership)

	prayerDetailsProvider.NextPrayerStarted.Connect(adhanPlayer.OnNextPrayerStarted)
	prayerDetailsProvider.NextPrayerStarted.Connect(reminderPlayer.OnNextPrayerStarted)
	prayerDetailsProvider.NextPrayerInfo.Connect(reminderPlayer.OnNextPrayerInfo)

	a := app.NewApplication("adhan")
	a.AddWorker(storeWorker)
	a.AddWorker(leadershipWorker)
	a.AddWorker(prayerDetailsProvider)
	a.AddWorker(adhanPlayer)
	a.AddWorker(reminderPlayer)
	a.Execute()
}
