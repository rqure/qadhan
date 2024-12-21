package main

import (
	"context"
	"math/rand"

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
		store: store,
	}
}

func (a *AdhanPlayer) Init(context.Context, app.Handle) {
}

func (a *AdhanPlayer) Deinit(context.Context) {
}

func (a *AdhanPlayer) DoWork(context.Context) {
}

func (a *AdhanPlayer) OnNextPrayerStarted(ctx context.Context, args ...interface{}) {
	prayerName := args[0].(string)

	adhans := query.New(a.store).
		ForType("Adhan").
		Where("IsFajr").Equals(false).
		Where("Enabled").Equals(true).
		Execute(ctx)

	if prayerName == "Fajr" {
		adhans = query.New(a.store).
			ForType("Adhan").
			Where("IsFajr").Equals(true).
			Where("Enabled").Equals(true).
			Execute(ctx)
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

	audioControllers := query.New(a.store).ForType("AudioController").Execute(ctx)

	for _, audioController := range audioControllers {
		audioController.GetField("AudioFile").WriteEntityReference(ctx, fileReference)
	}
}
