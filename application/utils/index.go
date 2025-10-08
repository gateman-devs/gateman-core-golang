package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	mathRand "math/rand"
	"net"
	"net/http"
	"net/url"
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

func CreateHMACSHA512Hash(data []byte, secretKey string) string {
	hmac := hmac.New(sha512.New, []byte(secretKey))
	hmac.Write(data)
	return hex.EncodeToString(hmac.Sum(nil))
}

func GenerateRandomHexKey(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than 0")
	}
	byteLength := length / 2
	if length%2 != 0 {
		byteLength++
	}
	randomBytes := make([]byte, byteLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	hexString := hex.EncodeToString(randomBytes)
	return hexString[:length], nil
}

func GenerateAccountRecoveryCodes(amount int) []string {
	mathRand.New(mathRand.NewSource(time.Now().UnixNano()))

	// Create a map to store unique codes
	uniqueCodes := make(map[string]bool)
	uniqueCodeArr := []string{}

	// Generate 8 unique 6-digit codes
	for len(uniqueCodes) < amount {
		code := fmt.Sprintf("%06d", mathRand.Intn(1000000))
		uniqueCodes[code] = true
		uniqueCodeArr = append(uniqueCodeArr, code)
	}
	return uniqueCodeArr
}

func IsBase64Image(input string) bool {
	if strings.HasPrefix(input, "data:image/") {
		return true
	}

	if len(input) > 100 && !strings.Contains(input, "http") && !strings.Contains(input, "://") {
		if len(input) > 50 {
			testStr := input[:50]
			_, err := base64.StdEncoding.DecodeString(testStr)
			return err == nil
		}
	}

	return false
}

func DecodeBase64Image(input string) ([]byte, error) {
	if strings.HasPrefix(input, "data:image/") {
		parts := strings.Split(input, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid data URL format")
		}
		return base64.StdEncoding.DecodeString(parts[1])
	}

	return base64.StdEncoding.DecodeString(input)
}

func DownloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func ExtractDomain(urlStr string) string {
	if !strings.Contains(urlStr, "://") {
		return urlStr
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	if parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" {
		return "localhost"
	}

	return parsed.Hostname()
}
