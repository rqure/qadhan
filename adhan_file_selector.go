package main

import "math/rand"

type AdhanFileSelector interface {
	Select(prayerName string) string
}

type DefaultAdhanFileSelector struct {
	FajrAdhanFiles  []string
	OtherAdhanFiles []string
}

func (s *DefaultAdhanFileSelector) Select(prayerName string) string {
	var files []string

	switch prayerName {
	case "Fajr":
		files = s.FajrAdhanFiles
	default:
		files = s.OtherAdhanFiles
	}

	randomIndex := rand.Intn(len(files))
	return files[randomIndex]
}
