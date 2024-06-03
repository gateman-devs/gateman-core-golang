package utils

import (
	"regexp"
	"time"

	"github.com/oklog/ulid/v2"
)

func GenerateUULDString() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.DefaultEntropy()).String()
}

func GetStringPointer(text string) *string {
	return &text
}

func GetBooleanPointer(data bool) *bool {
	return &data
}

func GetFloat32Pointer(data float32) *float32 {
	return &data
}

func GetUInt64Pointer(data uint64) *uint64 {
	return &data
}

func GetUIntPointer(data uint) *uint {
	return &data
}

func GetInt64Pointer(data int64) *int64 {
	return &data
}

func ExtractAppVersionFromUserAgentHeader(userAgent string) *string {
	regex := regexp.MustCompile(`Polymer/([0-9.]+)`)
	matches := regex.FindStringSubmatch(userAgent)
	if len(matches) >= 2 {
		return &matches[1]
	}
	return nil
}

func HasItemString(arr *[]string, target string) bool {
	for _, v := range *arr {
		if v == target {
			return true
		}
	}
	return false
}
