package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	qmq "github.com/rqure/qmq/src"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PrayerInfoProvider interface {
	GetNextPrayers() []*qmq.Prayer
}

type AlAdhanPrayerInfoProvider struct {
	locationProvider LocationProvider
	logger           qmq.Logger
}

func NewAlAdhanPrayerInfoProvider(locationProvider LocationProvider, logger qmq.Logger) PrayerInfoProvider {
	return &AlAdhanPrayerInfoProvider{
		locationProvider: locationProvider,
		logger:           logger,
	}
}

func (a *AlAdhanPrayerInfoProvider) GetNextPrayers() []*qmq.Prayer {
	if a.locationProvider.GetCity() == "" || a.locationProvider.GetCountry() == "" {
		a.logger.Panic("City or country not set")
	}

	result := []*qmq.Prayer{}

	baseURL := "http://api.aladhan.com/v1/calendarByCity"
	params := url.Values{}
	params.Add("city", a.locationProvider.GetCity())
	params.Add("country", a.locationProvider.GetCountry())

	resp, err := http.Get(fmt.Sprintf("%s?%s", baseURL, params.Encode()))
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to fetch prayer times: %v", err))
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
		a.logger.Error(fmt.Sprintf("Failed to decode prayer times: %v", err))
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
				a.logger.Warn(fmt.Sprintf("Failed to load timezone: %v", err))
				loc = time.Local
			}

			timeParsed, err := time.ParseInLocation("02 Jan 2006 15:04 (MST)", fmt.Sprintf("%s %s", day.Date.Readable, timeStr), loc)
			if err != nil {
				a.logger.Warn(fmt.Sprintf("Failed to parse time: %v", err))
				continue
			}

			if time.Now().Before(timeParsed) {
				result = append(result, &qmq.Prayer{
					Name: prayer,
					Time: timestamppb.New(timeParsed),
				})
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Time.AsTime().Before(result[j].Time.AsTime())
	})

	return result
}
