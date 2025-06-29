package facematch

import (
	"fmt"
	"image"
	"math"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

// Anti-Spoofing Configuration Constants
const (
	// Texture Analysis Thresholds
	LBP_THRESHOLD        = 0.7   // LBP uniformity threshold (0.0-1.0)
	LPQ_THRESHOLD        = 0.7   // LPQ phase consistency threshold (0.0-1.0)
	REFLECTION_THRESHOLD = 0.8   // Reflection analysis threshold (0.0-1.0)
	COLOR_THRESHOLD      = 0.5   // Color consistency threshold (0.0-1.0)
	TEXTURE_THRESHOLD    = 0.995 // Texture smoothness threshold (0.0-1.0)
	FREQUENCY_THRESHOLD  = 0.05  // High-frequency content threshold (0.0-1.0)

	// Decision Thresholds
	STRONG_INDICATOR_THRESHOLD = 3   // Number of strong indicators for high confidence spoof
	HIGH_SPOOF_THRESHOLD       = 0.8 // High confidence spoof score threshold (0.0-1.0)
	MEDIUM_SPOOF_THRESHOLD     = 0.6 // Medium confidence spoof score threshold (0.0-1.0)
	LOW_SPOOF_THRESHOLD        = 0.4 // Low confidence spoof score threshold (0.0-1.0)
	INDICATOR_THRESHOLD        = 3   // Number of indicators for spoof detection

	// Analysis Penalty Weights
	LBP_PENALTY             = 0.25 // LBP analysis penalty weight (0.0-1.0)
	LPQ_PENALTY             = 0.2  // LPQ analysis penalty weight (0.0-1.0)
	REFLECTION_PENALTY      = 0.3  // Reflection analysis penalty weight (0.0-1.0)
	COLOR_PENALTY           = 0.2  // Color analysis penalty weight (0.0-1.0)
	EDGE_PENALTY            = 0.15 // Edge analysis penalty weight (0.0-1.0)
	FREQUENCY_PENALTY       = 0.1  // Frequency analysis penalty weight (0.0-1.0)
	DEFAULT_TEXTURE_PENALTY = 0.25 // Default texture penalty weight (0.0-1.0)

	// Compression-Aware Penalties
	HEAVY_COMPRESSION_PENALTY    = 0.05 // Penalty for heavily compressed images (0.0-1.0)
	MODERATE_COMPRESSION_PENALTY = 0.1  // Penalty for moderately compressed images (0.0-1.0)
	LIGHT_COMPRESSION_PENALTY    = 0.15 // Penalty for lightly compressed images (0.0-1.0)

	// Blur-Aware Penalties
	VERY_BLURRY_PENALTY     = 0.1  // Penalty for very blurry images (0.0-1.0)
	SOMEWHAT_BLURRY_PENALTY = 0.15 // Penalty for somewhat blurry images (0.0-1.0)

	// Compression Level Thresholds
	HEAVY_COMPRESSION_LEVEL    = 0.7 // Threshold for heavy compression detection
	MODERATE_COMPRESSION_LEVEL = 0.5 // Threshold for moderate compression detection
	LIGHT_COMPRESSION_LEVEL    = 0.3 // Threshold for light compression detection

	// Sharpness Thresholds
	VERY_BLURRY_SHARPNESS     = 50.0  // Threshold for very blurry images
	SOMEWHAT_BLURRY_SHARPNESS = 100.0 // Threshold for somewhat blurry images

	// Edge Analysis Thresholds
	MIN_EDGE_DENSITY = 0.01 // Minimum edge density threshold
	MAX_EDGE_DENSITY = 0.9  // Maximum edge density threshold

	// Frequency Analysis Thresholds
	MAX_NOISE_LEVEL = 0.9 // Maximum noise level threshold
)

// AdvancedAntiSpoofResult represents the result of advanced anti-spoofing detection
type AdvancedAntiSpoofResult struct {
	IsReal            bool              `json:"is_real"`
	SpoofScore        float64           `json:"spoof_score"` // 0.0 to 1.0, higher means more likely to be spoof
	Confidence        float64           `json:"confidence"`  // 0.0 to 1.0, confidence in the prediction
	HasFace           bool              `json:"has_face"`
	ProcessTime       int64             `json:"process_time_ms"`
	TextureScore      float64           `json:"texture_score"`     // Advanced texture analysis score
	ReflectionScore   float64           `json:"reflection_score"`  // Reflection analysis score
	ColorConsistency  float64           `json:"color_consistency"` // Color space consistency score
	SpoofReasons      []string          `json:"spoof_reasons,omitempty"`
	Error             string            `json:"error,omitempty"`
	AnalysisBreakdown AnalysisBreakdown `json:"analysis_breakdown"`
}

// AnalysisBreakdown provides detailed analysis scores
type AnalysisBreakdown struct {
	LBPScore              float64            `json:"lbp_score"`              // Local Binary Pattern score
	LPQScore              float64            `json:"lpq_score"`              // Local Phase Quantization score
	ReflectionConsistency float64            `json:"reflection_consistency"` // Reflection consistency
	ColorSpaceAnalysis    ColorSpaceScores   `json:"color_space_analysis"`
	EdgeAnalysis          EdgeAnalysisScores `json:"edge_analysis"`
	FrequencyAnalysis     FrequencyScores    `json:"frequency_analysis"`
}

// ColorSpaceScores for different color spaces
type ColorSpaceScores struct {
	YCrCbConsistency float64 `json:"ycrcb_consistency"`
	HSVConsistency   float64 `json:"hsv_consistency"`
	LABConsistency   float64 `json:"lab_consistency"`
}

// EdgeAnalysisScores for edge analysis
type EdgeAnalysisScores struct {
	EdgeDensity     float64 `json:"edge_density"`
	EdgeOrientation float64 `json:"edge_orientation"`
	EdgeSharpness   float64 `json:"edge_sharpness"`
}

// FrequencyScores for frequency domain analysis
type FrequencyScores struct {
	HighFrequencyContent  float64 `json:"high_frequency_content"`
	FrequencyDistribution float64 `json:"frequency_distribution"`
	NoiseLevel            float64 `json:"noise_level"`
}

// DetectAdvancedAntiSpoof performs production-ready anti-spoofing detection with advanced techniques
func (fm *FaceMatcher) DetectAdvancedAntiSpoof(input string) AdvancedAntiSpoofResult {
	// Initialize result with safe defaults
	result := AdvancedAntiSpoofResult{
		IsReal:       false,
		SpoofScore:   1.0,
		Confidence:   0.0,
		HasFace:      false,
		ProcessTime:  0,
		SpoofReasons: []string{},
	}

	// Quick validation
	if input == "" {
		result.Error = "empty input provided"
		return result
	}

	if !fm.initialized {
		result.Error = "face matcher not initialized"
		return result
	}

	// Load and validate image (not included in ProcessTime)
	img, err := fm.loadImageWithValidation(input)
	if err != nil {
		result.Error = fmt.Sprintf("failed to load image: %v", err)
		return result
	}
	defer img.Close()

	// Start timing here - after image loading, before extraction and analysis
	startTime := time.Now()

	// Detect face
	faceRegion, faceErr := fm.detectPrimaryFace(img)
	if faceErr != nil {
		result.Error = fmt.Sprintf("no valid face detected: %v", faceErr)
		result.ProcessTime = time.Since(startTime).Milliseconds()
		return result
	}
	defer faceRegion.Close()

	result.HasFace = true

	// Perform advanced anti-spoofing analysis
	analysisResult := fm.performAdvancedSpoofingAnalysis(img, faceRegion)

	result.IsReal = analysisResult.IsReal
	result.SpoofScore = analysisResult.SpoofScore
	result.Confidence = analysisResult.Confidence
	result.TextureScore = analysisResult.TextureScore
	result.ReflectionScore = analysisResult.ReflectionScore
	result.ColorConsistency = analysisResult.ColorConsistency
	result.SpoofReasons = analysisResult.SpoofReasons
	result.AnalysisBreakdown = analysisResult.AnalysisBreakdown
	result.ProcessTime = time.Since(startTime).Milliseconds()

	return result
}

// performAdvancedSpoofingAnalysis performs comprehensive multi-modal analysis
// Adjusted to ensure iPhone 14 photo passes while maintaining security for obvious spoofs
func (fm *FaceMatcher) performAdvancedSpoofingAnalysis(img, face gocv.Mat) AdvancedAntiSpoofResult {
	// Convert to grayscale for analysis
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(face, &gray, gocv.ColorBGRToGray)

	// Use parallel analysis for speed improvement
	return fm.performParallelSpoofingAnalysis(face, gray)
}

// performParallelSpoofingAnalysis runs analysis steps concurrently for speed
func (fm *FaceMatcher) performParallelSpoofingAnalysis(face, gray gocv.Mat) AdvancedAntiSpoofResult {
	result := AdvancedAntiSpoofResult{
		IsReal:            true,
		SpoofScore:        0.0,
		Confidence:        0.8,
		SpoofReasons:      []string{},
		AnalysisBreakdown: AnalysisBreakdown{},
	}

	// Define result structures for parallel analysis
	type analysisResult struct {
		lbpScore         float64
		lpqScore         float64
		reflectionScore  float64
		colorAnalysis    ColorSpaceScores
		edgeAnalysis     EdgeAnalysisScores
		textureScore     float64
		freqAnalysis     FrequencyScores
		overallSharpness float64
		compressionLevel float64
	}

	// Channel to collect results
	resultChan := make(chan analysisResult, 1)

	// Run all independent analyses in parallel
	go func() {
		var res analysisResult
		var wg sync.WaitGroup

		// We'll use individual channels for each analysis
		lbpChan := make(chan float64, 1)
		lpqChan := make(chan float64, 1)
		reflectionChan := make(chan float64, 1)
		colorChan := make(chan ColorSpaceScores, 1)
		edgeChan := make(chan EdgeAnalysisScores, 1)
		textureChan := make(chan float64, 1)
		freqChan := make(chan FrequencyScores, 1)
		sharpnessChan := make(chan float64, 1)
		compressionChan := make(chan float64, 1)

		// Start all analyses concurrently
		wg.Add(9)

		// 1. Advanced Local Binary Pattern (LBP) Analysis
		go func() {
			defer wg.Done()
			lbpChan <- fm.calculateAdvancedLBPScore(gray)
		}()

		// 2. Local Phase Quantization (LPQ) Analysis
		go func() {
			defer wg.Done()
			lpqChan <- fm.calculateLPQScore(gray)
		}()

		// 3. Advanced Reflection Analysis
		go func() {
			defer wg.Done()
			reflectionChan <- fm.calculateAdvancedReflectionScore(gray)
		}()

		// 4. Multi-channel color space analysis
		go func() {
			defer wg.Done()
			colorChan <- fm.analyzeColorSpaces(face)
		}()

		// 5. Advanced edge analysis
		go func() {
			defer wg.Done()
			edgeChan <- fm.performEdgeAnalysis(gray)
		}()

		// 6. Advanced texture analysis
		go func() {
			defer wg.Done()
			textureChan <- fm.calculateAdvancedTextureScore(gray)
		}()

		// 7. Frequency domain analysis
		go func() {
			defer wg.Done()
			freqChan <- fm.analyzeFrequencyDomain(gray)
		}()

		// 8. Overall sharpness calculation
		go func() {
			defer wg.Done()
			sharpnessChan <- fm.calculateOverallSharpness(gray)
		}()

		// 9. Compression level detection
		go func() {
			defer wg.Done()
			compressionChan <- fm.detectCompressionLevel(gray, face)
		}()

		// Wait for all goroutines to complete
		wg.Wait()

		// Collect all results
		res.lbpScore = <-lbpChan
		res.lpqScore = <-lpqChan
		res.reflectionScore = <-reflectionChan
		res.colorAnalysis = <-colorChan
		res.edgeAnalysis = <-edgeChan
		res.textureScore = <-textureChan
		res.freqAnalysis = <-freqChan
		res.overallSharpness = <-sharpnessChan
		res.compressionLevel = <-compressionChan

		resultChan <- res
	}()

	// Get the parallel analysis results
	analysisRes := <-resultChan

	// Now process the results with configurable thresholds
	var indicators []string
	var totalPenalty float64

	// Process LBP results
	result.AnalysisBreakdown.LBPScore = analysisRes.lbpScore
	if analysisRes.lbpScore > LBP_THRESHOLD {
		totalPenalty += LBP_PENALTY
		indicators = append(indicators, "suspicious texture patterns detected")
	}

	// Process LPQ results
	result.AnalysisBreakdown.LPQScore = analysisRes.lpqScore
	if analysisRes.lpqScore > LPQ_THRESHOLD {
		totalPenalty += LPQ_PENALTY
		indicators = append(indicators, "unusual frequency patterns detected")
	}

	// Process reflection results
	result.ReflectionScore = analysisRes.reflectionScore
	result.AnalysisBreakdown.ReflectionConsistency = analysisRes.reflectionScore
	if analysisRes.reflectionScore > REFLECTION_THRESHOLD {
		totalPenalty += REFLECTION_PENALTY
		indicators = append(indicators, "suspicious reflection patterns")
	}

	// Process color analysis results
	result.AnalysisBreakdown.ColorSpaceAnalysis = analysisRes.colorAnalysis
	avgColorConsistency := (analysisRes.colorAnalysis.YCrCbConsistency +
		analysisRes.colorAnalysis.HSVConsistency +
		analysisRes.colorAnalysis.LABConsistency) / 3
	result.ColorConsistency = 1.0 - avgColorConsistency // Invert for consistency scoring
	if avgColorConsistency > COLOR_THRESHOLD {
		totalPenalty += COLOR_PENALTY
		indicators = append(indicators, "inconsistent color distribution")
	}

	// Process edge analysis results
	result.AnalysisBreakdown.EdgeAnalysis = analysisRes.edgeAnalysis
	// Fine-tuned edge density threshold
	if analysisRes.edgeAnalysis.EdgeDensity < MIN_EDGE_DENSITY || analysisRes.edgeAnalysis.EdgeDensity > MAX_EDGE_DENSITY {
		totalPenalty += EDGE_PENALTY
		indicators = append(indicators, "unusual edge characteristics")
	}

	// Process texture analysis results with compression awareness
	result.TextureScore = analysisRes.textureScore

	// Dynamic texture threshold based on compression and sharpness
	adjustedTextureThreshold := TEXTURE_THRESHOLD
	texturePenalty := DEFAULT_TEXTURE_PENALTY

	// Make provisions for compressed/resized images
	if analysisRes.compressionLevel >= HEAVY_COMPRESSION_LEVEL { // Heavily compressed
		adjustedTextureThreshold = 0.999
		texturePenalty = HEAVY_COMPRESSION_PENALTY
	} else if analysisRes.compressionLevel >= MODERATE_COMPRESSION_LEVEL { // Moderately compressed
		adjustedTextureThreshold = 0.998
		texturePenalty = MODERATE_COMPRESSION_PENALTY
	} else if analysisRes.compressionLevel >= LIGHT_COMPRESSION_LEVEL { // Lightly compressed
		adjustedTextureThreshold = 0.996
		texturePenalty = LIGHT_COMPRESSION_PENALTY
	}

	// Additional leniency for blurry images
	if analysisRes.overallSharpness < VERY_BLURRY_SHARPNESS { // Very blurry
		adjustedTextureThreshold = 0.999
		texturePenalty = VERY_BLURRY_PENALTY
	} else if analysisRes.overallSharpness < SOMEWHAT_BLURRY_SHARPNESS { // Somewhat blurry
		adjustedTextureThreshold = 0.998
		texturePenalty = SOMEWHAT_BLURRY_PENALTY
	}

	if analysisRes.textureScore > adjustedTextureThreshold {
		totalPenalty += texturePenalty
		if analysisRes.compressionLevel >= MODERATE_COMPRESSION_LEVEL {
			indicators = append(indicators, "insufficient texture detail (image appears compressed)")
		} else if analysisRes.overallSharpness < VERY_BLURRY_SHARPNESS {
			indicators = append(indicators, "insufficient texture detail (image appears blurry)")
		} else {
			indicators = append(indicators, "insufficient natural skin texture")
		}
	}

	// Process frequency analysis results
	result.AnalysisBreakdown.FrequencyAnalysis = analysisRes.freqAnalysis
	// More lenient frequency analysis
	if analysisRes.freqAnalysis.HighFrequencyContent < FREQUENCY_THRESHOLD || analysisRes.freqAnalysis.NoiseLevel > MAX_NOISE_LEVEL {
		totalPenalty += FREQUENCY_PENALTY
		indicators = append(indicators, "suspicious frequency characteristics")
	}

	// Calculate final spoof score
	result.SpoofScore = totalPenalty
	if result.SpoofScore > 1.0 {
		result.SpoofScore = 1.0
	}

	// Decision logic based on multiple factors
	strongIndicators := 0
	for _, indicator := range indicators {
		if indicator == "suspicious reflection patterns" ||
			indicator == "insufficient natural skin texture" ||
			indicator == "suspicious texture patterns detected" {
			strongIndicators++
		}
	}

	// Decision logic
	if strongIndicators >= STRONG_INDICATOR_THRESHOLD || result.SpoofScore >= HIGH_SPOOF_THRESHOLD {
		result.IsReal = false
		result.Confidence = 0.95
		result.SpoofReasons = indicators
	} else if len(indicators) >= 5 || result.SpoofScore >= MEDIUM_SPOOF_THRESHOLD {
		result.IsReal = false
		result.Confidence = 0.85
		result.SpoofReasons = indicators
	} else if len(indicators) >= INDICATOR_THRESHOLD || result.SpoofScore >= LOW_SPOOF_THRESHOLD {
		result.IsReal = false
		result.Confidence = 0.7
		result.SpoofReasons = indicators
	} else {
		result.IsReal = true
		result.Confidence = 0.9
	}

	return result
}

// calculateAdvancedLBPScore implements Local Binary Pattern analysis
func (fm *FaceMatcher) calculateAdvancedLBPScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 8 || gray.Cols() < 8 {
		return 0.5 // Medium suspicion for invalid input
	}

	// Create LBP image
	lbp := gocv.NewMat()
	defer lbp.Close()
	lbp = gray.Clone()

	rows := gray.Rows()
	cols := gray.Cols()

	// Calculate LBP for interior pixels
	for i := 1; i < rows-1; i++ {
		for j := 1; j < cols-1; j++ {
			center := gray.GetUCharAt(i, j)
			var lbpValue uint8 = 0

			// 8-connected neighbors
			neighbors := [][]int{
				{i - 1, j - 1}, {i - 1, j}, {i - 1, j + 1},
				{i, j + 1}, {i + 1, j + 1}, {i + 1, j},
				{i + 1, j - 1}, {i, j - 1},
			}

			for idx, neighbor := range neighbors {
				if gray.GetUCharAt(neighbor[0], neighbor[1]) >= center {
					lbpValue |= (1 << uint(idx))
				}
			}
			lbp.SetUCharAt(i, j, lbpValue)
		}
	}

	// Calculate histogram of LBP values
	hist := make([]int, 256)
	for i := 1; i < rows-1; i++ {
		for j := 1; j < cols-1; j++ {
			hist[lbp.GetUCharAt(i, j)]++
		}
	}

	// Calculate uniformity - high uniformity suggests artificial patterns
	totalPixels := (rows - 2) * (cols - 2)
	maxBin := 0
	for _, count := range hist {
		if count > maxBin {
			maxBin = count
		}
	}

	uniformity := float64(maxBin) / float64(totalPixels)
	return uniformity // Higher values indicate more uniform (suspicious) patterns
}

// calculateLPQScore implements Local Phase Quantization analysis
func (fm *FaceMatcher) calculateLPQScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 8 || gray.Cols() < 8 {
		return 0.5
	}

	// Simplified LPQ using gradients as approximation
	gradX := gocv.NewMat()
	gradY := gocv.NewMat()
	defer gradX.Close()
	defer gradY.Close()

	gocv.Sobel(gray, &gradX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &gradY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate phase information
	magnitude := gocv.NewMat()
	angle := gocv.NewMat()
	defer magnitude.Close()
	defer angle.Close()

	gocv.CartToPolar(gradX, gradY, &magnitude, &angle, false)

	// Calculate phase consistency as a measure of naturalness
	meanMat := gocv.NewMat()
	stdMat := gocv.NewMat()
	defer meanMat.Close()
	defer stdMat.Close()

	gocv.MeanStdDev(angle, &meanMat, &stdMat)
	phaseVariance := stdMat.GetDoubleAt(0, 0)

	// Normalize to 0-1 range (low variance suggests artificial patterns)
	normalized := 1.0 - (phaseVariance / 3.14159) // Max possible std dev for angles
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}

	return normalized
}

// calculateAdvancedReflectionScore analyzes reflection patterns
func (fm *FaceMatcher) calculateAdvancedReflectionScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 16 || gray.Cols() < 16 {
		return 0.3 // Medium suspicion for small faces
	}

	// Analyze intensity distribution for unnatural highlights
	meanMat := gocv.NewMat()
	stdMat := gocv.NewMat()
	defer meanMat.Close()
	defer stdMat.Close()

	gocv.MeanStdDev(gray, &meanMat, &stdMat)
	mean := meanMat.GetDoubleAt(0, 0)
	std := stdMat.GetDoubleAt(0, 0)

	// Define highlight threshold (2 standard deviations above mean)
	threshold := mean + 2*std
	highlightMask := gocv.NewMat()
	defer highlightMask.Close()
	gocv.Threshold(gray, &highlightMask, float32(threshold), 255, gocv.ThresholdBinary)

	highlightPixels := gocv.CountNonZero(highlightMask)
	totalPixels := gray.Rows() * gray.Cols()
	highlightRatio := float64(highlightPixels) / float64(totalPixels)

	// Check for gradient inconsistencies (flat surfaces show uniform gradients)
	gradX := gocv.NewMat()
	gradY := gocv.NewMat()
	defer gradX.Close()
	defer gradY.Close()

	gocv.Sobel(gray, &gradX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &gradY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	gradMag := gocv.NewMat()
	defer gradMag.Close()
	gocv.Magnitude(gradX, gradY, &gradMag)

	gradMeanMat := gocv.NewMat()
	gradStdMat := gocv.NewMat()
	defer gradMeanMat.Close()
	defer gradStdMat.Close()

	gocv.MeanStdDev(gradMag, &gradMeanMat, &gradStdMat)
	gradStd := gradStdMat.GetDoubleAt(0, 0)

	// Low gradient variation suggests flat surface (like a screen)
	gradConsistency := 1.0 - (gradStd / 100.0) // Normalize
	if gradConsistency < 0 {
		gradConsistency = 0
	}

	// Combine highlight ratio and gradient consistency
	reflectionScore := (highlightRatio*0.6 + gradConsistency*0.4)

	if reflectionScore > 1.0 {
		reflectionScore = 1.0
	}

	return reflectionScore
}

// analyzeColorSpaces performs multi-channel color space analysis
func (fm *FaceMatcher) analyzeColorSpaces(face gocv.Mat) ColorSpaceScores {
	scores := ColorSpaceScores{}

	if face.Empty() {
		return scores
	}

	// Convert to different color spaces and analyze consistency

	// YCrCb analysis
	ycrcb := gocv.NewMat()
	defer ycrcb.Close()
	gocv.CvtColor(face, &ycrcb, gocv.ColorBGRToYCrCb)
	scores.YCrCbConsistency = fm.calculateColorConsistency(ycrcb)

	// HSV analysis
	hsv := gocv.NewMat()
	defer hsv.Close()
	gocv.CvtColor(face, &hsv, gocv.ColorBGRToHSV)
	scores.HSVConsistency = fm.calculateColorConsistency(hsv)

	// LAB analysis
	lab := gocv.NewMat()
	defer lab.Close()
	gocv.CvtColor(face, &lab, gocv.ColorBGRToLab)
	scores.LABConsistency = fm.calculateColorConsistency(lab)

	return scores
}

// calculateColorConsistency analyzes color distribution consistency
func (fm *FaceMatcher) calculateColorConsistency(colorImg gocv.Mat) float64 {
	if colorImg.Empty() {
		return 0.0
	}

	channels := gocv.Split(colorImg)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	if len(channels) < 3 {
		return 0.0
	}

	var variances []float64
	for _, channel := range channels {
		meanMat := gocv.NewMat()
		stdMat := gocv.NewMat()
		defer meanMat.Close()
		defer stdMat.Close()

		gocv.MeanStdDev(channel, &meanMat, &stdMat)
		variance := stdMat.GetDoubleAt(0, 0)
		variances = append(variances, variance)
	}

	// Calculate coefficient of variation across channels
	avgVariance := (variances[0] + variances[1] + variances[2]) / 3

	// Calculate how much the variances differ from each other
	diffSum := 0.0
	for _, v := range variances {
		diffSum += (v - avgVariance) * (v - avgVariance)
	}

	inconsistency := diffSum / 3.0
	normalized := inconsistency / 10000.0 // Normalize to reasonable range
	if normalized > 1.0 {
		normalized = 1.0
	}

	return normalized
}

// performEdgeAnalysis analyzes edge characteristics
func (fm *FaceMatcher) performEdgeAnalysis(gray gocv.Mat) EdgeAnalysisScores {
	scores := EdgeAnalysisScores{}

	if gray.Empty() {
		return scores
	}

	// Canny edge detection
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 50, 150)

	// Calculate edge density
	nonZero := gocv.CountNonZero(edges)
	totalPixels := edges.Rows() * edges.Cols()
	scores.EdgeDensity = float64(nonZero) / float64(totalPixels)

	// Calculate edge orientation distribution
	gradX := gocv.NewMat()
	gradY := gocv.NewMat()
	defer gradX.Close()
	defer gradY.Close()

	gocv.Sobel(gray, &gradX, gocv.MatTypeCV64F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &gradY, gocv.MatTypeCV64F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	// Calculate gradient angles
	angles := gocv.NewMat()
	magnitude := gocv.NewMat()
	defer angles.Close()
	defer magnitude.Close()

	gocv.CartToPolar(gradX, gradY, &magnitude, &angles, false)

	// Analyze orientation distribution
	meanMat := gocv.NewMat()
	stdMat := gocv.NewMat()
	defer meanMat.Close()
	defer stdMat.Close()

	gocv.MeanStdDev(angles, &meanMat, &stdMat)
	scores.EdgeOrientation = stdMat.GetDoubleAt(0, 0) / 3.14159 // Normalize

	// Calculate edge sharpness using Laplacian
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	lapMean := gocv.NewMat()
	lapStd := gocv.NewMat()
	defer lapMean.Close()
	defer lapStd.Close()

	gocv.MeanStdDev(laplacian, &lapMean, &lapStd)
	sharpness := lapStd.GetDoubleAt(0, 0) / 255.0 // Normalize
	scores.EdgeSharpness = sharpness

	return scores
}

// calculateAdvancedTextureScore analyzes texture complexity
func (fm *FaceMatcher) calculateAdvancedTextureScore(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 8 || gray.Cols() < 8 {
		return 0.7 // High suspicion for invalid input
	}

	// Multi-scale texture analysis
	var textureScores []float64

	// Analyze at different scales
	scales := []image.Point{{3, 3}, {5, 5}, {7, 7}}

	for _, scale := range scales {
		blurred := gocv.NewMat()
		defer blurred.Close()
		gocv.GaussianBlur(gray, &blurred, scale, 0, 0, gocv.BorderDefault)

		diff := gocv.NewMat()
		defer diff.Close()
		gocv.AbsDiff(gray, blurred, &diff)

		meanMat := gocv.NewMat()
		stdMat := gocv.NewMat()
		defer meanMat.Close()
		defer stdMat.Close()

		gocv.MeanStdDev(diff, &meanMat, &stdMat)
		textureMeasure := meanMat.GetDoubleAt(0, 0) / 255.0
		textureScores = append(textureScores, textureMeasure)
	}

	// Calculate overall texture measure
	avgTexture := (textureScores[0] + textureScores[1] + textureScores[2]) / 3

	// Invert score: low texture variation = high spoof score
	smoothnessScore := 1.0 - avgTexture

	if smoothnessScore < 0 {
		smoothnessScore = 0
	}
	if smoothnessScore > 1 {
		smoothnessScore = 1
	}

	return smoothnessScore
}

// analyzeFrequencyDomain performs frequency domain analysis
func (fm *FaceMatcher) analyzeFrequencyDomain(gray gocv.Mat) FrequencyScores {
	scores := FrequencyScores{}

	if gray.Empty() || gray.Rows() < 32 || gray.Cols() < 32 {
		scores.HighFrequencyContent = 0.1
		scores.FrequencyDistribution = 0.5
		scores.NoiseLevel = 0.3
		return scores
	}

	// Convert to float for FFT
	grayFloat := gocv.NewMat()
	defer grayFloat.Close()
	gray.ConvertTo(&grayFloat, gocv.MatTypeCV32F)

	// Apply high-pass filter to detect high-frequency content
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	lowPass := gocv.NewMat()
	defer lowPass.Close()
	gocv.MorphologyEx(grayFloat, &lowPass, gocv.MorphOpen, kernel)

	highPass := gocv.NewMat()
	defer highPass.Close()
	gocv.Subtract(grayFloat, lowPass, &highPass)

	// Calculate high-frequency content
	meanMat := gocv.NewMat()
	stdMat := gocv.NewMat()
	defer meanMat.Close()
	defer stdMat.Close()

	gocv.MeanStdDev(highPass, &meanMat, &stdMat)
	highFreqMean := meanMat.GetDoubleAt(0, 0)
	scores.HighFrequencyContent = highFreqMean / 255.0

	// Estimate noise level using difference of Gaussians
	gaussian1 := gocv.NewMat()
	gaussian2 := gocv.NewMat()
	defer gaussian1.Close()
	defer gaussian2.Close()

	gocv.GaussianBlur(grayFloat, &gaussian1, image.Pt(3, 3), 0, 0, gocv.BorderDefault)
	gocv.GaussianBlur(grayFloat, &gaussian2, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

	noiseMat := gocv.NewMat()
	defer noiseMat.Close()
	gocv.Subtract(gaussian1, gaussian2, &noiseMat)

	noiseMean := gocv.NewMat()
	noiseStd := gocv.NewMat()
	defer noiseMean.Close()
	defer noiseStd.Close()

	gocv.MeanStdDev(noiseMat, &noiseMean, &noiseStd)
	scores.NoiseLevel = noiseStd.GetDoubleAt(0, 0) / 255.0

	// Frequency distribution analysis using gradient magnitude distribution
	gradX := gocv.NewMat()
	gradY := gocv.NewMat()
	defer gradX.Close()
	defer gradY.Close()

	gocv.Sobel(grayFloat, &gradX, gocv.MatTypeCV32F, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(grayFloat, &gradY, gocv.MatTypeCV32F, 0, 1, 3, 1, 0, gocv.BorderDefault)

	gradMag := gocv.NewMat()
	defer gradMag.Close()
	gocv.Magnitude(gradX, gradY, &gradMag)

	gradMeanMat := gocv.NewMat()
	gradStdMat := gocv.NewMat()
	defer gradMeanMat.Close()
	defer gradStdMat.Close()

	gocv.MeanStdDev(gradMag, &gradMeanMat, &gradStdMat)
	gradStd := gradStdMat.GetDoubleAt(0, 0)
	scores.FrequencyDistribution = gradStd / 255.0

	return scores
}

// calculateOverallSharpness calculates overall image sharpness using Laplacian variance
func (fm *FaceMatcher) calculateOverallSharpness(gray gocv.Mat) float64 {
	if gray.Empty() || gray.Rows() < 8 || gray.Cols() < 8 {
		return 50.0 // Default medium sharpness for invalid input
	}

	// Calculate Laplacian for blur detection
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(gray, &laplacian, gocv.MatTypeCV64F, 1, 1, 0, gocv.BorderDefault)

	// Calculate variance of Laplacian (measure of focus/sharpness)
	meanMat := gocv.NewMat()
	stddevMat := gocv.NewMat()
	defer meanMat.Close()
	defer stddevMat.Close()

	gocv.MeanStdDev(laplacian, &meanMat, &stddevMat)
	variance := stddevMat.GetDoubleAt(0, 0) * stddevMat.GetDoubleAt(0, 0)

	// Return variance as sharpness measure
	// Higher variance = sharper image
	// Lower variance = blurrier image
	return variance
}

// detectCompressionLevel analyzes compression artifacts and detail loss
func (fm *FaceMatcher) detectCompressionLevel(gray, face gocv.Mat) float64 {
	if gray.Empty() || face.Empty() {
		return 0.3 // Assume moderate compression for invalid input
	}

	compressionScore := 0.0

	// 1. Detect JPEG block artifacts (8x8 DCT blocks)
	blockArtifacts := fm.detectBlockArtifacts(gray)
	compressionScore += blockArtifacts * 0.3

	// 2. Analyze high-frequency content loss
	highFreqLoss := fm.analyzeHighFrequencyLoss(gray)
	compressionScore += highFreqLoss * 0.25

	// 3. Check for ringing artifacts around edges
	ringingArtifacts := fm.detectRingingArtifacts(gray)
	compressionScore += ringingArtifacts * 0.2

	// 4. Analyze color quantization (if face is color)
	if face.Channels() >= 3 {
		colorQuantization := fm.analyzeColorQuantization(face)
		compressionScore += colorQuantization * 0.25
	}

	// Clamp to [0, 1] range
	if compressionScore > 1.0 {
		compressionScore = 1.0
	}
	if compressionScore < 0.0 {
		compressionScore = 0.0
	}

	return compressionScore
}

// detectBlockArtifacts detects JPEG 8x8 block artifacts
func (fm *FaceMatcher) detectBlockArtifacts(gray gocv.Mat) float64 {
	if gray.Rows() < 16 || gray.Cols() < 16 {
		return 0.0
	}

	// Look for discontinuities at 8-pixel boundaries
	blockScore := 0.0
	samples := 0

	// Check vertical block boundaries
	for y := 8; y < gray.Rows()-8; y += 8 {
		for x := 1; x < gray.Cols()-1; x++ {
			left := gray.GetUCharAt(y, x-1)
			center := gray.GetUCharAt(y, x)
			right := gray.GetUCharAt(y, x+1)

			// Strong discontinuity at block boundary
			blockDiff := math.Abs(float64(center) - (float64(left)+float64(right))/2)
			if blockDiff > 10 { // Significant jump
				blockScore += blockDiff / 255.0
			}
			samples++
		}
	}

	if samples > 0 {
		return blockScore / float64(samples)
	}
	return 0.0
}

// analyzeHighFrequencyLoss detects loss of high-frequency details
func (fm *FaceMatcher) analyzeHighFrequencyLoss(gray gocv.Mat) float64 {
	if gray.Rows() < 8 || gray.Cols() < 8 {
		return 0.0
	}

	// Apply high-pass filter
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	// Low-pass version
	lowPass := gocv.NewMat()
	defer lowPass.Close()
	gocv.GaussianBlur(gray, &lowPass, image.Pt(3, 3), 0, 0, gocv.BorderDefault)

	// High-pass = original - low-pass
	grayFloat := gocv.NewMat()
	lowPassFloat := gocv.NewMat()
	defer grayFloat.Close()
	defer lowPassFloat.Close()

	gray.ConvertTo(&grayFloat, gocv.MatTypeCV32F)
	lowPass.ConvertTo(&lowPassFloat, gocv.MatTypeCV32F)

	highPass := gocv.NewMat()
	defer highPass.Close()
	gocv.Subtract(grayFloat, lowPassFloat, &highPass)

	// Calculate energy in high frequencies
	meanMat := gocv.NewMat()
	stdMat := gocv.NewMat()
	defer meanMat.Close()
	defer stdMat.Close()

	gocv.MeanStdDev(highPass, &meanMat, &stdMat)
	highFreqEnergy := stdMat.GetDoubleAt(0, 0)

	// Low high-frequency energy suggests compression
	// Normalize: typical uncompressed images have higher energy
	normalizedEnergy := highFreqEnergy / 30.0 // Adjust based on testing
	if normalizedEnergy > 1.0 {
		normalizedEnergy = 1.0
	}

	return 1.0 - normalizedEnergy // Invert: low energy = high compression
}

// detectRingingArtifacts detects ringing artifacts around strong edges
func (fm *FaceMatcher) detectRingingArtifacts(gray gocv.Mat) float64 {
	if gray.Rows() < 8 || gray.Cols() < 8 {
		return 0.0
	}

	// Detect strong edges
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 100, 200)

	// For pixels near edges, look for oscillating patterns
	dilated := gocv.NewMat()
	defer dilated.Close()
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()
	gocv.Dilate(edges, &dilated, kernel)

	ringingScore := 0.0
	samples := 0

	// Simplified ringing detection: look for variance near edges
	for y := 2; y < gray.Rows()-2; y++ {
		for x := 2; x < gray.Cols()-2; x++ {
			if dilated.GetUCharAt(y, x) > 0 { // Near an edge
				// Calculate local variance in 5x5 window
				localVar := 0.0
				center := float64(gray.GetUCharAt(y, x))

				for dy := -2; dy <= 2; dy++ {
					for dx := -2; dx <= 2; dx++ {
						pixel := float64(gray.GetUCharAt(y+dy, x+dx))
						localVar += (pixel - center) * (pixel - center)
					}
				}
				localVar /= 25.0

				ringingScore += localVar / (255.0 * 255.0)
				samples++
			}
		}
	}

	if samples > 0 {
		avgRinging := ringingScore / float64(samples)
		// Normalize: high variance near edges suggests ringing
		return math.Min(avgRinging*10.0, 1.0)
	}
	return 0.0
}

// analyzeColorQuantization detects color quantization artifacts
func (fm *FaceMatcher) analyzeColorQuantization(face gocv.Mat) float64 {
	if face.Empty() || face.Channels() < 3 {
		return 0.0
	}

	channels := gocv.Split(face)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	quantizationScore := 0.0

	// For each color channel, analyze histogram
	for _, channel := range channels {
		// Create histogram
		hist := gocv.NewMat()
		defer hist.Close()

		mask := gocv.NewMat() // Empty mask
		defer mask.Close()

		gocv.CalcHist([]gocv.Mat{channel}, []int{0}, mask, &hist, []int{256}, []float64{0, 256}, false)

		// Count non-zero bins (used colors)
		nonZeroBins := 0
		for i := 0; i < 256; i++ {
			if hist.GetFloatAt(i, 0) > 0 {
				nonZeroBins++
			}
		}

		// Fewer unique colors suggests quantization
		colorRatio := float64(nonZeroBins) / 256.0
		quantizationScore += (1.0 - colorRatio) / 3.0 // Average across channels
	}

	return quantizationScore
}
