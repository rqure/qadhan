package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	qdb "github.com/rqure/qdb/src"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PrayerDetails struct {
	Name *qdb.String
	Time *qdb.Timestamp
}

type PrayerDetailsProviderSignals struct {
	NextPrayerStarted qdb.Signal
	NextPrayerInfo    qdb.Signal
}

type PrayerDetailsProvider struct {
	db           qdb.IDatabase
	isLeader     bool
	tickInterval time.Duration
	lastTick     time.Time
	Signals      PrayerDetailsProviderSignals
}

func NewPrayerDetailsProvider(db qdb.IDatabase) *PrayerDetailsProvider {
	return &PrayerDetailsProvider{
		db:           db,
		tickInterval: 10 * time.Second,
	}
}

func (a *PrayerDetailsProvider) OnBecameLeader() {
	a.isLeader = true
}

func (a *PrayerDetailsProvider) OnLostLeadership() {
	a.isLeader = false
}

func (a *PrayerDetailsProvider) Init() {

}

func (a *PrayerDetailsProvider) Deinit() {

}

func (a *PrayerDetailsProvider) DoWork() {
	if !a.isLeader {
		return
	}

	if time.Since(a.lastTick) < a.tickInterval {
		return
	}

	a.lastTick = time.Now()
	controllers := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "AdhanController",
		Conditions: []qdb.FieldConditionEval{},
	})

	for _, controller := range controllers {
		capacity := controller.GetField("Prayer Buffer->Capacity").PullInt()
		currentIndex := controller.GetField("Prayer Buffer->CurrentIndex").PullInt()
		endIndex := controller.GetField("Prayer Buffer->EndIndex").PullInt()

		if currentIndex == endIndex {
			qdb.Info("[PrayerDetailsProvider::DoWork] Prayer buffer is empty, querying next prayers")
			country := controller.GetField("Country").PullValue(&qdb.String{}).(*qdb.String)
			city := controller.GetField("City").PullValue(&qdb.String{}).(*qdb.String)
			baseUrl := controller.GetField("BaseURL").PullValue(&qdb.String{}).(*qdb.String)
			prayerDetails := a.QueryNextPrayers(baseUrl.Raw, country.Raw, city.Raw)

			for _, prayer := range prayerDetails {
				if (endIndex+1)%capacity == currentIndex {
					qdb.Warn("[PrayerDetailsProvider::DoWork] Prayer buffer is full")
					break
				}

				controller.GetField(fmt.Sprintf("Prayer Buffer->%d->PrayerName", endIndex)).PushValue(prayer.Name)
				controller.GetField(fmt.Sprintf("Prayer Buffer->%d->StartTime", endIndex)).PushValue(prayer.Time)

				qdb.Info("[PrayerDetailsProvider::DoWork] Added prayer '%s' (startTime=%s) to the buffer (endIndex=%d)", prayer.Name.Raw, prayer.Time.Raw.AsTime().Format(time.RFC3339), endIndex)

				endIndex = (endIndex + 1) % capacity
				controller.GetField("Prayer Buffer->EndIndex").PushInt(endIndex)
			}
		} else {
			nextPrayer := &PrayerDetails{
				Name: &qdb.String{},
				Time: &qdb.Timestamp{},
			}

			controller.GetField(fmt.Sprintf("Prayer Buffer->%d->PrayerName", currentIndex)).PullValue(nextPrayer.Name)
			controller.GetField(fmt.Sprintf("Prayer Buffer->%d->StartTime", currentIndex)).PullValue(nextPrayer.Time)

			if time.Now().After(nextPrayer.Time.Raw.AsTime()) {
				qdb.Info("[PrayerDetailsProvider::DoWork] Next prayer '%s' has started", nextPrayer.Name.Raw)
				currentIndex = (currentIndex + 1) % capacity
				controller.GetField("Prayer Buffer->CurrentIndex").PushInt(currentIndex)
				a.Signals.NextPrayerStarted.Emit(nextPrayer.Name.Raw)
			} else {
				a.Signals.NextPrayerInfo.Emit(nextPrayer.Name.Raw, nextPrayer.Time.Raw.AsTime())
			}
		}
	}
}

func (a *PrayerDetailsProvider) QueryNextPrayers(baseUrl, country, city string) []*PrayerDetails {
	if baseUrl == "" || country == "" || city == "" {
		qdb.Error("[PrayerDetailsProvider::QueryNextPrayers] Query options are invalid")
		return []*PrayerDetails{}
	}

	result := []*PrayerDetails{}
	params := url.Values{}
	params.Add("city", city)
	params.Add("country", country)

	resp, err := http.Get(fmt.Sprintf("%s?%s", baseUrl, params.Encode()))
	if err != nil {
		qdb.Error("[PrayerDetailsProvider::QueryNextPrayers] Failed to fetch prayer times: %v", err)
		return result
	}
	defer resp.Body.Close()

	var response struct {
		Code   int    `json:"code"`
		Status string `json:"status"`
		Data   []struct {
			Timings struct {
				Fajr    string `json:"Fajr"`
				Dhuhr   string `json:"Dhuhr"`
				Asr     string `json:"Asr"`
				Maghrib string `json:"Maghrib"`
				Isha    string `json:"Isha"`
				// Add other prayer times as needed
			} `json:"timings"`
			Date struct {
				Readable  string `json:"readable"`
				Timestamp string `json:"timestamp"`
				// Extend this struct if you need more fields from the 'date' object
			} `json:"date"`
			Meta struct {
				Timezone string `json:"timezone"`
			} `json:"meta"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		qdb.Error("[PrayerDetailsProvider::QueryNextPrayers] Failed to decode prayer times: %v", err)
		return result
	}

	for _, day := range response.Data {
		for prayer, timeStr := range map[string]string{
			"Fajr":    day.Timings.Fajr,
			"Dhuhr":   day.Timings.Dhuhr,
			"Asr":     day.Timings.Asr,
			"Maghrib": day.Timings.Maghrib,
			"Isha":    day.Timings.Isha,
		} {
			loc, err := time.LoadLocation(day.Meta.Timezone)
			if err != nil {
				qdb.Warn("[PrayerDetailsProvider::QueryNextPrayers] Failed to load timezone: %v", err)
				loc = time.Local
			}

			timeParsed, err := time.ParseInLocation("02 Jan 2006 15:04 (MST)", fmt.Sprintf("%s %s", day.Date.Readable, timeStr), loc)
			if err != nil {
				qdb.Warn("[PrayerDetailsProvider::QueryNextPrayers] Failed to parse time: %v", err)
				continue
			}

			if time.Now().Before(timeParsed) {
				result = append(result, &PrayerDetails{
					Name: &qdb.String{Raw: prayer},
					Time: &qdb.Timestamp{Raw: timestamppb.New(timeParsed)},
				})
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Time.Raw.AsTime().Before(result[j].Time.Raw.AsTime())
	})

	return result
}
