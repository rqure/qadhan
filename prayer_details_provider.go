package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/data"
	"github.com/rqure/qlib/pkg/data/query"
	"github.com/rqure/qlib/pkg/log"
	"github.com/rqure/qlib/pkg/signalslots"
)

type PrayerDetails struct {
	Name string
	Time time.Time
}

type PrayerDetailsProvider struct {
	store        data.Store
	isLeader     bool
	tickInterval time.Duration
	lastTick     time.Time

	NextPrayerStarted signalslots.Signal
	NextPrayerInfo    signalslots.Signal
}

func NewPrayerDetailsProvider(store data.Store) *PrayerDetailsProvider {
	return &PrayerDetailsProvider{
		store:        store,
		tickInterval: 10 * time.Second,
	}
}

func (a *PrayerDetailsProvider) OnBecameLeader(context.Context) {
	a.isLeader = true
}

func (a *PrayerDetailsProvider) OnLostLeadership(context.Context) {
	a.isLeader = false
}

func (a *PrayerDetailsProvider) Init(context.Context, app.Handle) {

}

func (a *PrayerDetailsProvider) Deinit(context.Context) {

}

func (a *PrayerDetailsProvider) DoWork(ctx context.Context) {
	if !a.isLeader {
		return
	}

	if time.Since(a.lastTick) < a.tickInterval {
		return
	}

	a.lastTick = time.Now()
	controllers := query.New(a.store).
		ForType("AdhanController").
		Execute(ctx)

	for _, controller := range controllers {
		capacity := controller.GetField("Prayer Buffer->Capacity").ReadInt(ctx)
		currentIndex := controller.GetField("Prayer Buffer->CurrentIndex").ReadInt(ctx)
		endIndex := controller.GetField("Prayer Buffer->EndIndex").ReadInt(ctx)

		if currentIndex == endIndex {
			log.Info("Prayer buffer is empty, querying next prayers")
			country := controller.GetField("Country").ReadString(ctx)
			city := controller.GetField("City").ReadString(ctx)
			baseUrl := controller.GetField("BaseURL").ReadString(ctx)
			prayerDetails := a.QueryNextPrayers(baseUrl, country, city)

			for _, prayer := range prayerDetails {
				if (endIndex+1)%capacity == currentIndex {
					log.Warn("Prayer buffer is full")
					break
				}

				controller.GetField(fmt.Sprintf("Prayer Buffer->%d->PrayerName", endIndex)).WriteString(ctx, prayer.Name)
				controller.GetField(fmt.Sprintf("Prayer Buffer->%d->StartTime", endIndex)).WriteTimestamp(ctx, prayer.Time)

				log.Info("Added prayer '%s' (startTime=%s) to the buffer (endIndex=%d)", prayer.Name, prayer.Time.Format(time.RFC3339), endIndex)

				endIndex = (endIndex + 1) % capacity
				controller.GetField("Prayer Buffer->EndIndex").WriteInt(ctx, endIndex)
			}
		} else {
			nextPrayer := &PrayerDetails{}

			nextPrayer.Name = controller.GetField(fmt.Sprintf("Prayer Buffer->%d->PrayerName", currentIndex)).ReadString(ctx)
			nextPrayer.Time = controller.GetField(fmt.Sprintf("Prayer Buffer->%d->StartTime", currentIndex)).ReadTimestamp(ctx)

			if time.Now().After(nextPrayer.Time) {
				log.Info("Next prayer '%s' has started", nextPrayer.Name)
				currentIndex = (currentIndex + 1) % capacity
				controller.GetField("Prayer Buffer->CurrentIndex").WriteInt(ctx, currentIndex)
				a.NextPrayerStarted.Emit(nextPrayer.Name)
			} else {
				a.NextPrayerInfo.Emit(nextPrayer.Name, nextPrayer.Time)
			}
		}
	}
}

func (a *PrayerDetailsProvider) QueryNextPrayers(baseUrl, country, city string) []*PrayerDetails {
	if baseUrl == "" || country == "" || city == "" {
		log.Error("Query options are invalid")
		return []*PrayerDetails{}
	}

	result := []*PrayerDetails{}
	params := url.Values{}
	params.Add("city", city)
	params.Add("country", country)

	resp, err := http.Get(fmt.Sprintf("%s?%s", baseUrl, params.Encode()))
	if err != nil {
		log.Error("Failed to fetch prayer times: %v", err)
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
		log.Error("Failed to decode prayer times: %v", err)
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
				log.Warn("Failed to load timezone: %v", err)
				loc = time.Local
			}

			timeParsed, err := time.ParseInLocation("02 Jan 2006 15:04 (MST)", fmt.Sprintf("%s %s", day.Date.Readable, timeStr), loc)
			if err != nil {
				log.Warn("Failed to parse time: %v", err)
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
