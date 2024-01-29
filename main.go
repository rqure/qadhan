package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"time"

	qmq "github.com/rqure/qmq/src"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func QueryPrayerTimesForThisMonth() ([]*qmq.QMQPrayer, error) {
	var result []*qmq.QMQPrayer

	baseURL := "http://api.aladhan.com/v1/calendarByCity"
	params := url.Values{}
	params.Add("city", os.Getenv("CITY"))
	params.Add("country", os.Getenv("COUNTRY"))

	resp, err := http.Get(fmt.Sprintf("%s?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
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
			// Add 'Meta' field here if needed
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	for _, day := range response.Data {
		for prayer, timeStr := range map[string]string{
			"Fajr":    day.Timings.Fajr,
			"Dhuhr":   day.Timings.Dhuhr,
			"Asr":     day.Timings.Asr,
			"Maghrib": day.Timings.Maghrib,
			"Isha":    day.Timings.Isha,
		} {
			timeParsed, err := time.Parse("02 Jan 2006 15:04 (MST)", fmt.Sprintf("%s %s", day.Date.Readable, timeStr))
			if err != nil {
				continue
			}

			if time.Now().Before(timeParsed) {
				result = append(result, &qmq.QMQPrayer{
					Name: prayer,
					Time: timestamppb.New(timeParsed),
				})
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Time.AsTime().Before(result[j].Time.AsTime())
	})

	return result, nil
}

func main() {
	app := qmq.NewQMQApplication("prayer")
	app.Initialize()
	defer app.Deinitialize()

	app.AddProducer("prayer:time:queue").Initialize(500)
	app.AddProducer("prayer:adhan:exchange").Initialize(1)
	app.AddConsumer("prayer:time:queue").Initialize()

	tickRateMs, err := strconv.Atoi(os.Getenv("TICK_RATE_MS"))
	if err != nil {
		tickRateMs = 100
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	ticker := time.NewTicker(time.Duration(tickRateMs) * time.Millisecond)
	for {
		select {
		case <-sigint:
			app.Logger().Advise("SIGINT received")
			return
		case <-ticker.C:
			next_prayer := &qmq.QMQPrayer{}
			popped := app.Consumer("prayer:time:queue").Pop(next_prayer)
			if popped != nil {
				if time.Now().After(next_prayer.Time.AsTime()) {
					app.Logger().Advise(fmt.Sprintf("It is now time for: %s", next_prayer.Name))

					audioFiles := []string{
						"adhan-0.mp3",
						"adhan-1.mp3",
						"adhan-2.mp3",
						"adhan-3.mp3",
						"adhan-5.mp3",
						"adhan-wahhab.mp3"}

					if next_prayer.Name == "Fajr" {
						audioFiles = []string{"/app/audio/adhan/fajr-1.mp3"}
					}

					randomIndex := rand.Intn(len(audioFiles))
					audioFile := audioFiles[randomIndex]

					app.Producer("prayer:adhan:exchange").Push(&qmq.QMQAudioRequest{
						Filename: audioFile,
					})
					popped.Ack()
				} else {
					popped.Dispose()
				}
			} else {
				app.Logger().Advise("Querying prayer timees for this month")
				prayers, err := QueryPrayerTimesForThisMonth()
				if err != nil {
					app.Logger().Error(fmt.Sprintf("Failed to query prayer times: %v", err))
				} else {
					for _, prayer := range prayers {
						app.Logger().Debug(fmt.Sprintf("Found prayer '%s' starting at '%s'", prayer.Name, prayer.Time.AsTime().String()))
						app.Producer("prayer:time:queue").Push(prayer)
					}
				}
			}
		}
	}
}
