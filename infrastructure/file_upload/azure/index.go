package azure

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"authone.usepolymer.co/infrastructure/file_upload/types"
	"authone.usepolymer.co/infrastructure/logger"
	_azblob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	azblob_sas "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	azblob "github.com/Azure/azure-storage-blob-go/azblob"
)

type AzureBlobSignedURLService struct {
	AccountName   string
	ContainerName string
	AccountKey    string
}

func (azurlservice *AzureBlobSignedURLService) GeneratedSignedURL(file_name string, permission types.SignedURLPermission) (*string, error) {
	if permission.Read == permission.Write {
		return nil, errors.New("permission must be either read or write")
	}
	_credential, err := _azblob.NewSharedKeyCredential(azurlservice.AccountName, azurlservice.AccountKey)
	if err != nil {
		logger.Error("error generated _azblob shared key credential", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}

	credential, err := azblob.NewSharedKeyCredential(azurlservice.AccountName, azurlservice.AccountKey)
	if err != nil {
		logger.Error("error generated azblob shared key credential", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}
	URL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", azurlservice.AccountName, azurlservice.ContainerName, file_name))
	if err != nil {
		logger.Error("error parsing shared token url", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}
	blobURL := azblob.NewBlockBlobURL(*URL, azblob.NewPipeline(credential, azblob.PipelineOptions{}))

	sasQueryParams, err := azblob_sas.BlobSignatureValues{
		Protocol:      azblob_sas.ProtocolHTTPS,
		StartTime:     time.Now().UTC(),
		ExpiryTime:    time.Now().UTC().Add(5 * time.Minute), // url is valid for only 5 mins
		Permissions:   (&azblob_sas.BlobPermissions{Read: permission.Read, Write: permission.Write, Delete: permission.Delete}).String(),
		ContainerName: azurlservice.ContainerName,
		BlobName:      file_name,
	}.SignWithSharedKey(_credential)
	if err != nil {
		logger.Error("error blob signature values", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}
	sasURL := fmt.Sprintf("%s?%s", blobURL.String(), sasQueryParams.Encode())
	return &sasURL, nil
}

func (azurlservice *AzureBlobSignedURLService) DeleteFile(file_name string) error {
	credential, err := azblob.NewSharedKeyCredential(azurlservice.AccountName, azurlservice.AccountKey)
	if err != nil {
		logger.Error("error generated azblob shared key credential", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	URL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", azurlservice.AccountName, azurlservice.ContainerName, file_name))
	if err != nil {
		logger.Error("error parsing shared token url", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	blobURL := azblob.NewBlockBlobURL(*URL, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	_, err = blobURL.Delete(context.TODO(), azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
	if err != nil {
		logger.Error("error parsing shared token url", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	return nil
}

func (azurlservice *AzureBlobSignedURLService) CheckFileExists(file_name string) (bool, error) {
	credential, err := azblob.NewSharedKeyCredential(azurlservice.AccountName, azurlservice.AccountKey)
	if err != nil {
		logger.Error("error generated azblob shared key credential", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return false, err
	}
	URL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", azurlservice.AccountName, azurlservice.ContainerName, file_name))
	if err != nil {
		logger.Error("error parsing shared token url", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return false, err
	}
	blobURL := azblob.NewBlockBlobURL(*URL, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	_, err = blobURL.GetProperties(context.TODO(), azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		if serr, ok := err.(azblob.StorageError); ok {
			if serr.ServiceCode() == azblob.ServiceCodeBlobNotFound {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}
