package main

import (
	"fmt"
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

	FajrAdhans := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "Adhan",
		Conditions: []qdb.FieldConditionEval{
			new(qdb.FieldCondition[int, qdb.Bool]).Where("IsFajr").IsEqualTo(true),
			new(qdb.FieldCondition[bool, qdb.Bool]).Where("Enabled").IsEqualTo(true),
		},
	})
	OtherAdhans := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "Adhan",
		Conditions: []qdb.FieldConditionEval{
			new(qdb.FieldCondition[bool, qdb.Bool]).Where("IsFajr").IsEqualTo(false),
			new(qdb.FieldCondition[bool, qdb.Bool]).Where("Enabled").IsEqualTo(true),
		},
	})

	var adhans []string

	switch prayerName {
	case "Fajr":
		adhans = FajrAdhans
	default:
		adhans = OtherAdhans
	}

	randomIndex := rand.Intn(len(adhans))
	adhan := adhans[randomIndex]
	file := adhan.GetField("AudioFile").PullValue(&qdb.EntityReference{}).(*qdb.EntityReference)

	if file.Raw == "" {
		qdb.Warn("[AdhanPlayer::OnNextPrayerStarted] Adhan (%v) has no audio file configured", adhan)
		return
	}

	fileDescription := adhan.GetField("AudioFile->Description").PullValue(&qdb.String{}).(*qdb.String).Raw
	qdb.Info("[AdhanPlayer::OnNextPrayerStarted] Playing adhan: %s", fileDescription)

	audioControllers := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "AudioController",
		Conditions: []qdb.FieldConditionEval{},
	})
	for _, audioController := range audioControllers {
		audioController.GetField("TextToSpeech").PushValue(&qdb.String{
			Raw: fmt.Sprintf("It is now time for %s", prayerName),
		})

		audioController.GetField("AudioFile").PushValue(file)
	}
}
