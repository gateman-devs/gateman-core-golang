package dto

type SetNINDetails struct {
	NIN string `json:"nin" validate:"required,len=11"`
}

type SetBVNDetails struct {
	BVN string `json:"bvn" validate:"required,len=11"`
}
