package main

import (
	"context"
	"os"
	"strings"
	"time"

	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/data"
	"github.com/rqure/qlib/pkg/data/query"
	"github.com/rqure/qlib/pkg/log"
)

type ReminderPlayer struct {
	store data.Store
}

func NewReminderPlayer(store data.Store) *ReminderPlayer {
	return &ReminderPlayer{
		db: db,
	}
}

func (a *ReminderPlayer) Init(context.Context, app.Handle) {
}

func (a *ReminderPlayer) Deinit(context.Context) {
}

func (a *ReminderPlayer) DoWork(context.Context) {
}

func (a *ReminderPlayer) OnNextPrayerStarted(args ...interface{}) {
	reminders := query.New(a.db).Find(qdb.SearchCriteria{
		EntityType: "PrayerReminder",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewBoolCondition().Where("HasPlayed").IsEqualTo(&qdb.Bool{Raw: true}),
		},
	})

	for _, reminder := range reminders {
		reminder.GetField("HasPlayed").WriteValue(ctx, &qdb.Bool{Raw: false})
	}
}

func (a *ReminderPlayer) OnNextPrayerInfo(args ...interface{}) {
	prayerName := args[0].(string)
	prayerTime := args[1].(time.Time)

	reminders := query.New(a.db).Find(qdb.SearchCriteria{
		EntityType: "PrayerReminder",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewBoolCondition().Where("HasPlayed").IsEqualTo(&qdb.Bool{Raw: false}),
			qdb.NewIntCondition().Where("MinutesBefore").IsGreaterThanOrEqualTo(&qdb.Int{Raw: int64(time.Until(prayerTime).Minutes())}),
		},
	})

	for _, reminder := range reminders {
		textToSpeech := reminder.GetField("TextToSpeech").ReadString(ctx)
		if textToSpeech == "" {
			continue
		}

		textToSpeech = strings.ReplaceAll(textToSpeech, "{Prayer}", prayerName)

		log.Info("Playing reminder: %s", reminder)

		alertControllers := query.New(a.db).Find(qdb.SearchCriteria{
			EntityType: "AlertController",
			Conditions: []qdb.FieldConditionEval{},
		})

		for _, alertController := range alertControllers {
			a.db.Write([]*qdb.DatabaseRequest{
				{
					Id:    alertController.GetId(),
					Field: "ApplicationName",
					Value: qdb.NewStringValue(qdb.GetApplicationName()),
				},
				{
					Id:    alertController.GetId(),
					Field: "Description",
					Value: qdb.NewStringValue(textToSpeech),
				},
				{
					Id:    alertController.GetId(),
					Field: "TTSAlert",
					Value: qdb.NewBoolValue(strings.Contains(os.Getenv("ALERTS"), "TTS")),
				},
				{
					Id:    alertController.GetId(),
					Field: "EmailAlert",
					Value: qdb.NewBoolValue(strings.Contains(os.Getenv("ALERTS"), "EMAIL")),
				},
				{
					Id:    alertController.GetId(),
					Field: "SendTrigger",
					Value: qdb.NewIntValue(0),
				},
			})
		}

		reminder.GetField("HasPlayed").WriteBool(ctx, true)
	}
}
