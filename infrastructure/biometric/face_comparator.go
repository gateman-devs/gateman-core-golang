package biometric

import (
	"encoding/base64"
	"fmt"
	"image"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

type FaceComparator struct {
	net         *gocv.Net
	initialized bool
	mu          sync.RWMutex
}

func NewFaceComparator() *FaceComparator {
	return &FaceComparator{}
}

func (fc *FaceComparator) Initialize(modelPath string) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.initialized {
		return nil
	}

	net := gocv.ReadNet(modelPath, "")
	if net.Empty() {
		net.Close() // Clean up if empty
		return fmt.Errorf("failed to load ArcFace model from %s", modelPath)
	}

	fc.net = &net
	fc.initialized = true
	return nil
}

func (fc *FaceComparator) Compare(input1, input2 string, threshold float64) FaceComparisonResult {
	start := time.Now()

	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if !fc.initialized {
		return FaceComparisonResult{
			Success:     false,
			Error:       "face comparator not initialized",
			ProcessTime: time.Since(start).Milliseconds(),
			Threshold:   threshold,
		}
	}

	// Load both images concurrently
	done1 := make(chan struct {
		img gocv.Mat
		err error
	})
	done2 := make(chan struct {
		img gocv.Mat
		err error
	})

	go func() {
		img, err := fc.loadImage(input1)
		done1 <- struct {
			img gocv.Mat
			err error
		}{img, err}
	}()

	go func() {
		img, err := fc.loadImage(input2)
		done2 <- struct {
			img gocv.Mat
			err error
		}{img, err}
	}()

	result1 := <-done1
	result2 := <-done2

	if result1.err != nil {
		if !result1.img.Empty() {
			result1.img.Close()
		}
		if !result2.img.Empty() {
			result2.img.Close()
		}
		return FaceComparisonResult{
			Success:     false,
			Error:       fmt.Sprintf("failed to load first image: %v", result1.err),
			ProcessTime: time.Since(start).Milliseconds(),
			Threshold:   threshold,
		}
	}

	if result2.err != nil {
		result1.img.Close()
		if !result2.img.Empty() {
			result2.img.Close()
		}
		return FaceComparisonResult{
			Success:     false,
			Error:       fmt.Sprintf("failed to load second image: %v", result2.err),
			ProcessTime: time.Since(start).Milliseconds(),
			Threshold:   threshold,
		}
	}

	defer result1.img.Close()
	defer result2.img.Close()

	// Extract features from both images
	features1, err := fc.extractFeatures(result1.img)
	if err != nil {
		return FaceComparisonResult{
			Success:     false,
			Error:       fmt.Sprintf("failed to extract features from first image: %v", err),
			ProcessTime: time.Since(start).Milliseconds(),
			Threshold:   threshold,
		}
	}
	defer features1.Close()

	features2, err := fc.extractFeatures(result2.img)
	if err != nil {
		return FaceComparisonResult{
			Success:     false,
			Error:       fmt.Sprintf("failed to extract features from second image: %v", err),
			ProcessTime: time.Since(start).Milliseconds(),
			Threshold:   threshold,
		}
	}
	defer features2.Close()

	// Calculate similarity
	similarity := fc.calculateCosineSimilarity(features1, features2)
	isMatch := similarity >= threshold

	return FaceComparisonResult{
		Success:     true,
		Match:       isMatch,
		Similarity:  similarity,
		Threshold:   threshold,
		ProcessTime: time.Since(start).Milliseconds(),
		Metadata:    fc.generateMetadata(result1.img, result2.img, similarity, threshold),
	}
}

func (fc *FaceComparator) extractFeatures(img gocv.Mat) (gocv.Mat, error) {
	if img.Empty() {
		return gocv.NewMat(), fmt.Errorf("input image is empty")
	}

	// Preprocess image for ArcFace (112x112, normalized)
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(img, &resized, image.Pt(112, 112), 0, 0, gocv.InterpolationLinear)

	// Convert to float and normalize
	normalized := gocv.NewMat()
	defer normalized.Close()
	resized.ConvertTo(&normalized, gocv.MatTypeCV32F)

	// Subtract mean
	meanScalar := gocv.NewMatWithSize(normalized.Rows(), normalized.Cols(), gocv.MatTypeCV32F)
	defer meanScalar.Close()
	meanScalar.SetTo(gocv.NewScalar(127.5, 127.5, 127.5, 0))
	gocv.Subtract(normalized, meanScalar, &normalized)

	// Scale
	normalized.MultiplyFloat(1.0 / 128.0)

	// Create blob from image
	blob := gocv.BlobFromImage(normalized, 1.0, image.Pt(112, 112), gocv.NewScalar(0, 0, 0, 0), false, false)
	defer blob.Close()

	// Set input and run forward pass
	fc.net.SetInput(blob, "")
	output := fc.net.Forward("")

	// Normalize features (L2 normalization)
	features := fc.normalizeFeatures(output)

	return features, nil
}

func (fc *FaceComparator) normalizeFeatures(features gocv.Mat) gocv.Mat {
	// Calculate L2 norm
	norm := gocv.Norm(features, gocv.NormL2)

	// Normalize features
	normalized := gocv.NewMat()
	features.DivideFloat(float32(norm))
	features.CopyTo(&normalized)

	return normalized
}

func (fc *FaceComparator) calculateCosineSimilarity(features1, features2 gocv.Mat) float64 {
	// Cosine similarity = dot product of normalized vectors
	dotProduct := 0.0

	for i := 0; i < features1.Cols(); i++ {
		val1 := features1.GetFloatAt(0, i)
		val2 := features2.GetFloatAt(0, i)
		dotProduct += float64(val1 * val2)
	}

	// Since features are already normalized, dot product = cosine similarity
	// Convert to similarity score (0-1 range)
	similarity := (dotProduct + 1.0) / 2.0

	return math.Max(0.0, math.Min(1.0, similarity))
}

func (fc *FaceComparator) loadImage(input string) (gocv.Mat, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return fc.loadImageFromURL(input)
	}
	return fc.loadImageFromBase64(input)
}

func (fc *FaceComparator) loadImageFromURL(url string) (gocv.Mat, error) {
	resp, err := http.Get(url)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return gocv.NewMat(), fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	buf := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(buf)
	if err != nil {
		return gocv.NewMat(), err
	}

	return gocv.IMDecode(buf, gocv.IMReadColor)
}

func (fc *FaceComparator) loadImageFromBase64(data string) (gocv.Mat, error) {
	// Remove data URL prefix if present
	if idx := strings.Index(data, ","); idx != -1 {
		data = data[idx+1:]
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to decode base64: %v", err)
	}

	return gocv.IMDecode(decoded, gocv.IMReadColor)
}

func (fc *FaceComparator) generateMetadata(img1, img2 gocv.Mat, similarity, threshold float64) ComparisonMetadata {
	metadata := ComparisonMetadata{
		Face1Quality: fc.assessImageQuality(img1),
		Face2Quality: fc.assessImageQuality(img2),
	}

	// Determine match confidence
	if similarity >= threshold+0.1 {
		metadata.MatchConfidence = "high"
	} else if similarity >= threshold {
		metadata.MatchConfidence = "medium"
	} else if similarity >= threshold-0.1 {
		metadata.MatchConfidence = "low"
	} else {
		metadata.MatchConfidence = "very_low"
	}

	// Add warnings
	if similarity < 0.3 {
		metadata.Warnings = append(metadata.Warnings, "Very low similarity score - images likely from different people")
	} else if similarity > 0.9 {
		metadata.Warnings = append(metadata.Warnings, "Very high similarity - possible identical images")
	}

	if similarity >= threshold-0.05 && similarity <= threshold+0.05 {
		metadata.Warnings = append(metadata.Warnings, "Similarity score is close to threshold - manual review recommended")
	}

	return metadata
}

func (fc *FaceComparator) assessImageQuality(img gocv.Mat) string {
	if img.Empty() {
		return "invalid"
	}

	// Basic quality assessment based on image properties
	size := img.Cols() * img.Rows()

	if size < 10000 { // Less than 100x100
		return "poor"
	} else if size < 40000 { // Less than 200x200
		return "fair"
	} else if size < 160000 { // Less than 400x400
		return "good"
	} else {
		return "excellent"
	}
}

func (fc *FaceComparator) Close() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.net != nil {
		fc.net.Close()
		fc.net = nil
	}
	fc.initialized = false
}
