package main

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/BlackestDawn/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Get JWT
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	// Validate JWT
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// Max file size
	const maxMemory = 1 << 30

	// Get video ID
	videoIDString := r.URL.Path[len("/api/video_upload/"):]
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// Get and validate metadata
	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not fetch metadata", err)
		return
	}
	if metadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "UserID mismatch", nil)
		return
	}

	// Parse form
	r.ParseMultipartForm(maxMemory)
	videoFile, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer videoFile.Close()

	// Validate MIME
	mimeType := header.Header.Get("Content-Type")
	if mimeType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Unsupported file format for video: "+mimeType, nil)
		return
	}
	fileExt, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing MIME type", err)
		return
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "tubely-upload.*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temporary file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Save file to temporary file
	_, err = io.Copy(tempFile, videoFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to write temporary file", err)
		return
	}

	// Get aspect ratio
	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get aspect ratio", err)
		return
	}
	if aspectRatio == "16:9" {
		aspectRatio = "landscape"
	} else if aspectRatio == "9:16" {
		aspectRatio = "portrait"
	}

	// Process for fast start
	tempFile.Seek(0, io.SeekStart)
	fastStartPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to process video for fast start", err)
		return
	}
	defer os.Remove(fastStartPath)

	// Upload to S3
	filename := make([]byte, 32)
	_, err = rand.Read(filename)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating filename", err)
		return
	}
	fastStartFile, err := os.Open(fastStartPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to open fast start file", err)
		return
	}
	defer fastStartFile.Close()
	fileKey := aspectRatio + "/" + base64.RawURLEncoding.EncodeToString(filename) + fileExt[0]
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         &fileKey,
		Body:        fastStartFile,
		ContentType: &mimeType,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload to S3", err)
		return
	}

	// Update metadata
	//videoURL := "https://" + cfg.s3Bucket + ".s3." + cfg.s3Region + ".amazonaws.com/" + fileKey
	videoURL := cfg.s3Bucket + "," + fileKey
	metadata.VideoURL = &videoURL
	metadata.UpdatedAt = time.Now()
	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error updating video data", err)
		return
	}

	// Sign URL
	metadata, err = cfg.dbVideoToSignedVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to generate presigned URL", err)
		return
	}

	// Response
	log.Println("Video uploaded successfully to:", videoURL)
	respondWithJSON(w, http.StatusOK, metadata)
}
