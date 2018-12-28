package main

import (
	"encoding/json"
	"net/http"
	"log"
	"io/ioutil"
	"errors"
	"strconv"
)

// Struct definitions
type AzuraCast struct {
	base_url       string
}

type AzuraCastApiResponse struct {
	Station *AzuraCastStation `json:"station"`
	NowPlaying struct {
		Song *AzuraCastNowPlaying `json:"song"`
	} `json:"now_playing"`
}

type AzuraCastStation struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ListenURL   string `json:"listen_url"`
}

type AzuraCastNowPlaying struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
}

// Update now playing data using the already stored station ID.
func (ac *AzuraCast) UpdateNowPlaying(v *VoiceInstance) (error) {
	return ac.GetNowPlaying(v, strconv.FormatInt(v.station.ID, 10))
}

// Get now playing data for the station specified.
func (ac *AzuraCast) GetNowPlaying(v *VoiceInstance, stationName string) (error) {

	apiUrl := ac.base_url+"/api/nowplaying/"+stationName

	resp, err := http.Get(apiUrl)
	if err != nil {
		log.Println("ERROR: Odpowiedź AzuraCast", err.Error())
		return err
	}

	if resp.StatusCode == http.StatusOK {
		apiResponseBody, _ := ioutil.ReadAll(resp.Body)

		var apiResponse AzuraCastApiResponse
		json.Unmarshal(apiResponseBody, &apiResponse)

		v.station = apiResponse.Station
		v.np = apiResponse.NowPlaying.Song
		return nil
	} else {
		return errors.New("API URL zwrócił status"+resp.Status)
	}
}