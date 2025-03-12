package dto

type SetNINDetails struct {
	NIN string `json:"nin" validate:"required,len=11"`
}

type SetBVNDetails struct {
	BVN string `json:"bvn" validate:"required,len=11"`
}

type SetDriversLicenseDetails struct {
	DriverID string `json:"driverID" validate:"required,max=100"`
}

type SetVoterIDDetails struct {
	VoterID string `json:"voterID" validate:"required,max=100"`
}
