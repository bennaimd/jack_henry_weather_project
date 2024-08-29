package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

const (
	// shortcut: I would typically use an enum if I encountered something like this in a production-ready environment
	tempModerate = 50
	tempHot      = 80
	cold         = "Cold"
	moderate     = "Moderate"
	hot          = "Hot"

	weatherService  = "api.weather.gov"
	serviceEndpoint = "points"
)

type forecastHandler struct{}

func (h forecastHandler) fetchForecast(forecastUrl string) (characterization string, forecast string, err error) {
	var (
		data       map[string]interface{}
		properties map[string]interface{}
		periods    []interface{}
		periodNow  map[string]interface{}

		temp int
	)
	resp, err := http.Get(forecastUrl)
	if err != nil {
		log.Println(fmt.Printf("failed to retrieve Forecast API: %s", err))
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: http status %s", resp.Status)
		return
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(fmt.Printf("failed to read Forecast API response: %s", err))
		return
	}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		log.Println(fmt.Printf("failed to unmarshal Forecast API response: %s", err))
		return
	}
	// shortcut: I would typically add validation steps
	properties = data["properties"].(map[string]interface{})
	periods = properties["periods"].([]interface{})
	periodNow = periods[0].(map[string]interface{})

	temp, err = strconv.Atoi(fmt.Sprintf("%v", periodNow["temperature"]))
	if err != nil {
		log.Println(fmt.Printf("failed to parse temperature from Forecast API response: %s", err))
		return

	}
	if temp >= tempHot {
		characterization = hot
	} else if temp >= tempModerate {
		characterization = moderate
	} else {
		characterization = cold
	}
	forecast = fmt.Sprintf("%v", periodNow["shortForecast"])
	return
}

func (h forecastHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		data       map[string]interface{}
		properties map[string]interface{}
	)
	weatherUrl := url.URL{
		Scheme: "https",
		Host:   weatherService,
		Path:   serviceEndpoint,
	}
	weatherUrlString, err := url.JoinPath(weatherUrl.String(), r.URL.String()) //shortcut: not validating URL contains a valid lat/long
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Error"))) //shortcut: skipping the error handling for Write() here because the function will exit anyway
		log.Println(fmt.Printf("failed to build weather API URL: %s", err))
	}

	resp, err := http.Get(weatherUrlString)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Error")))
		log.Println(fmt.Printf("failed to retrieve Points API: %s", err))
	}
	if resp.StatusCode != http.StatusOK {
		w.Write([]byte(fmt.Sprintf("Error")))
		return
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Error")))
		log.Println(fmt.Printf("failed to read Points API response: %s", err))
	}
	if err := json.Unmarshal(bytes, &data); err != nil {
		w.Write([]byte(fmt.Sprintf("Error")))
		log.Println(fmt.Printf("failed to unmarshal Points API response: %s", err))
		return
	}

	properties = data["properties"].(map[string]interface{})
	forecastUrlString := fmt.Sprintf("%v", properties["forecast"]) //shortcut: lazy/potentially unsafe string conversion
	temp, characterization, err := h.fetchForecast(forecastUrlString)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Error")))
		log.Println(fmt.Printf("failed get forecast: %s", err))
		return
	}
	if _, err = w.Write([]byte(fmt.Sprintf("The current forecast is %s and %s", characterization, temp))); err != nil {
		log.Fatal(err)
	}
}

func main() {
	handler := forecastHandler{}
	http.Handle("/forecast/{latLong}", handler)
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
