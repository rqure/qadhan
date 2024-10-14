package main

import (
	"os"

	qdb "github.com/rqure/qdb/src"
)

func getDatabaseAddress() string {
	addr := os.Getenv("QDB_ADDR")
	if addr == "" {
		addr = "redis:6379"
	}

	return addr
}

func main() {
	db := qdb.NewRedisDatabase(qdb.RedisDatabaseConfig{
		Address: getDatabaseAddress(),
	})

	dbWorker := qdb.NewDatabaseWorker(db)
	leaderElectionWorker := qdb.NewLeaderElectionWorker(db)
	adhanPlayer := NewAdhanPlayer(db)
	prayerDetailsProvider := NewPrayerDetailsProvider(db)
	reminderPlayer := NewReminderPlayer(db)

	schemaValidator := qdb.NewSchemaValidator(db)

	schemaValidator.AddEntity("Root", "SchemaUpdateTrigger")
	schemaValidator.AddEntity("AdhanController", "Country", "City", "BaseURL")
	schemaValidator.AddEntity("RingBuffer", "Capacity", "CurrentIndex", "EndIndex")
	schemaValidator.AddEntity("MP3File", "Content", "Description")
	schemaValidator.AddEntity("Adhan", "AudioFile", "Enabled", "IsFajr")
	schemaValidator.AddEntity("Prayer", "PrayerName", "StartTime")
	schemaValidator.AddEntity("PrayerReminder", "MinutesBefore", "TextToSpeech", "HasPlayed")

	dbWorker.Signals.SchemaUpdated.Connect(qdb.Slot(schemaValidator.ValidationRequired))
	dbWorker.Signals.Connected.Connect(qdb.Slot(schemaValidator.ValidationRequired))
	leaderElectionWorker.AddAvailabilityCriteria(func() bool {
		return dbWorker.IsConnected() && schemaValidator.IsValid()
	})

	dbWorker.Signals.Connected.Connect(qdb.Slot(leaderElectionWorker.OnDatabaseConnected))
	dbWorker.Signals.Disconnected.Connect(qdb.Slot(leaderElectionWorker.OnDatabaseDisconnected))

	leaderElectionWorker.Signals.BecameLeader.Connect(qdb.Slot(prayerDetailsProvider.OnBecameLeader))
	leaderElectionWorker.Signals.LosingLeadership.Connect(qdb.Slot(prayerDetailsProvider.OnLostLeadership))

	prayerDetailsProvider.Signals.NextPrayerStarted.Connect(qdb.SlotWithArgs(adhanPlayer.OnNextPrayerStarted))
	prayerDetailsProvider.Signals.NextPrayerStarted.Connect(qdb.SlotWithArgs(reminderPlayer.OnNextPrayerStarted))
	prayerDetailsProvider.Signals.NextPrayerInfo.Connect(qdb.SlotWithArgs(reminderPlayer.OnNextPrayerInfo))

	// Create a new application configuration
	config := qdb.ApplicationConfig{
		Name: "adhan",
		Workers: []qdb.IWorker{
			dbWorker,
			leaderElectionWorker,
			prayerDetailsProvider,
			adhanPlayer,
			reminderPlayer,
		},
	}

	// Create a new application
	app := qdb.NewApplication(config)

	// Execute the application
	app.Execute()
}
