package main

import (
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

}

func (a *ReminderPlayer) OnNextPrayerInfo(args ...interface{}) {
	prayerName := args[0].(string)
	prayerTime := args[1].(time.Time)

	reminders := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "PrayerReminder",
		Conditions: []qdb.FieldConditionEval{
			qdb.NewBoolCondition().Where("HasPlayed").IsEqualTo(&qdb.Bool{Raw: false}),
			qdb.NewIntCondition().Where("MinutesBefore").IsLessThanOrEqualTo(&qdb.Int{Raw: int64(time.Until(prayerTime).Minutes())}),
		},
	})

	for _, reminder := range reminders {
		textToSpeech := reminder.GetField("TextToSpeech").PullValue(&qdb.String{}).(*qdb.String).Raw
		if textToSpeech == "" {
			continue
		}
		textToSpeech = strings.ReplaceAll(textToSpeech, "{Prayer}", prayerName)

		qdb.Info("[ReminderPlayer::OnNextPrayerInfo] Playing reminder: %s", reminder)

		audioControllers := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
			EntityType: "AudioController",
			Conditions: []qdb.FieldConditionEval{},
		})

		for _, audioController := range audioControllers {
			audioController.GetField("TextToSpeech").PushValue(&qdb.String{Raw: textToSpeech})
		}

		reminder.GetField("HasPlayed").PushValue(&qdb.Bool{Raw: true})
	}
}
