package biometric

import (
	"encoding/base64"
	"os"
	"testing"
)

// Simple test image (1x1 pixel PNG encoded in base64)
const testImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAGA4nRzgwAAAABJRU5ErkJggg=="

func TestBiometricSystemInitialization(t *testing.T) {
	// Test system creation
	system := NewBiometricSystem()
	if system == nil {
		t.Fatal("Failed to create biometric system")
	}

	// Test individual component creation
	if system.faceDetector == nil {
		t.Error("Face detector not created")
	}
	if system.faceComparator == nil {
		t.Error("Face comparator not created")
	}
	if system.livenessChecker == nil {
		t.Error("Liveness checker not created")
	}

	// Test system initialization with default paths
	err := system.Initialize(DEFAULT_YUNET_MODEL_PATH, DEFAULT_ARCFACE_MODEL_PATH, DEFAULT_ANTISPOOFING_MODEL_PATH)
	if err != nil {
		t.Logf("Initialization warning (expected if models not available): %v", err)
	}

	// Clean up
	system.Close()
}

func TestFaceDetectorWithValidModel(t *testing.T) {
	// Check if YuNet model exists
	if _, err := os.Stat(DEFAULT_YUNET_MODEL_PATH); os.IsNotExist(err) {
		t.Skip("YuNet model not found, skipping face detection test")
	}

	detector := NewFaceDetector()
	err := detector.Initialize(DEFAULT_YUNET_MODEL_PATH)
	if err != nil {
		t.Fatalf("Failed to initialize face detector: %v", err)
	}
	defer detector.Close()

	// Test with base64 input
	result := detector.DetectFace(testImageBase64)
	if !result.Success {
		t.Logf("Face detection expected to fail with test image: %s", result.Error)
	}

	// Verify result structure
	if result.ProcessTime <= 0 {
		t.Error("Process time should be positive")
	}
	if result.Metadata.ImageSize == "" {
		t.Error("Image size metadata should not be empty")
	}
}

func TestFaceComparatorWithValidModel(t *testing.T) {
	// Check if ArcFace model exists
	if _, err := os.Stat(DEFAULT_ARCFACE_MODEL_PATH); os.IsNotExist(err) {
		t.Skip("ArcFace model not found, skipping face comparison test")
	}

	comparator := NewFaceComparator()
	err := comparator.Initialize(DEFAULT_ARCFACE_MODEL_PATH)
	if err != nil {
		t.Fatalf("Failed to initialize face comparator: %v", err)
	}
	defer comparator.Close()

	// Test with identical images
	result := comparator.Compare(testImageBase64, testImageBase64, 0.7)
	if !result.Success {
		t.Logf("Face comparison expected to fail with test image: %s", result.Error)
	}

	// Verify result structure
	if result.ProcessTime <= 0 {
		t.Error("Process time should be positive")
	}
	if result.Threshold != 0.7 {
		t.Error("Threshold should match input")
	}
}

func TestLivenessChecker(t *testing.T) {
	checker := NewLivenessChecker()
	err := checker.Initialize(DEFAULT_ANTISPOOFING_MODEL_PATH)
	if err != nil {
		t.Logf("Liveness checker initialization warning: %v", err)
	}
	defer checker.Close()

	// Test with base64 input
	result := checker.CheckLiveness(testImageBase64)
	if !result.Success {
		t.Logf("Liveness check expected to fail with test image: %s", result.Error)
	}

	// Verify result structure
	if result.ProcessTime <= 0 {
		t.Error("Process time should be positive")
	}
	if result.Threshold != LIVENESS_THRESHOLD {
		t.Error("Threshold should match constant")
	}
}

func TestFullBiometricVerification(t *testing.T) {
	// Initialize global system
	err := InitializeBiometricSystem()
	if err != nil {
		t.Logf("System initialization warning: %v", err)
	}
	defer GlobalBiometricSystem.Close()

	// Test full verification
	result := GlobalBiometricSystem.VerifyBiometric(testImageBase64, testImageBase64)

	// Verify result structure
	if result.TotalProcessTime <= 0 {
		t.Error("Total process time should be positive")
	}

	// Check if liveness check was performed
	if result.LivenessCheck.ProcessTime <= 0 {
		t.Error("Liveness check process time should be positive")
	}

	// Check if face comparison was performed (only if liveness passed)
	if result.LivenessCheck.IsLive && result.FaceComparison.ProcessTime <= 0 {
		t.Error("Face comparison process time should be positive when liveness passes")
	}

	t.Logf("Biometric verification result: Success=%v, OverallMatch=%v", result.Success, result.OverallMatch)
	t.Logf("Liveness: IsLive=%v, Score=%.3f", result.LivenessCheck.IsLive, result.LivenessCheck.LivenessScore)
	t.Logf("Comparison: Match=%v, Similarity=%.3f", result.FaceComparison.Match, result.FaceComparison.Similarity)

	for _, rec := range result.Recommendations {
		t.Logf("Recommendation: %s", rec)
	}
}

func TestImageLoadingFromBase64(t *testing.T) {
	// Test base64 decoding
	data, err := base64.StdEncoding.DecodeString(testImageBase64)
	if err != nil {
		t.Fatalf("Failed to decode test image: %v", err)
	}

	if len(data) == 0 {
		t.Error("Decoded image data should not be empty")
	}
}

func TestModelFilesExistence(t *testing.T) {
	models := map[string]string{
		"YuNet":                DEFAULT_YUNET_MODEL_PATH,
		"ArcFace":              DEFAULT_ARCFACE_MODEL_PATH,
		"Silent Anti-Spoofing": DEFAULT_ANTISPOOFING_MODEL_PATH,
	}

	for name, path := range models {
		if info, err := os.Stat(path); os.IsNotExist(err) {
			t.Logf("Model %s not found at %s", name, path)
		} else if err != nil {
			t.Logf("Error checking model %s: %v", name, err)
		} else {
			t.Logf("Model %s found: %s (%d bytes)", name, path, info.Size())
		}
	}
}

// Benchmark tests
func BenchmarkFaceDetection(b *testing.B) {
	if _, err := os.Stat(DEFAULT_YUNET_MODEL_PATH); os.IsNotExist(err) {
		b.Skip("YuNet model not found")
	}

	detector := NewFaceDetector()
	err := detector.Initialize(DEFAULT_YUNET_MODEL_PATH)
	if err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}
	defer detector.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectFace(testImageBase64)
	}
}

func BenchmarkLivenessCheck(b *testing.B) {
	checker := NewLivenessChecker()
	checker.Initialize(DEFAULT_ANTISPOOFING_MODEL_PATH)
	defer checker.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.CheckLiveness(testImageBase64)
	}
}
