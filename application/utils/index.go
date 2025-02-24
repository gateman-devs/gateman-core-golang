package utils

import (
	"net"
	"regexp"
	"strings"
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

func ValidateIPAddresses(addresses []string) bool {
	invalid := false

	for _, addr := range addresses {
		// Remove any whitespace
		addr = strings.TrimSpace(addr)

		// Try parsing as IP address
		ip := net.ParseIP(addr)

		if ip == nil {
			invalid = true
			break
		}
	}

	return invalid
}

func MakeStringArrayUnique(arr []string) []string {
	seen := make(map[string]struct{})
	for _, str := range arr {
		seen[str] = struct{}{}
	}

	// Convert map keys back to slice
	result := make([]string, 0, len(seen))
	for str := range seen {
		result = append(result, str)
	}

	return result
}
