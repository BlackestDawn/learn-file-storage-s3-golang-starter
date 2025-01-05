package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/BlackestDawn/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	fileType := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(fileType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing mediatype", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Unsupported file format for thumbnail: "+mediaType, nil)
		return
	}
	fileExt, err := mime.ExtensionsByType(fileType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing MIME type", err)
		return
	}

	/*
		fileContent, err := io.ReadAll(file)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Unable to read thumbnail file", err)
			return
		}
	*/

	//fileContentBase64 := base64.StdEncoding.EncodeToString(fileContent)

	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not fetch metadata", err)
		return
	}
	if metadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "UserID mismatch", fmt.Errorf("user: %s, file owner: %s", userID.String(), metadata.UserID.String()))
		return
	}

	/*
		thumbnailData := thumbnail{
			data:      fileContent,
			mediaType: fileType,
		}
		videoThumbnails[metadata.UserID] = thumbnailData
	*/

	filename := make([]byte, 32)
	_, err = rand.Read(filename)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating filename", err)
		return
	}
	thumbnailPath := filepath.Join(cfg.assetsRoot, base64.RawURLEncoding.EncodeToString(filename)+fileExt[0])

	thumbnailFile, err := os.Create(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create new thumbnail file", err)
		return
	}

	_, err = io.Copy(thumbnailFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to write thumbnail file", err)
		return
	}

	/*
		thumbnailURL := "data:" + fileType + ";base64," + fileContentBase64
		metadata.ThumbnailURL = &thumbnailURL
	*/
	thumbnailURL := "http://localhost:" + cfg.port + "/" + thumbnailPath
	//thumbnailURL := "/" + thumbnailPath
	metadata.ThumbnailURL = &thumbnailURL
	metadata.UpdatedAt = time.Now()

	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error updating video data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, metadata)
}
