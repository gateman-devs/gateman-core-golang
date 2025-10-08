package biometric

import (
	"fmt"
	"math"
	"time"

	"gateman.io/infrastructure/logger"
)

// TestEnhancedFaceComparison tests the enhanced face comparison with detailed logging
func TestEnhancedFaceComparison(image1URL, image2URL string) error {
	fmt.Println("\n========================================")
	fmt.Println("ENHANCED FACE COMPARISON TEST")
	fmt.Println("========================================\n")

	startTime := time.Now()

	// Create local face service
	lfs := NewLocalFaceService()
	if !lfs.modelsLoaded {
		return fmt.Errorf("models not loaded")
	}

	// Process first image with enhanced detection
	fmt.Println("📥 Processing Image 1...")
	img1, faces1, quality1, err := lfs.ProcessImageWithEnhancedDetection(image1URL)
	if err != nil {
		return fmt.Errorf("failed to process image 1: %v", err)
	}
	defer img1.Close()

	fmt.Printf("✅ Image 1 processed: %d faces detected, quality: %.4f\n", len(faces1), quality1)

	// Process second image with enhanced detection
	fmt.Println("\n📥 Processing Image 2...")
	img2, faces2, quality2, err := lfs.ProcessImageWithEnhancedDetection(image2URL)
	if err != nil {
		return fmt.Errorf("failed to process image 2: %v", err)
	}
	defer img2.Close()

	fmt.Printf("✅ Image 2 processed: %d faces detected, quality: %.4f\n", len(faces2), quality2)

	// Check if faces are detected
	if len(faces1) == 0 {
		return fmt.Errorf("no face detected in image 1")
	}
	if len(faces2) == 0 {
		return fmt.Errorf("no face detected in image 2")
	}

	// Use the largest face from each image
	face1 := lfs.getLargestFace(faces1)
	face2 := lfs.getLargestFace(faces2)

	fmt.Printf("\n📐 Face 1 dimensions: %dx%d pixels\n", face1.Dx(), face1.Dy())
	fmt.Printf("📐 Face 2 dimensions: %dx%d pixels\n", face2.Dx(), face2.Dy())

	// Extract face regions with padding
	fmt.Println("\n🔍 Extracting face regions with padding...")
	faceRegion1 := lfs.extractFaceRegionWithPadding(img1, face1)
	faceRegion2 := lfs.extractFaceRegionWithPadding(img2, face2)
	defer faceRegion1.Close()
	defer faceRegion2.Close()

	// Enhanced preprocessing for better comparison
	fmt.Println("⚙️  Applying enhanced preprocessing...")
	processed1 := lfs.enhancedFacePreprocessing(faceRegion1)
	processed2 := lfs.enhancedFacePreprocessing(faceRegion2)
	defer processed1.Close()
	defer processed2.Close()

	fmt.Println("\n========================================")
	fmt.Println("DETAILED SIMILARITY ANALYSIS")
	fmt.Println("========================================\n")

	// Calculate individual similarity components
	fmt.Println("🔬 Calculating base similarity (template + SSIM + histogram + edge)...")
	baseSimilarity := float64(lfs.calculateSimilarity(processed1, processed2))
	fmt.Printf("   Base Similarity: %.4f\n", baseSimilarity)

	fmt.Println("\n🔬 Calculating feature-based similarity (ORB features)...")
	featureScore := lfs.calculateFeatureSimilarity(processed1, processed2)
	fmt.Printf("   Feature Score: %.4f\n", featureScore)

	fmt.Println("\n🔬 Calculating edge similarity...")
	edgeScore := lfs.calculateSimpleEdgeSimilarity(processed1, processed2)
	fmt.Printf("   Edge Score: %.4f\n", edgeScore)

	// Calculate enhanced similarity using multiple methods
	fmt.Println("\n🎯 Calculating ENHANCED SIMILARITY...")
	weights := []float64{0.6, 0.25, 0.15}
	scores := []float64{baseSimilarity, featureScore, edgeScore}
	
	enhancedSimilarity := 0.0
	for i, score := range scores {
		enhancedSimilarity += score * weights[i]
	}
	
	fmt.Printf("   Enhanced Similarity: %.4f\n", enhancedSimilarity)
	fmt.Printf("   (Weighted: base=%.1f%%, feature=%.1f%%, edge=%.1f%%)\n", 
		weights[0]*100, weights[1]*100, weights[2]*100)

	// Calculate enhanced confidence
	fmt.Println("\n🎯 Calculating ENHANCED CONFIDENCE...")
	confidence := lfs.calculateEnhancedConfidence(enhancedSimilarity, quality1, quality2, faces1, faces2)
	fmt.Printf("   Enhanced Confidence: %.4f\n", confidence)

	// Calculate adaptive thresholds
	fmt.Println("\n========================================")
	fmt.Println("THRESHOLD ANALYSIS")
	fmt.Println("========================================\n")

	baseThreshold := 0.62
	minConfidence := 0.52

	fmt.Printf("📊 Base Threshold: %.4f\n", baseThreshold)
	fmt.Printf("📊 Min Confidence: %.4f\n", minConfidence)

	// Quality-based threshold adjustment
	avgQuality := (quality1 + quality2) / 2.0
	fmt.Printf("\n📊 Average Quality: %.4f\n", avgQuality)
	
	if avgQuality < 0.3 {
		baseThreshold += 0.1
		fmt.Printf("   ⚠️  Low quality detected - threshold increased by 0.1 to %.4f\n", baseThreshold)
	} else if avgQuality > 0.7 {
		baseThreshold -= 0.05
		fmt.Printf("   ✅ High quality detected - threshold decreased by 0.05 to %.4f\n", baseThreshold)
	} else {
		fmt.Printf("   ℹ️  Medium quality - threshold unchanged\n")
	}

	// Multiple faces penalty
	if len(faces1) != 1 || len(faces2) != 1 {
		baseThreshold += 0.12
		fmt.Printf("   ⚠️  Multiple faces detected - threshold increased by 0.12 to %.4f\n", baseThreshold)
	} else {
		fmt.Printf("   ✅ Single face in each image - no penalty\n")
	}

	// Quality difference penalty
	qualityDiff := math.Abs(quality1 - quality2)
	fmt.Printf("\n📊 Quality Difference: %.4f\n", qualityDiff)
	
	if qualityDiff > 0.4 {
		baseThreshold += 0.08
		fmt.Printf("   ⚠️  Large quality difference - threshold increased by 0.08 to %.4f\n", baseThreshold)
	} else {
		fmt.Printf("   ✅ Quality difference acceptable - no penalty\n")
	}

	// Face size consistency check
	sizeRatio := float64(face1.Dx()*face1.Dy()) / float64(face2.Dx()*face2.Dy())
	fmt.Printf("\n📊 Face Size Ratio: %.4f\n", sizeRatio)
	
	if sizeRatio < 0.4 || sizeRatio > 2.5 {
		baseThreshold += 0.08
		fmt.Printf("   ⚠️  Large size difference - threshold increased by 0.08 to %.4f\n", baseThreshold)
	} else {
		fmt.Printf("   ✅ Face sizes consistent - no penalty\n")
	}

	// Final decision
	fmt.Println("\n========================================")
	fmt.Println("FINAL DECISION")
	fmt.Println("========================================\n")

	fmt.Printf("📊 Final Threshold: %.4f\n", baseThreshold)
	fmt.Printf("📊 Enhanced Similarity: %.4f\n", enhancedSimilarity)
	fmt.Printf("📊 Enhanced Confidence: %.4f\n", confidence)
	fmt.Printf("📊 Min Confidence Required: %.4f\n", minConfidence)

	isMatch := enhancedSimilarity > baseThreshold && confidence > minConfidence

	fmt.Println("\n🎯 MATCH DECISION:")
	if isMatch {
		fmt.Println("   ✅ MATCH - The faces are from the SAME person")
		fmt.Printf("   Similarity (%.4f) > Threshold (%.4f) ✓\n", enhancedSimilarity, baseThreshold)
		fmt.Printf("   Confidence (%.4f) > Min Required (%.4f) ✓\n", confidence, minConfidence)
	} else {
		fmt.Println("   ❌ NO MATCH - The faces are from DIFFERENT people")
		if enhancedSimilarity <= baseThreshold {
			fmt.Printf("   Similarity (%.4f) ≤ Threshold (%.4f) ✗\n", enhancedSimilarity, baseThreshold)
		}
		if confidence <= minConfidence {
			fmt.Printf("   Confidence (%.4f) ≤ Min Required (%.4f) ✗\n", confidence, minConfidence)
		}
	}

	processingTime := time.Since(startTime)
	fmt.Printf("\n⏱️  Total Processing Time: %dms\n", processingTime.Milliseconds())

	fmt.Println("\n========================================")
	fmt.Println("TEST COMPLETE")
	fmt.Println("========================================\n")

	// Log to system logger as well
	logger.Info("Enhanced face comparison test completed", logger.LoggerOptions{
		Key: "test_result",
		Data: map[string]interface{}{
			"is_match":             isMatch,
			"enhanced_similarity":  enhancedSimilarity,
			"base_similarity":      baseSimilarity,
			"feature_score":        featureScore,
			"edge_score":           edgeScore,
			"confidence":           confidence,
			"threshold":            baseThreshold,
			"quality1":             quality1,
			"quality2":             quality2,
			"faces_detected_img1":  len(faces1),
			"faces_detected_img2":  len(faces2),
			"processing_time_ms":   processingTime.Milliseconds(),
		},
	})

	return nil
}

// RunEnhancedComparisonTest is a convenience function to run the test
func RunEnhancedComparisonTest() {
	// Test with the provided images (2 different people)
	image1 := "https://res.cloudinary.com/themizehq/image/upload/v1703815583/ba30e7e5-5518-4818-91f6-a1e3f8941932.jpg"
	image2 := "https://res.cloudinary.com/themizehq/image/upload/v1680938819/core/profile-images/64260ff04223586e8fa41413/file.png"

	fmt.Println("Testing with 2 DIFFERENT people:")
	fmt.Printf("Image 1: %s\n", image1)
	fmt.Printf("Image 2: %s\n", image2)

	err := TestEnhancedFaceComparison(image1, image2)
	if err != nil {
		fmt.Printf("\n❌ TEST FAILED: %v\n", err)
		logger.Error("Enhanced face comparison test failed", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
	}
}
