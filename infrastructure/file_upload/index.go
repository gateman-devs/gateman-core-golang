package fileupload

import (
	"authone.usepolymer.co/infrastructure/file_upload/cloudflare"
	"authone.usepolymer.co/infrastructure/file_upload/types"
)

var FileUploader types.FileUploaderType

func InitialiseFileUploader() {
	FileUploader = &cloudflare.R2SignedURLService{
		AccountID:       "f04e2cfcc01f9104510c6209b400063c",
		AccessKeyID:     "9da6a81cc32f959faf59b6c66e50678e",
		SecretAccessKey: "gVRw0yYvUG8CQUYzY0yiIA6OFwUCWDVycsVyOWlE",
		Region:          "auto",
	}
}
