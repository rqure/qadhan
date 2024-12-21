package main

import (
	"context"
	"math/rand"

	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/data"
	"github.com/rqure/qlib/pkg/data/query"
	"github.com/rqure/qlib/pkg/log"
)

type AdhanPlayer struct {
	store data.Store
}

func NewAdhanPlayer(store data.Store) *AdhanPlayer {
	return &AdhanPlayer{
		db: db,
	}
}

func (a *AdhanPlayer) Init(context.Context, app.Handle) {
}

func (a *AdhanPlayer) Deinit(context.Context) {
}

func (a *AdhanPlayer) DoWork(context.Context) {
}

func (a *AdhanPlayer) OnNextPrayerStarted(args ...interface{}) {
	prayerName := args[0].(string)

	adhans := query.New(a.db).Find(qdb.SearchCriteria{
		EntityType: "Adhan",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewBoolCondition().Where("IsFajr").IsEqualTo(&qdb.Bool{Raw: false}),
			qdb.NewBoolCondition().Where("Enabled").IsEqualTo(&qdb.Bool{Raw: true}),
		},
	})

	if prayerName == "Fajr" {
		adhans = query.New(a.db).Find(qdb.SearchCriteria{
			EntityType: "Adhan",
			Conditions: []qdb.FieldConditionEval{
				qdb.NewBoolCondition().Where("IsFajr").IsEqualTo(&qdb.Bool{Raw: true}),
				qdb.NewBoolCondition().Where("Enabled").IsEqualTo(&qdb.Bool{Raw: true}),
			},
		})
	}

	randomIndex := rand.Intn(len(adhans))
	adhan := adhans[randomIndex]
	fileReference := adhan.GetField("AudioFile").ReadEntityReference(ctx)

	if fileReference == "" {
		log.Warn("Adhan (%v) has no audio file configured", adhan)
		return
	}

	fileDescription := adhan.GetField("AudioFile->Description").ReadString(ctx)
	log.Info("Playing adhan: %s", fileDescription)

	audioControllers := query.New(a.db).Find(qdb.SearchCriteria{
		EntityType: "AudioController",
		Conditions: []qdb.FieldConditionEval{},
	})

	for _, audioController := range audioControllers {
		audioController.GetField("AudioFile").WriteEntityReference(ctx, fileReference)
	}
}
