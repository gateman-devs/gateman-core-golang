package types

type FileUploaderType interface {
	GeneratedSignedURL(file_name string, permission SignedURLPermission) (*string, error)
	CheckFileExists(file_name string) (bool, error)
	DeleteFile(file_name string) error
}

type SignedURLPermission struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Delete bool `json:"delete"`
}