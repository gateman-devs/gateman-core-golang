package fileupload

import (
	"os"

	"gateman.io/infrastructure/file_upload/cloudflare"
	"gateman.io/infrastructure/file_upload/types"
)

var FileUploader types.FileUploaderType

func InitialiseFileUploader() {
	r2Client := &cloudflare.R2SignedURLService{
		AccountID:       os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("CLOUDFLARE_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("CLOUDFLARE_SECRET_ACCESS_KEY"),
	}
	r2Client.InitialiseClient()
	FileUploader = r2Client
}
