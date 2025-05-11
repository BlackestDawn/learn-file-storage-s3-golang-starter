package main

import (
	"context"
	"strings"
	"time"

	"github.com/BlackestDawn/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (retVal string, err error) {
	presignClient := s3.NewPresignClient(s3Client, s3.WithPresignExpires(expireTime))
	presignData, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expireTime))
	if err != nil {
		return
	}

	retVal = presignData.URL

	return
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	pathData := strings.Split(*video.VideoURL, ",")
	presignedURL, err := generatePresignedURL(cfg.s3Client, pathData[0], pathData[1], time.Hour)
	if err != nil {
		return video, err
	}

	video.VideoURL = &presignedURL

	return video, nil
}
