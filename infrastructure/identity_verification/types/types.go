package identity_verification_types

import "encoding/json"

type IdentityVerifierType interface {
	FetchBVNDetails(string) (*BVNData, error)
	FetchNINDetails(string) (*NINData, error)
	EmailVerification(email string) (bool, error)
	ImgLivenessCheck(img string) (bool, error)
}

type BVNData struct {
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	MiddleName       *string `json:"middle_name"`
	Gender           string  `json:"gender"`
	DateOfBirth      string  `json:"date_of_birth"`
	PhoneNumber      string  `json:"phone_number1"`
	Image            string  `json:"image"`
	Email            string  `json:"email"`
	EnrollmentBank   string  `json:"enrollment_bank"`
	EnrollmentBranch string  `json:"enrollment_branch"`
	LevelOfAccount   string  `json:"level_of_account"`
	LGAOfOrigin      string  `json:"lga_of_origin"`
	LGAOfResidence   string  `json:"lga_of_residence"`
	MaritalStatus    string  `json:"marital_status"`
	NameOnCard       string  `json:"name_on_card"`
	Nationality      string  `json:"nationality"`
	RegistrationDate string  `json:"registration_date"`
	Address          string  `json:"residential_address"`
	StateOfOrigin    string  `json:"state_of_origin"`
	StateOfResidence string  `json:"state_of_residence"`
	Title            string  `json:"title"`
	WatchListed      string  `json:"watch_listed"`
}

func (b *BVNData) MarshalBinary() ([]byte, error) {
	return json.Marshal(b) // Serialize to JSON
}

// Implement encoding.BinaryUnmarshaler
func (b *BVNData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b) // Deserialize from JSON
}

type NINData struct {
	FirstName             string  `json:"first_name"`
	MiddleName            *string  `json:"middle_name"`
	LastName              string  `json:"last_name"`
	PhoneNumber           *string  `json:"phone_number"`
	Photo                 string  `json:"photo"`
	Gender                string  `json:"gender"`
	Customer              string  `json:"customer"`
	DateOfBirth           string  `json:"date_of_birth"`
	Email                 *string `json:"email"`
	EmploymentStatus      string  `json:"employment_status"`
	MaritalStatus         string  `json:"marital_status"`
	BirthCountry          string  `json:"birth_country"`
	BirthLGA              string  `json:"birth_lga"`
	BirthState            string  `json:"birth_state"`
	EducationalLevel      string  `json:"educational_level"`
	MaidenName            *string `json:"maiden_name"`
	NSpokenLang           string  `json:"nspoken_lang"`
	Profession            *string `json:"profession"`
	Religion              string  `json:"religion"`
	Address string  `json:"residence_address_line_1"`
	ResidenceAddressLine2 *string `json:"residence_address_line_2"`
	ResidenceStatus       string  `json:"residence_status"`
	ResidenceTown         string  `json:"residence_town"`
	ResidenceLGA          string  `json:"residence_lga"`
	ResidenceState        string  `json:"residence_state"`
	Signature             string  `json:"signature"`
	NOKFirstName          string  `json:"nok_first_name"`
	NOKLastName           string  `json:"nok_last_name"`
	NOKMiddleName         string  `json:"nok_middle_name"`
	NOKAddress1           string  `json:"nok_address_1"`
	NOKAddress2           string  `json:"nok_address_2"`
	NOKLGA                string  `json:"nok_lga"`
	NOKState              string  `json:"nok_state"`
	NOKPostalCode         string  `json:"nok_postal_code"`
	OSpokenLang           *string `json:"ospoken_lang"`
	OriginLGA             string  `json:"origin_lga"`
	OriginPlace           string  `json:"origin_place"`
	OriginState           string  `json:"origin_state"`
	Height                string  `json:"height"`
	PFirstName            *string `json:"p_first_name"`
	PMiddleName           *string `json:"p_middle_name"`
	PLastName             *string `json:"p_last_name"`
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
