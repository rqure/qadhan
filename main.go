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
