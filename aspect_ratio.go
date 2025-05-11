package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os/exec"
)

type aspectData struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type videoData struct {
	Streams []aspectData `json:"streams"`
}

func getVideoAspectRatio(filepath string) (retVal string, err error) {
	// Probe video file
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	outputBuffer := bytes.Buffer{}
	cmd.Stdout = &outputBuffer
	err = cmd.Run()
	if err != nil {
		return
	}

	// Unmarshal output
	outputJSON := videoData{
		Streams: []aspectData{},
	}
	err = json.Unmarshal(outputBuffer.Bytes(), &outputJSON)
	if err != nil {
		return
	}

	// Calculate apsect ratio
	aspectRatio := math.Round(float64(outputJSON.Streams[0].Width)/float64(outputJSON.Streams[0].Height)*100) / 100
	if aspectRatio == math.Round(1600.0/9.0)/100 {
		retVal = "16:9"
	} else if aspectRatio == math.Round(900.0/16.0)/100 {
		retVal = "9:16"
	} else {
		retVal = "other"
	}

	return
}
