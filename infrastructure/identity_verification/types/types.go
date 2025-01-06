package identity_verification_types

import "encoding/json"

type IdentityVerifierType interface {
	FetchBVNDetails(string) (*BVNData, error)
	FetchNINDetails(string) (*NINData, error)
	EmailVerification(email string) (bool, error)
	ImgLivenessCheck(img string) (bool, error)
}

type BVNData struct {
	Gender        string  `json:"gender"`
	WatchListed   string  `json:"watch_listed"`
	FirstName     string  `json:"first_name"`
	MiddleName    *string `json:"middle_name"`
	LastName      string  `json:"last_name"`
	DateOfBirth   string  `json:"date_of_birth"`
	PhoneNumber   string  `json:"phone_number1"`
	Nationality   string  `json:"nationality"`
	Address       string  `json:"residential_address"`
	Base64Image   string  `json:"image"`
	NIN           string  `json:"nin"`
	LGAOfOrigin   string  `json:"lga_of_origin"`
	StateOfOrigin string  `json:"state_of_origin"`
	Title         string  `json:"title"`
}

func (b *BVNData) MarshalBinary() ([]byte, error) {
	return json.Marshal(b) // Serialize to JSON
}

// Implement encoding.BinaryUnmarshaler
func (b *BVNData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b) // Deserialize from JSON
}

type NINData struct {
	Gender      string  `json:"gender"`
	FirstName   string  `json:"first_name"`
	MiddleName  *string `json:"middle_name"`
	LastName    string  `json:"last_name"`
	DateOfBirth string  `json:"date_of_birth"`
	PhoneNumber *string `json:"phone_number"`
	Address     string  `json:"address"`
	Base64Image string  `json:"photo"`
}

func (n *NINData) MarshalBinary() ([]byte, error) {
	return json.Marshal(n) // Serialize to JSON
}

// Implement encoding.BinaryUnmarshaler
func (n *NINData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, n) // Deserialize from JSON
}

type LivenessCheckResult struct {
	Entity LivenessCheckResultEntity `json:"entity"`
}

type LivenessCheckResultEntity struct {
	Liveness LivenessCheckResultLiveness `json:"liveness"`
}

type LivenessCheckResultLiveness struct {
	LivenessCheck       bool    `json:"liveness_check"`
	LivenessProbability float32 `json:"liveness_probability"`
}
