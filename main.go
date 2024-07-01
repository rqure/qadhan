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

	schemaValidator := qdb.NewSchemaValidator(db)

	schemaValidator.AddEntity("Root", "SchemaUpdateTrigger")
	schemaValidator.AddEntity("AdhanController", "Country", "City", "BaseURL")
	schemaValidator.AddEntity("RingBuffer", "Capacity", "CurrentIndex", "EndIndex")
	schemaValidator.AddEntity("MP3File", "Content", "Description")
	schemaValidator.AddEntity("Adhan", "AudioFile", "Enabled", "IsFajr")
	schemaValidator.AddEntity("Prayer", "PrayerName", "StartTime")

	dbWorker.Signals.SchemaUpdated.Connect(qdb.Slot(schemaValidator.OnSchemaUpdated))
	leaderElectionWorker.AddAvailabilityCriteria(func() bool {
		return schemaValidator.IsValid()
	})

	dbWorker.Signals.Connected.Connect(qdb.Slot(leaderElectionWorker.OnDatabaseConnected))
	dbWorker.Signals.Disconnected.Connect(qdb.Slot(leaderElectionWorker.OnDatabaseDisconnected))

	// leaderElectionWorker.Signals.BecameLeader.Connect(qdb.Slot(audioFileRequestHandler.OnBecameLeader))
	// leaderElectionWorker.Signals.BecameFollower.Connect(qdb.Slot(audioFileRequestHandler.OnLostLeadership))
	// leaderElectionWorker.Signals.BecameUnavailable.Connect(qdb.Slot(audioFileRequestHandler.OnLostLeadership))

	// leaderElectionWorker.Signals.BecameLeader.Connect(qdb.Slot(textToSpeechRequestHandler.OnBecameLeader))
	// leaderElectionWorker.Signals.BecameFollower.Connect(qdb.Slot(textToSpeechRequestHandler.OnLostLeadership))
	// leaderElectionWorker.Signals.BecameUnavailable.Connect(qdb.Slot(textToSpeechRequestHandler.OnLostLeadership))

	// Create a new application configuration
	config := qdb.ApplicationConfig{
		Name: "adhan",
		Workers: []qdb.IWorker{
			dbWorker,
			leaderElectionWorker,
		},
	}

	// Create a new application
	app := qdb.NewApplication(config)

	// Execute the application
	app.Execute()
}
