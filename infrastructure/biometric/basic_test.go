package biometric

import (
	"testing"
)

func TestBasicSystemCreation(t *testing.T) {
	// Test basic system creation without initialization
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

	// Test that system is not initialized yet
	if system.initialized {
		t.Error("System should not be initialized without calling Initialize()")
	}

	t.Log("Basic biometric system creation test passed")
}

func TestConstantsAndPaths(t *testing.T) {
	// Test that constants are defined
	if DEFAULT_YUNET_MODEL_PATH == "" {
		t.Error("DEFAULT_YUNET_MODEL_PATH should not be empty")
	}
	if DEFAULT_ARCFACE_MODEL_PATH == "" {
		t.Error("DEFAULT_ARCFACE_MODEL_PATH should not be empty")
	}
	if DEFAULT_ANTISPOOFING_MODEL_PATH == "" {
		t.Error("DEFAULT_ANTISPOOFING_MODEL_PATH should not be empty")
	}

	// Test threshold constants
	if FACE_COMPARISON_THRESHOLD <= 0 || FACE_COMPARISON_THRESHOLD >= 1 {
		t.Error("FACE_COMPARISON_THRESHOLD should be between 0 and 1")
	}
	if LIVENESS_THRESHOLD <= 0 || LIVENESS_THRESHOLD >= 1 {
		t.Error("LIVENESS_THRESHOLD should be between 0 and 1")
	}

	t.Logf("YuNet model path: %s", DEFAULT_YUNET_MODEL_PATH)
	t.Logf("ArcFace model path: %s", DEFAULT_ARCFACE_MODEL_PATH)
	t.Logf("Anti-spoofing model path: %s", DEFAULT_ANTISPOOFING_MODEL_PATH)
	t.Logf("Face comparison threshold: %.2f", FACE_COMPARISON_THRESHOLD)
	t.Logf("Liveness threshold: %.2f", LIVENESS_THRESHOLD)
}

func TestUninitializedSystemBehavior(t *testing.T) {
	system := NewBiometricSystem()

	// Test that uninitialized system returns proper errors
	faceResult := system.ExtractFace("test")
	if faceResult.Success {
		t.Error("Face extraction should fail on uninitialized system")
	}
	if faceResult.Error != "biometric system not initialized" {
		t.Errorf("Expected 'biometric system not initialized' error, got: %s", faceResult.Error)
	}

	compResult := system.CompareFaces("test1", "test2")
	if compResult.Success {
		t.Error("Face comparison should fail on uninitialized system")
	}
	if compResult.Error != "biometric system not initialized" {
		t.Errorf("Expected 'biometric system not initialized' error, got: %s", compResult.Error)
	}

	livenessResult := system.CheckLiveness("test")
	if livenessResult.Success {
		t.Error("Liveness check should fail on uninitialized system")
	}
	if livenessResult.Error != "biometric system not initialized" {
		t.Errorf("Expected 'biometric system not initialized' error, got: %s", livenessResult.Error)
	}

	bioResult := system.VerifyBiometric("test1", "test2")
	if bioResult.Success {
		t.Error("Biometric verification should fail on uninitialized system")
	}
	if bioResult.Error != "biometric system not initialized" {
		t.Errorf("Expected 'biometric system not initialized' error, got: %s", bioResult.Error)
	}

	t.Log("Uninitialized system behavior test passed")
}

func TestResultStructures(t *testing.T) {
	// Test FaceBox structure
	faceBox := FaceBox{
		X:      10,
		Y:      20,
		Width:  100,
		Height: 150,
		Score:  0.95,
	}
	if faceBox.X != 10 || faceBox.Y != 20 || faceBox.Width != 100 || faceBox.Height != 150 || faceBox.Score != 0.95 {
		t.Error("FaceBox structure not working correctly")
	}

	// Test metadata structures
	faceMetadata := FaceMetadata{
		ImageSize:   "640x480",
		FaceSize:    "100x150",
		Quality:     "good",
		Lighting:    "adequate",
		Orientation: "upright",
		Warnings:    []string{"test warning"},
	}
	if len(faceMetadata.Warnings) != 1 || faceMetadata.Quality != "good" {
		t.Error("FaceMetadata structure not working correctly")
	}

	// Test result structures
	faceResult := FaceDetectionResult{
		Success:     true,
		FaceFound:   true,
		FaceRegion:  faceBox,
		Confidence:  0.95,
		ProcessTime: 100,
		Metadata:    faceMetadata,
	}
	if !faceResult.Success || !faceResult.FaceFound || faceResult.Confidence != 0.95 {
		t.Error("FaceDetectionResult structure not working correctly")
	}

	t.Log("Result structures test passed")
}

func TestComponentCreation(t *testing.T) {
	// Test individual component creation
	detector := NewFaceDetector()
	if detector == nil {
		t.Error("Failed to create face detector")
	}
	if detector.initialized {
		t.Error("Face detector should not be initialized on creation")
	}

	comparator := NewFaceComparator()
	if comparator == nil {
		t.Error("Failed to create face comparator")
	}
	if comparator.initialized {
		t.Error("Face comparator should not be initialized on creation")
	}

	checker := NewLivenessChecker()
	if checker == nil {
		t.Error("Failed to create liveness checker")
	}
	if checker.initialized {
		t.Error("Liveness checker should not be initialized on creation")
	}

	t.Log("Component creation test passed")
}

func TestSystemCloseBehavior(t *testing.T) {
	system := NewBiometricSystem()

	// Close system without initialization should not crash
	system.Close()

	// System should be safe to close multiple times
	system.Close()
	system.Close()

	t.Log("System close behavior test passed")
}

func TestGlobalSystemAccess(t *testing.T) {
	// Test that GlobalBiometricSystem starts as nil
	if GlobalBiometricSystem != nil {
		t.Error("GlobalBiometricSystem should be nil initially")
	}

	// Test initialization function (will fail without models but shouldn't crash)
	err := InitializeBiometricSystem()
	if err == nil {
		t.Log("Biometric system initialized successfully (models available)")
		if GlobalBiometricSystem == nil {
			t.Error("GlobalBiometricSystem should not be nil after initialization")
		}
		GlobalBiometricSystem.Close()
	} else {
		t.Logf("Biometric system initialization failed as expected: %v", err)
		if GlobalBiometricSystem == nil {
			t.Error("GlobalBiometricSystem should be created even if initialization fails")
		}
	}

	t.Log("Global system access test passed")
}
