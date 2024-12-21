package main

import (
	"os"

	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/app/workers"
	"github.com/rqure/qlib/pkg/data/store"
)

func getDatabaseAddress() string {
	addr := os.Getenv("Q_ADDR")
	if addr == "" {
		addr = "ws://webgateway:20000/ws"
	}

	return addr
}

func main() {
	db := store.NewWeb(store.WebConfig{
		Address: getDatabaseAddress(),
	})

	storeWorker := workers.NewStore(db)
	leadershipWorker := workers.NewLeadership(db)
	adhanPlayer := NewAdhanPlayer(db)
	prayerDetailsProvider := NewPrayerDetailsProvider(db)
	reminderPlayer := NewReminderPlayer(db)

	schemaValidator := leadershipWorker.GetEntityFieldValidator()

	schemaValidator.RegisterEntityFields("Root", "SchemaUpdateTrigger")
	schemaValidator.RegisterEntityFields("AdhanController", "Country", "City", "BaseURL")
	schemaValidator.RegisterEntityFields("RingBuffer", "Capacity", "CurrentIndex", "EndIndex")
	schemaValidator.RegisterEntityFields("MP3File", "Content", "Description")
	schemaValidator.RegisterEntityFields("Adhan", "AudioFile", "Enabled", "IsFajr")
	schemaValidator.RegisterEntityFields("Prayer", "PrayerName", "StartTime")
	schemaValidator.RegisterEntityFields("PrayerReminder", "MinutesBefore", "TextToSpeech", "HasPlayed")

	storeWorker.Connected.Connect(leadershipWorker.OnStoreConnected)
	storeWorker.Disconnected.Connect(leadershipWorker.OnStoreDisconnected)

	leadershipWorker.BecameLeader().Connect(prayerDetailsProvider.OnBecameLeader)
	leadershipWorker.LosingLeadership().Connect(prayerDetailsProvider.OnLostLeadership)

	prayerDetailsProvider.Signals.NextPrayerStarted.Connect(adhanPlayer.OnNextPrayerStarted)
	prayerDetailsProvider.Signals.NextPrayerStarted.Connect(reminderPlayer.OnNextPrayerStarted)
	prayerDetailsProvider.Signals.NextPrayerInfo.Connect(reminderPlayer.OnNextPrayerInfo)

	// Create a new application configuration
	config := qdb.ApplicationConfig{
		Name: "adhan",
		Workers: []qdb.IWorker{
			storeWorker,
			leadershipWorker,
			prayerDetailsProvider,
			adhanPlayer,
			reminderPlayer,
		},
	}

	app := app.NewApplication(config)

	app.Execute()
}
