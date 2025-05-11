package main

import "os/exec"

func processVideoForFastStart(filePath string) (retVal string, err error) {
	outputPath := filePath + ".processing"

	err = exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath).Run()
	if err != nil {
		return
	}

	retVal = outputPath

	return
}
