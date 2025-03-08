package types

import "time"

type FileUploaderType interface {
	GeneratedSignedURL(fileName string, permission SignedURLPermission, writeExpiresAt *time.Time, readExpiresAt *time.Duration) (*string, error)
	CheckFileExists(file_name string) (bool, error)
	DeleteFile(file_name string) error
}

type SignedURLPermission struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Delete bool `json:"delete"`
}
