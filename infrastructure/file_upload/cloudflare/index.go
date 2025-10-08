package cloudflare

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gateman.io/application/utils"
	upload_types "gateman.io/infrastructure/file_upload/types"
	"gateman.io/infrastructure/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2SignedURLService struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Client          *s3.Client
}

func (c *R2SignedURLService) InitialiseClient() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Fatal(err)
	}

	c.Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.AccountID))
	})
}
func (c *R2SignedURLService) GeneratedSignedURL(fileName string, permission upload_types.SignedURLPermission, expiresAt time.Duration) (*string, error) {
	presignClient := s3.NewPresignClient(c.Client)
	var presignResult *v4.PresignedHTTPRequest
	var url *string
	var err error

	putOpts := s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("R2_BUCKET")),
		Key:    aws.String(fileName),
	}

	// Public access is handled via bucket policy, so no need to set ACL here.
	if permission.Write {
		presignResult, err = presignClient.PresignPutObject(context.TODO(), &putOpts, func(opts *s3.PresignOptions) {
			opts.Expires = expiresAt
		})
	} else {
		presignResult, err = presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("R2_BUCKET")),
			Key:    aws.String(fileName),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = expiresAt
		})
	}

	if err != nil {
		logger.Error("an error occurred while trying to generate presigned URL", logger.LoggerOptions{
			Key:  "fileName",
			Data: fileName,
		}, logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		return nil, err
	}

	if url != nil {
		return url, nil
	}

	return utils.GetStringPointer(presignResult.URL), nil
}

func (c *R2SignedURLService) DeleteFile(fileName string) error {
	_, err := c.Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(os.Getenv("R2_BUCKET")),
		Key:    aws.String(fileName),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (c *R2SignedURLService) CheckFileExists(fileName string) (bool, error) {
	_, err := c.Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(os.Getenv("R2_BUCKET")),
		Key:    aws.String(fileName),
	})
	if err != nil {
		if len(strings.Split(err.Error(), "StatusCode: 404")) == 2 {
			return false, nil
		}
		logger.Error("an error occured while checking file exists on cloudflare", logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}
	return true, nil

}
