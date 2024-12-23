package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/data"
	"github.com/rqure/qlib/pkg/data/binding"
	"github.com/rqure/qlib/pkg/data/query"
	"github.com/rqure/qlib/pkg/log"
)

type ReminderPlayer struct {
	store data.Store
}

func NewReminderPlayer(store data.Store) *ReminderPlayer {
	return &ReminderPlayer{
		store: store,
	}
}

func (a *ReminderPlayer) Init(context.Context, app.Handle) {
}

func (a *ReminderPlayer) Deinit(context.Context) {
}

func (a *ReminderPlayer) DoWork(context.Context) {
}

func (a *ReminderPlayer) OnNextPrayerStarted(ctx context.Context, args ...interface{}) {
	reminders := query.New(a.store).
		ForType("PrayerReminder").
		Where("HasPlayed").Equals(true).
		Execute(ctx)

	for _, reminder := range reminders {
		reminder.GetField("HasPlayed").WriteBool(ctx, false)
	}
}

func (a *ReminderPlayer) OnNextPrayerInfo(ctx context.Context, args ...interface{}) {
	prayerName := args[0].(string)
	prayerTime := args[1].(time.Time)

	reminders := query.New(a.store).
		ForType("PrayerReminder").
		Where("Prayer").Equals(prayerName).
		Where("HasPlayed").Equals(false).
		Where("MinutesBefore").GreaterThanOrEqual(int64(time.Until(prayerTime).Minutes())).
		Execute(ctx)

	for _, reminder := range reminders {
		textToSpeech := reminder.GetField("TextToSpeech").ReadString(ctx)
		language := reminder.GetField("Language").ReadString(ctx)
		if textToSpeech == "" {
			continue
		}

		log.Info("Playing reminder: %s", reminder)

		multi := binding.NewMulti(a.store)
		alertControllers := query.New(multi).
			ForType("AlertController").
			Execute(ctx)

		for _, alertController := range alertControllers {
			alertController.GetField("ApplicationName").WriteString(ctx, app.GetName())
			alertController.GetField("Description").WriteString(ctx, textToSpeech)
			alertController.GetField("TTSLanguage").WriteString(ctx, language)
			alertController.GetField("TTSAlert").WriteBool(ctx, strings.Contains(os.Getenv("ALERTS"), "TTS"))
			alertController.GetField("EmailAlert").WriteBool(ctx, strings.Contains(os.Getenv("ALERTS"), "EMAIL"))
			alertController.GetField("SendTrigger").WriteInt(ctx)
		}

		multi.Commit(ctx)

		reminder.GetField("HasPlayed").WriteBool(ctx, true)
	}
}
