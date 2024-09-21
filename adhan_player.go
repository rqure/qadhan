package main

import (
	"math/rand"

	qdb "github.com/rqure/qdb/src"
)

type AdhanPlayer struct {
	db qdb.IDatabase
}

func NewAdhanPlayer(db qdb.IDatabase) *AdhanPlayer {
	return &AdhanPlayer{
		db: db,
	}
}

func (a *AdhanPlayer) Init() {
}

func (a *AdhanPlayer) Deinit() {
}

func (a *AdhanPlayer) DoWork() {
}

func (a *AdhanPlayer) OnNextPrayerStarted(args ...interface{}) {
	prayerName := args[0].(string)

	adhans := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "Adhan",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewBoolCondition().Where("IsFajr").IsEqualTo(&qdb.Bool{Raw: false}),
			qdb.NewBoolCondition().Where("Enabled").IsEqualTo(&qdb.Bool{Raw: true}),
		},
	})

	if prayerName == "Fajr" {
		adhans = qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
			EntityType: "Adhan",
			Conditions: []qdb.FieldConditionEval{
				qdb.NewBoolCondition().Where("IsFajr").IsEqualTo(&qdb.Bool{Raw: true}),
				qdb.NewBoolCondition().Where("Enabled").IsEqualTo(&qdb.Bool{Raw: true}),
			},
		})
	}

	randomIndex := rand.Intn(len(adhans))
	adhan := adhans[randomIndex]
	fileReference := adhan.GetField("AudioFile").PullEntityReference()

	if fileReference == "" {
		qdb.Warn("[AdhanPlayer::OnNextPrayerStarted] Adhan (%v) has no audio file configured", adhan)
		return
	}

	fileDescription := adhan.GetField("AudioFile->Description").PullString()
	qdb.Info("[AdhanPlayer::OnNextPrayerStarted] Playing adhan: %s", fileDescription)

	audioControllers := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "AudioController",
		Conditions: []qdb.FieldConditionEval{},
	})

	for _, audioController := range audioControllers {
		audioController.GetField("AudioFile").PushEntityReference(fileReference)
	}
}
