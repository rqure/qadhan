package main

import (
	"os"
	"strings"
	"time"

	qdb "github.com/rqure/qdb/src"
)

type ReminderPlayer struct {
	db qdb.IDatabase
}

func NewReminderPlayer(db qdb.IDatabase) *ReminderPlayer {
	return &ReminderPlayer{
		db: db,
	}
}

func (a *ReminderPlayer) Init() {
}

func (a *ReminderPlayer) Deinit() {
}

func (a *ReminderPlayer) DoWork() {
}

func (a *ReminderPlayer) OnNextPrayerStarted(args ...interface{}) {
	reminders := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "PrayerReminder",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewBoolCondition().Where("HasPlayed").IsEqualTo(&qdb.Bool{Raw: true}),
		},
	})

	for _, reminder := range reminders {
		reminder.GetField("HasPlayed").PushValue(&qdb.Bool{Raw: false})
	}
}

func (a *ReminderPlayer) OnNextPrayerInfo(args ...interface{}) {
	prayerName := args[0].(string)
	prayerTime := args[1].(time.Time)

	reminders := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "PrayerReminder",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewBoolCondition().Where("HasPlayed").IsEqualTo(&qdb.Bool{Raw: false}),
			qdb.NewIntCondition().Where("MinutesBefore").IsGreaterThanOrEqualTo(&qdb.Int{Raw: int64(time.Until(prayerTime).Minutes())}),
		},
	})

	for _, reminder := range reminders {
		textToSpeech := reminder.GetField("TextToSpeech").PullString()
		if textToSpeech == "" {
			continue
		}

		textToSpeech = strings.ReplaceAll(textToSpeech, "{Prayer}", prayerName)

		qdb.Info("[ReminderPlayer::OnNextPrayerInfo] Playing reminder: %s", reminder)

		alertControllers := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
			EntityType: "AlertController",
			Conditions: []qdb.FieldConditionEval{},
		})

		for _, alertController := range alertControllers {
			// alertController.GetField("TextToSpeech").PushValue(&qdb.String{Raw: textToSpeech})
			a.db.Write([]*qdb.DatabaseRequest{
				{
					Id:    alertController.GetId(),
					Field: "ApplicationName",
					Value: qdb.NewStringValue(os.Getenv("QDB_APP_NAME")),
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

		reminder.GetField("HasPlayed").PushBool(true)
	}
}
