package main

import "os"

type LocationProvider interface {
	GetCity() string
	GetCountry() string
}

type EnvironmentLocationProvider struct{}

func (e *EnvironmentLocationProvider) GetCity() string {
	return os.Getenv("CITY")
}

func (e *EnvironmentLocationProvider) GetCountry() string {
	return os.Getenv("COUNTRY")
}
