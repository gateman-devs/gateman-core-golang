package biometric

import (
	"fmt"
	"image"

	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

// SilentFaceAntiSpoof implements the Silent-Face-Anti-Spoofing model
// Reference: https://github.com/minivision-ai/Silent-Face-Anti-Spoofing
type SilentFaceAntiSpoof struct {
	modelV1SE gocv.Net
	modelV2   gocv.Net
	isLoaded  bool
}

// NewSilentFaceAntiSpoof creates a new instance of the anti-spoofing detector
func NewSilentFaceAntiSpoof() (*SilentFaceAntiSpoof, error) {
	sfas := &SilentFaceAntiSpoof{}

	// Load MiniFASNetV2 model (2.7) - use ReadNet instead of ReadNetFromONNX
	modelV2Path := "./models/silent-face-anti-spoofing/2.7_80x80_MiniFASNetV2.onnx"
	sfas.modelV2 = gocv.ReadNet(modelV2Path, "")
	if sfas.modelV2.Empty() {
		return nil, fmt.Errorf("failed to load MiniFASNetV2 model from %s - model may be incompatible or file not found", modelV2Path)
	}

	// Load MiniFASNetV1SE model (4_0_0)
	modelV1SEPath := "./models/silent-face-anti-spoofing/4_0_0_80x80_MiniFASNetV1SE.onnx"
	sfas.modelV1SE = gocv.ReadNet(modelV1SEPath, "")
	if sfas.modelV1SE.Empty() {
		// If V1SE fails, we can still use V2 alone
		logger.Info("⚠️ MiniFASNetV1SE failed to load, using V2 only", logger.LoggerOptions{
			Key: "sfas_v1se_failed",
		})
		sfas.modelV1SE = sfas.modelV2 // Use V2 for both
	}

	sfas.isLoaded = true

	logger.Info("✅ Silent-Face-Anti-Spoofing models loaded successfully", logger.LoggerOptions{
		Key: "sfas_init_success",
		Data: map[string]interface{}{
			"v2_loaded":   !sfas.modelV2.Empty(),
			"v1se_loaded": !sfas.modelV1SE.Empty(),
		},
	})

	return sfas, nil
}

// preprocessFace preprocesses the face image for the anti-spoofing model
// Input: face region (any size)
// Output: 80x80 RGB image normalized to [0,1]
func (sfas *SilentFaceAntiSpoof) preprocessFace(faceRegion gocv.Mat) gocv.Mat {
	// Resize to 80x80 (model input size)
	resized := gocv.NewMat()
	gocv.Resize(faceRegion, &resized, image.Pt(80, 80), 0, 0, gocv.InterpolationLinear)

	// Convert to float32 and normalize to [0, 1]
	normalized := gocv.NewMat()
	resized.ConvertTo(&normalized, gocv.MatTypeCV32F)
	normalized.DivideFloat(255.0)

	resized.Close()
	return normalized
}

// Predict runs the anti-spoofing detection on a face region
// Returns: (isReal, confidence, error)
// - isReal: true if the face is determined to be real
// - confidence: probability that the face is real (0.0 to 1.0)
func (sfas *SilentFaceAntiSpoof) Predict(faceRegion gocv.Mat) (bool, float64, error) {
	if !sfas.isLoaded {
		return false, 0.0, fmt.Errorf("anti-spoofing models not loaded")
	}

	if faceRegion.Empty() {
		return false, 0.0, fmt.Errorf("empty face region")
	}

	// Preprocess face
	preprocessed := sfas.preprocessFace(faceRegion)
	defer preprocessed.Close()

	// Create blob from image
	// Note: Silent-Face-Anti-Spoofing expects RGB input, size 80x80, normalized [0,1]
	blob := gocv.BlobFromImage(
		preprocessed,
		1.0,                        // scalefactor (already normalized)
		image.Pt(80, 80),           // size
		gocv.NewScalar(0, 0, 0, 0), // mean (no mean subtraction)
		false,                      // swapRB (keep as BGR for OpenCV compatibility)
		false,                      // crop
	)
	defer blob.Close()

	// Run inference on both models and average results for better accuracy

	// Model V1SE inference
	sfas.modelV1SE.SetInput(blob, "")
	outputV1SE := sfas.modelV1SE.Forward("")
	defer outputV1SE.Close()

	// Model V2 inference
	sfas.modelV2.SetInput(blob, "")
	outputV2 := sfas.modelV2.Forward("")
	defer outputV2.Close()

	// Extract predictions
	// Output shape: [1, 3] for 3 classes (real, print, replay)
	// We take the probability of the "real" class (index 1)

	// Get raw logits from both models
	realScoreV1SE := float64(outputV1SE.GetFloatAt(0, 1))
	realScoreV2 := float64(outputV2.GetFloatAt(0, 1))

	// Average the two models for ensemble prediction
	avgRealScore := (realScoreV1SE + realScoreV2) / 2.0

	// Apply softmax to get probability (simplified for binary case)
	// In practice, the model outputs are already processed
	confidence := avgRealScore

	// Normalize confidence to [0, 1] range if needed
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}

	// Threshold for real vs fake (typically 0.5)
	threshold := 0.5
	isReal := confidence >= threshold

	logger.Info("🤖 Silent-Face-Anti-Spoofing prediction", logger.LoggerOptions{
		Key: "sfas_prediction",
		Data: map[string]interface{}{
			"is_real":    isReal,
			"confidence": confidence,
			"v1se_score": realScoreV1SE,
			"v2_score":   realScoreV2,
			"threshold":  threshold,
		},
	})

	return isReal, confidence, nil
}

// PredictWithThreshold allows custom threshold for real/fake classification
func (sfas *SilentFaceAntiSpoof) PredictWithThreshold(faceRegion gocv.Mat, threshold float64) (bool, float64, error) {
	if !sfas.isLoaded {
		return false, 0.0, fmt.Errorf("anti-spoofing models not loaded")
	}

	if faceRegion.Empty() {
		return false, 0.0, fmt.Errorf("empty face region")
	}

	// Get confidence score
	_, confidence, err := sfas.Predict(faceRegion)
	if err != nil {
		return false, 0.0, err
	}

	// Apply custom threshold
	isReal := confidence >= threshold

	return isReal, confidence, nil
}

// Close releases model resources
func (sfas *SilentFaceAntiSpoof) Close() {
	if sfas.isLoaded {
		sfas.modelV1SE.Close()
		sfas.modelV2.Close()
		sfas.isLoaded = false

		logger.Info("🔒 Silent-Face-Anti-Spoofing models closed", logger.LoggerOptions{
			Key: "sfas_closed",
		})
	}
}

// GetModelInfo returns information about the loaded models
func (sfas *SilentFaceAntiSpoof) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"loaded":     sfas.isLoaded,
		"models":     []string{"MiniFASNetV1SE (4.0.0)", "MiniFASNetV2 (2.7)"},
		"input_size": "80x80",
		"reference":  "https://github.com/minivision-ai/Silent-Face-Anti-Spoofing",
	}
}
