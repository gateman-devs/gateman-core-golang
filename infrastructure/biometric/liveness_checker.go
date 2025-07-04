package biometric

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

type LivenessChecker struct {
	net         *gocv.Net
	initialized bool
	hasModel    bool
	mu          sync.RWMutex
}

type LivenessTest struct {
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Passed  bool    `json:"passed"`
	Details string  `json:"details"`
}

func NewLivenessChecker() *LivenessChecker {
	return &LivenessChecker{}
}

func (lc *LivenessChecker) Initialize(modelPath string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.initialized {
		return nil
	}

	// Check if the actual model file exists (not just placeholder)
	if info, err := os.Stat(modelPath); err == nil && info.Size() > 1000 {
		net := gocv.ReadNet(modelPath, "")
		if !net.Empty() {
			lc.net = &net
			lc.hasModel = true
		} else {
			net.Close() // Clean up if empty
		}
	}

	lc.initialized = true
	return nil
}

func (lc *LivenessChecker) CheckLiveness(input string) LivenessCheckResult {
	start := time.Now()

	lc.mu.RLock()
	defer lc.mu.RUnlock()

	if !lc.initialized {
		return LivenessCheckResult{
			Success:     false,
			Error:       "liveness checker not initialized",
			ProcessTime: time.Since(start).Milliseconds(),
			Threshold:   LIVENESS_THRESHOLD,
		}
	}

	img, err := lc.loadImage(input)
	if err != nil {
		return LivenessCheckResult{
			Success:     false,
			Error:       fmt.Sprintf("failed to load image: %v", err),
			ProcessTime: time.Since(start).Milliseconds(),
			Threshold:   LIVENESS_THRESHOLD,
		}
	}
	defer img.Close()

	if img.Empty() {
		return LivenessCheckResult{
			Success:     false,
			Error:       "loaded image is empty",
			ProcessTime: time.Since(start).Milliseconds(),
			Threshold:   LIVENESS_THRESHOLD,
		}
	}

	// Perform liveness tests
	tests := lc.performLivenessTests(img)

	// Calculate overall liveness score
	totalScore := 0.0
	passedTests := 0
	testNames := make([]string, len(tests))

	for i, test := range tests {
		totalScore += test.Score
		testNames[i] = test.Name
		if test.Passed {
			passedTests++
		}
	}

	avgScore := totalScore / float64(len(tests))
	isLive := avgScore >= LIVENESS_THRESHOLD && float64(passedTests)/float64(len(tests)) >= 0.6

	return LivenessCheckResult{
		Success:       true,
		IsLive:        isLive,
		LivenessScore: avgScore,
		Threshold:     LIVENESS_THRESHOLD,
		ProcessTime:   time.Since(start).Milliseconds(),
		Metadata:      lc.generateMetadata(img, tests, avgScore),
	}
}

func (lc *LivenessChecker) performLivenessTests(img gocv.Mat) []LivenessTest {
	var tests []LivenessTest

	// Test 1: Color Distribution Analysis
	colorTest := lc.analyzeColorDistribution(img)
	tests = append(tests, colorTest)

	// Test 2: Texture Analysis (LBP - Local Binary Patterns)
	textureTest := lc.analyzeTexture(img)
	tests = append(tests, textureTest)

	// Test 3: Edge Analysis
	edgeTest := lc.analyzeEdges(img)
	tests = append(tests, edgeTest)

	// Test 4: Frequency Domain Analysis
	frequencyTest := lc.analyzeFrequencyDomain(img)
	tests = append(tests, frequencyTest)

	// Test 5: Model-based detection (if available)
	if lc.hasModel && lc.net != nil {
		modelTest := lc.modelBasedDetection(img)
		tests = append(tests, modelTest)
	}

	// Test 6: Illumination Analysis
	illuminationTest := lc.analyzeIllumination(img)
	tests = append(tests, illuminationTest)

	return tests
}

func (lc *LivenessChecker) analyzeColorDistribution(img gocv.Mat) LivenessTest {
	// Convert to different color spaces for analysis
	hsv := gocv.NewMat()
	lab := gocv.NewMat()
	defer hsv.Close()
	defer lab.Close()

	gocv.CvtColor(img, &hsv, gocv.ColorBGRToHSV)
	gocv.CvtColor(img, &lab, gocv.ColorBGRToLab)

	// Calculate color variance in different channels
	channels := gocv.Split(hsv)
	var variances []float64

	for _, channel := range channels {
		mean := gocv.NewMat()
		stddev := gocv.NewMat()
		gocv.MeanStdDev(channel, &mean, &stddev)

		variance := stddev.GetDoubleAt(0, 0)
		variances = append(variances, variance)

		mean.Close()
		stddev.Close()
		channel.Close()
	}

	// Calculate overall color variance score
	totalVariance := variances[0] + variances[1] + variances[2]

	// Live faces typically have more color variation than printed photos
	score := math.Min(totalVariance/100.0, 1.0)
	passed := score > 0.4

	return LivenessTest{
		Name:    "color_distribution",
		Score:   score,
		Passed:  passed,
		Details: fmt.Sprintf("Color variance: %.2f", totalVariance),
	}
}

func (lc *LivenessChecker) analyzeTexture(img gocv.Mat) LivenessTest {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Apply Gaussian blur to reduce noise
	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(3, 3), 0, 0, gocv.BorderDefault)

	// Calculate local standard deviation (texture measure)
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(9, 9))
	defer kernel.Close()

	mean := gocv.NewMat()
	sqrMean := gocv.NewMat()
	defer mean.Close()
	defer sqrMean.Close()

	// Calculate local mean
	gocv.Filter2D(blurred, &mean, -1, kernel, image.Pt(-1, -1), 0, gocv.BorderDefault)

	// Calculate squared values
	squared := gocv.NewMat()
	defer squared.Close()
	gocv.Multiply(blurred, blurred, &squared)

	// Calculate local mean of squared values
	gocv.Filter2D(squared, &sqrMean, -1, kernel, image.Pt(-1, -1), 0, gocv.BorderDefault)

	// Calculate variance = E[X²] - (E[X])²
	meanSquared := gocv.NewMat()
	defer meanSquared.Close()
	gocv.Multiply(mean, mean, &meanSquared)

	variance := gocv.NewMat()
	defer variance.Close()
	gocv.Subtract(sqrMean, meanSquared, &variance)

	// Calculate average texture score
	meanResult := gocv.NewMat()
	defer meanResult.Close()
	gocv.Reduce(variance, &meanResult, 0, gocv.ReduceAvg, gocv.MatTypeCV64F)
	avgVariance := meanResult.GetDoubleAt(0, 0)
	textureScore := math.Min(avgVariance/1000.0, 1.0)

	// Live faces typically have more texture variation
	passed := textureScore > 0.3

	return LivenessTest{
		Name:    "texture_analysis",
		Score:   textureScore,
		Passed:  passed,
		Details: fmt.Sprintf("Average texture variance: %.2f", avgVariance),
	}
}

func (lc *LivenessChecker) analyzeEdges(img gocv.Mat) LivenessTest {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Apply Canny edge detection
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 50, 150)

	// Count edge pixels
	edgePixels := gocv.CountNonZero(edges)
	totalPixels := edges.Rows() * edges.Cols()
	edgeRatio := float64(edgePixels) / float64(totalPixels)

	// Live faces typically have more natural edge distribution
	score := math.Min(edgeRatio*10.0, 1.0)
	passed := score > 0.2 && score < 0.8 // Too few or too many edges might indicate spoofing

	return LivenessTest{
		Name:    "edge_analysis",
		Score:   score,
		Passed:  passed,
		Details: fmt.Sprintf("Edge ratio: %.4f", edgeRatio),
	}
}

func (lc *LivenessChecker) analyzeFrequencyDomain(img gocv.Mat) LivenessTest {
	// Convert to grayscale and float
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	grayFloat := gocv.NewMat()
	defer grayFloat.Close()
	gray.ConvertTo(&grayFloat, gocv.MatTypeCV32F)

	// Apply DFT
	complexImg := gocv.NewMat()
	defer complexImg.Close()
	gocv.DFT(grayFloat, &complexImg, gocv.DftComplexOutput)

	// Calculate magnitude spectrum
	planes := gocv.Split(complexImg)
	defer func() {
		for _, plane := range planes {
			plane.Close()
		}
	}()

	magnitude := gocv.NewMat()
	defer magnitude.Close()
	gocv.Magnitude(planes[0], planes[1], &magnitude)

	// Add 1 to avoid log(0)
	magnitude.AddFloat(1.0)

	// Apply log transform
	gocv.Log(magnitude, &magnitude)

	// Calculate high frequency content
	center := image.Pt(magnitude.Cols()/2, magnitude.Rows()/2)
	mask := gocv.NewMatWithSize(magnitude.Rows(), magnitude.Cols(), gocv.MatTypeCV8UC1)
	defer mask.Close()
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Create circular mask for high frequencies
	radius := int(math.Min(float64(center.X), float64(center.Y)) * 0.7)
	white := color.RGBA{255, 255, 255, 255}
	gocv.Circle(&mask, center, radius, white, -1)

	// Invert mask to get high frequencies
	gocv.BitwiseNot(mask, &mask)

	// Calculate mean of high frequency components
	meanResult := gocv.NewMat()
	defer meanResult.Close()
	gocv.Reduce(magnitude, &meanResult, 0, gocv.ReduceAvg, gocv.MatTypeCV64F)
	highFreqMean := meanResult.GetDoubleAt(0, 0)

	// Live faces typically have more high frequency content
	score := math.Min(highFreqMean/10.0, 1.0)
	passed := score > 0.3

	return LivenessTest{
		Name:    "frequency_analysis",
		Score:   score,
		Passed:  passed,
		Details: fmt.Sprintf("High frequency content: %.2f", highFreqMean),
	}
}

func (lc *LivenessChecker) modelBasedDetection(img gocv.Mat) LivenessTest {
	// Preprocess image for the model
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(img, &resized, image.Pt(80, 80), 0, 0, gocv.InterpolationLinear)

	// Create blob from image
	blob := gocv.BlobFromImage(resized, 1.0/255.0, image.Pt(80, 80), gocv.NewScalar(0, 0, 0, 0), false, false)
	defer blob.Close()

	// Set input and run forward pass
	lc.net.SetInput(blob, "")
	output := lc.net.Forward("")
	defer output.Close()

	// Get liveness score from model output
	livenessScore := output.GetFloatAt(0, 1) // Assuming second output is liveness score

	passed := livenessScore > 0.7

	return LivenessTest{
		Name:    "model_based_detection",
		Score:   float64(livenessScore),
		Passed:  passed,
		Details: fmt.Sprintf("Model confidence: %.3f", livenessScore),
	}
}

func (lc *LivenessChecker) analyzeIllumination(img gocv.Mat) LivenessTest {
	// Convert to LAB color space for better illumination analysis
	lab := gocv.NewMat()
	defer lab.Close()
	gocv.CvtColor(img, &lab, gocv.ColorBGRToLab)

	// Split channels (L = lightness)
	channels := gocv.Split(lab)
	defer func() {
		for _, channel := range channels {
			channel.Close()
		}
	}()

	lightness := channels[0]

	// Calculate illumination statistics
	mean := gocv.NewMat()
	stddev := gocv.NewMat()
	defer mean.Close()
	defer stddev.Close()

	gocv.MeanStdDev(lightness, &mean, &stddev)

	avgLightness := mean.GetDoubleAt(0, 0)
	lightnessStdDev := stddev.GetDoubleAt(0, 0)

	// Good illumination should have reasonable mean and variation
	illuminationScore := 1.0

	// Penalize too dark or too bright images
	if avgLightness < 30 || avgLightness > 200 {
		illuminationScore *= 0.5
	}

	// Penalize too uniform illumination (might be artificial)
	if lightnessStdDev < 10 {
		illuminationScore *= 0.7
	}

	// Penalize too variable illumination (might be poor quality)
	if lightnessStdDev > 80 {
		illuminationScore *= 0.8
	}

	passed := illuminationScore > 0.6

	return LivenessTest{
		Name:    "illumination_analysis",
		Score:   illuminationScore,
		Passed:  passed,
		Details: fmt.Sprintf("Mean: %.1f, StdDev: %.1f", avgLightness, lightnessStdDev),
	}
}

func (lc *LivenessChecker) loadImage(input string) (gocv.Mat, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return lc.loadImageFromURL(input)
	}
	return lc.loadImageFromBase64(input)
}

func (lc *LivenessChecker) loadImageFromURL(url string) (gocv.Mat, error) {
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

func (lc *LivenessChecker) loadImageFromBase64(data string) (gocv.Mat, error) {
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

func (lc *LivenessChecker) generateMetadata(img gocv.Mat, tests []LivenessTest, overallScore float64) LivenessMetadata {
	metadata := LivenessMetadata{
		ImageQuality:      lc.assessImageQuality(img),
		AntiSpoofingTests: make([]string, len(tests)),
	}

	// Collect test names
	for i, test := range tests {
		metadata.AntiSpoofingTests[i] = test.Name
	}

	// Determine confidence level
	if overallScore >= 0.8 {
		metadata.Confidence = "high"
	} else if overallScore >= 0.6 {
		metadata.Confidence = "medium"
	} else if overallScore >= 0.4 {
		metadata.Confidence = "low"
	} else {
		metadata.Confidence = "very_low"
	}

	// Add warnings based on test results
	for _, test := range tests {
		if !test.Passed {
			metadata.Warnings = append(metadata.Warnings, fmt.Sprintf("Failed %s test", test.Name))
		}
	}

	if overallScore < 0.5 {
		metadata.Warnings = append(metadata.Warnings, "Low overall liveness score - possible spoofing attempt")
	}

	if !lc.hasModel {
		metadata.Warnings = append(metadata.Warnings, "Model-based detection unavailable - using CV techniques only")
	}

	return metadata
}

func (lc *LivenessChecker) assessImageQuality(img gocv.Mat) string {
	if img.Empty() {
		return "invalid"
	}

	// Basic quality assessment
	size := img.Cols() * img.Rows()

	if size < 10000 {
		return "poor"
	} else if size < 40000 {
		return "fair"
	} else if size < 160000 {
		return "good"
	} else {
		return "excellent"
	}
}

func (lc *LivenessChecker) Close() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.net != nil {
		lc.net.Close()
		lc.net = nil
	}
	lc.initialized = false
	lc.hasModel = false
}
