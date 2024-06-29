package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	qdb "github.com/rqure/qdb/src"
)

type AudioController struct {
	Country string
	City    string
	BaseURL string
}

type PrayerDetails struct {
	Name string
	Time time.Time
}

type PrayerDetailsProvider struct {
	db       qdb.IDatabase
	isLeader bool
}

func NewPrayerDetailsProvider(db qdb.IDatabase) *PrayerDetailsProvider {
	return &PrayerDetailsProvider{db: db}
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
}

func (a *PrayerDetailsProvider) GetAudioController() *AudioController {
	controllers := qdb.NewEntityFinder(a.db).Find(qdb.SearchCriteria{
		EntityType: "AdhanController",
		Conditions: []qdb.FieldConditionEval{},
	})

	for _, controller := range controllers {
		country := controller.GetField("Country").PullValue(&qdb.String{}).(*qdb.String).Raw
		city := controller.GetField("City").PullValue(&qdb.String{}).(*qdb.String).Raw
		baseUrl := controller.GetField("BaseURL").PullValue(&qdb.String{}).(*qdb.String).Raw
		return &AudioController{
			Country: country,
			City:    city,
			BaseURL: baseUrl,
		}
	}

	qdb.Error("[PrayerDetailsProvider::GetAudioController] No AudioController entity exists in the database")
	return nil
}

func (a *PrayerDetailsProvider) QueryNextPrayers() []*PrayerDetails {
	opts := a.GetAudioController()
	if opts == nil || opts.City == "" || opts.Country == "" || opts.BaseURL == "" {
		qdb.Error("[PrayerDetailsProvider::QueryNextPrayers] Query options are invalid (%v)", opts)
		return []*PrayerDetails{}
	}

	result := []*PrayerDetails{}
	params := url.Values{}
	params.Add("city", opts.City)
	params.Add("country", opts.Country)

	resp, err := http.Get(fmt.Sprintf("%s?%s", opts.BaseURL, params.Encode()))
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
					Name: prayer,
					Time: timeParsed,
				})
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Time.Before(result[j].Time)
	})

	return result
}
