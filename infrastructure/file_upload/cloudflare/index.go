package cloudflare

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"authone.usepolymer.co/application/utils"
)

type R2SignedURLService struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

func (c *R2SignedURLService) GenerateUploadURL(fileName string) (*string, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.AccountID)
	timestamp := time.Now().UTC()
	dateStamp := timestamp.Format("20060102")
	amzDateTime := timestamp.Format("20060102T150405Z")

	// Define the canonical request components
	httpMethod := "PUT"
	canonicalURI := fmt.Sprintf("/%s/%s", os.Getenv("R2_BUCKET"), fileName)

	// Query parameters for signing
	query := url.Values{}
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", fmt.Sprintf("%s/%s/%s/s3/aws4_request",
		c.AccessKeyID, dateStamp, c.Region))
	query.Set("X-Amz-Date", amzDateTime)
	query.Set("X-Amz-Expires", fmt.Sprintf("%d", 300))
	query.Set("X-Amz-SignedHeaders", "content-type;host;x-amz-acl")

	// Create canonical headers
	host := fmt.Sprintf("%s.r2.cloudflarestorage.com", c.AccountID)
	canonicalHeaders := fmt.Sprintf("host:%s\n", host)

	// Create canonical request
	canonicalRequest := strings.Join([]string{
		httpMethod,
		canonicalURI,
		query.Encode(),
		canonicalHeaders,
		"content-type;host;x-amz-acl",
		"UNSIGNED-PAYLOAD",
	}, "\n")

	// Create string to sign
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDateTime,
		fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, c.Region),
		hex.EncodeToString(sha256.New().Sum([]byte(canonicalRequest))),
	}, "\n")

	// Calculate signature
	dateKey := hmacSHA256([]byte("AWS4"+c.SecretAccessKey), []byte(dateStamp))
	dateRegionKey := hmacSHA256(dateKey, []byte(c.Region))
	dateRegionServiceKey := hmacSHA256(dateRegionKey, []byte("s3"))
	signingKey := hmacSHA256(dateRegionServiceKey, []byte("aws4_request"))
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Add signature to query parameters
	query.Set("X-Amz-Signature", signature)

	// Build final URL
	return utils.GetStringPointer(fmt.Sprintf("%s%s?%s", endpoint, canonicalURI, query.Encode())), nil
}

func (c *R2SignedURLService) GenerateDownloadURL(fileName string) (*string, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.AccountID)
	timestamp := time.Now().UTC()
	dateStamp := timestamp.Format("20060102")
	amzDateTime := timestamp.Format("20060102T150405Z")

	// Define the canonical request components
	httpMethod := "GET"
	canonicalURI := fmt.Sprintf("/%s/%s", os.Getenv("R2_BUCKET"), fileName)

	// Query parameters for signing
	query := url.Values{}
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", fmt.Sprintf("%s/%s/%s/s3/aws4_request",
		c.AccessKeyID, dateStamp, c.Region))
	query.Set("X-Amz-Date", amzDateTime)
	query.Set("X-Amz-Expires", fmt.Sprintf("%d", time.Second*300))
	query.Set("X-Amz-SignedHeaders", "host")

	// Create canonical headers
	host := fmt.Sprintf("%s.r2.cloudflarestorage.com", c.AccountID)
	canonicalHeaders := fmt.Sprintf("host:%s\n", host)

	// Create canonical request
	canonicalRequest := strings.Join([]string{
		httpMethod,
		canonicalURI,
		query.Encode(),
		canonicalHeaders,
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")

	// Create string to sign
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDateTime,
		fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, c.Region),
		hex.EncodeToString(sha256.New().Sum([]byte(canonicalRequest))),
	}, "\n")

	// Calculate signature
	dateKey := hmacSHA256([]byte("AWS4"+c.SecretAccessKey), []byte(dateStamp))
	dateRegionKey := hmacSHA256(dateKey, []byte(c.Region))
	dateRegionServiceKey := hmacSHA256(dateRegionKey, []byte("s3"))
	signingKey := hmacSHA256(dateRegionServiceKey, []byte("aws4_request"))
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Add signature to query parameters
	query.Set("X-Amz-Signature", signature)

	// Build final URL
	return utils.GetStringPointer(fmt.Sprintf("%s%s?%s", endpoint, canonicalURI, query.Encode())), nil
}

func (azurlservice *R2SignedURLService) DeleteFile(fileName string) error {
	// credential, err := azblob.NewSharedKeyCredential(azurlservice.AccountName, azurlservice.AccountKey)
	// if err != nil {
	// 	logger.Error("error generated azblob shared key credential", logger.LoggerOptions{
	// 		Key:  "error",
	// 		Data: err,
	// 	})
	// 	return err
	// }
	// URL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", azurlservice.AccountName, azurlservice.ContainerName, fileName))
	// if err != nil {
	// 	logger.Error("error parsing shared token url", logger.LoggerOptions{
	// 		Key:  "error",
	// 		Data: err,
	// 	})
	// 	return err
	// }
	// blobURL := azblob.NewBlockBlobURL(*URL, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	// _, err = blobURL.Delete(context.TODO(), azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
	// if err != nil {
	// 	logger.Error("error parsing shared token url", logger.LoggerOptions{
	// 		Key:  "error",
	// 		Data: err,
	// 	})
	// 	return err
	// }
	return nil
}

func (azurlservice *R2SignedURLService) CheckFileExists(fileName string) (bool, error) {
	// credential, err := azblob.NewSharedKeyCredential(azurlservice.AccountName, azurlservice.AccountKey)
	// if err != nil {
	// 	logger.Error("error generated azblob shared key credential", logger.LoggerOptions{
	// 		Key:  "error",
	// 		Data: err,
	// 	})
	// 	return false, err
	// }
	// URL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", azurlservice.AccountName, azurlservice.ContainerName, fileName))
	// if err != nil {
	// 	logger.Error("error parsing shared token url", logger.LoggerOptions{
	// 		Key:  "error",
	// 		Data: err,
	// 	})
	// 	return false, err
	// }
	// blobURL := azblob.NewBlockBlobURL(*URL, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	// _, err = blobURL.GetProperties(context.TODO(), azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	// if err != nil {
	// 	if serr, ok := err.(azblob.StorageError); ok {
	// 		if serr.ServiceCode() == azblob.ServiceCodeBlobNotFound {
	// 			return false, nil
	// 		}
	// 	}
	// 	return false, err
	// }
	return true, nil
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
