package dto

type SetNINDetails struct {
	NIN string `json:"nin" validate:"required,eq=11"`
}

type SetBVNDetails struct {
	BVN string `json:"bvn" validate:"required,eq=11"`
}
