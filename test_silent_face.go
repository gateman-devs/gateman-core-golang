//go:build testsilentface
// +build testsilentface

package main

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gateman.io/infrastructure/biometric"
	"gocv.io/x/gocv"
)

func downloadImage(url, filename string) error {
	fmt.Printf("Downloading %s...\n", filename)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	fmt.Printf("✅ Downloaded %s\n", filename)
	return nil
}

func detectFaceRegion(img gocv.Mat) (gocv.Mat, error) {
	// Load YuNet face detector
	yunetPath := "./models/yunet/face_detection_yunet_2023mar.onnx"
	detector := gocv.NewFaceDetectorYN(yunetPath, "", image.Pt(img.Cols(), img.Rows()))
	defer detector.Close()

	// Detect faces
	faces := gocv.NewMat()
	defer faces.Close()

	detector.Detect(img, &faces)

	if faces.Rows() == 0 {
		return gocv.NewMat(), fmt.Errorf("no face detected")
	}

	// Get first face
	x := int(faces.GetFloatAt(0, 0))
	y := int(faces.GetFloatAt(0, 1))
	w := int(faces.GetFloatAt(0, 2))
	h := int(faces.GetFloatAt(0, 3))

	// Ensure coordinates are within bounds
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+w > img.Cols() {
		w = img.Cols() - x
	}
	if y+h > img.Rows() {
		h = img.Rows() - y
	}

	// Extract face region
	rect := image.Rect(x, y, x+w, y+h)
	faceRegion := img.Region(rect)

	fmt.Printf("✅ Face detected at (%d, %d) with size %dx%d\n", x, y, w, h)
	return faceRegion, nil
}

func testImage(sfas *biometric.SilentFaceAntiSpoof, imagePath, imageType string) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("Testing %s Image: %s\n", imageType, imagePath)
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	// Load image
	img := gocv.IMRead(imagePath, gocv.IMReadColor)
	if img.Empty() {
		fmt.Printf("❌ Failed to load image: %s\n", imagePath)
		return
	}
	defer img.Close()

	fmt.Printf("Image size: %dx%d\n", img.Cols(), img.Rows())

	// Detect face region
	faceRegion, err := detectFaceRegion(img)
	if err != nil {
		fmt.Printf("❌ Face detection failed: %v\n", err)
		return
	}
	defer faceRegion.Close()

	// Run anti-spoofing prediction
	startTime := time.Now()
	isReal, confidence, err := sfas.Predict(faceRegion)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Prediction failed: %v\n", err)
		return
	}

	// Display results
	fmt.Printf("\n📊 RESULTS:\n")
	fmt.Printf("  Is Real: %v\n", isReal)
	fmt.Printf("  Confidence: %.4f\n", confidence)
	fmt.Printf("  Processing Time: %v\n", elapsed)

	if isReal {
		fmt.Printf("  ✅ VERDICT: REAL FACE (Live person detected)\n")
	} else {
		fmt.Printf("  ❌ VERDICT: FAKE/SPOOF (Attack detected)\n")
	}

	// Test with different thresholds
	fmt.Printf("\n🎯 Threshold Analysis:\n")
	thresholds := []float64{0.3, 0.4, 0.5, 0.6, 0.7}
	for _, threshold := range thresholds {
		result := confidence >= threshold
		status := "FAIL"
		if result {
			status = "PASS"
		}
		fmt.Printf("  Threshold %.1f: %s (confidence: %.4f)\n", threshold, status, confidence)
	}
}

func main() {
	fmt.Println("🚀 Silent-Face-Anti-Spoofing Test Suite")
	fmt.Println(strings.Repeat("=", 60))

	// Create test directory
	os.MkdirAll("./test_images", 0755)

	// Download test images
	paintingURL := "https://64.media.tumblr.com/0f1d9be0930e0fd6e1421e0af63b4baa/4b38c49aa49bf456-a5/s1280x1920/54be3df4f578d67626ed9b3849f53d129667b940.jpg"
	legitURL := "https://lookaside.fbsbx.com/elementpath/media/?media_id=323764547398564&version=1759785626"

	paintingPath := "./test_images/painting.jpg"
	legitPath := "./test_images/legit.jpg"

	// Download if not exists
	if _, err := os.Stat(paintingPath); os.IsNotExist(err) {
		if err := downloadImage(paintingURL, paintingPath); err != nil {
			fmt.Printf("❌ Failed to download painting image: %v\n", err)
			return
		}
	}

	if _, err := os.Stat(legitPath); os.IsNotExist(err) {
		if err := downloadImage(legitURL, legitPath); err != nil {
			fmt.Printf("❌ Failed to download legit image: %v\n", err)
			return
		}
	}

	// Initialize Silent-Face-Anti-Spoofing
	fmt.Println("\n🔧 Initializing Silent-Face-Anti-Spoofing models...")
	sfas, err := biometric.NewSilentFaceAntiSpoof()
	if err != nil {
		fmt.Printf("❌ Failed to initialize anti-spoofing: %v\n", err)
		return
	}
	defer sfas.Close()

	// Print model info
	modelInfo := sfas.GetModelInfo()
	fmt.Printf("\n📦 Model Information:\n")
	fmt.Printf("  Loaded: %v\n", modelInfo["loaded"])
	fmt.Printf("  Models: %v\n", modelInfo["models"])
	fmt.Printf("  Input Size: %v\n", modelInfo["input_size"])
	fmt.Printf("  Reference: %v\n", modelInfo["reference"])

	// Test painting image (should be detected as fake)
	testImage(sfas, paintingPath, "PAINTING/SPOOF")

	// Test legitimate image (should be detected as real)
	testImage(sfas, legitPath, "LEGITIMATE")

	// Summary
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Println("✅ Test suite completed!")
	fmt.Printf(strings.Repeat("=", 60) + "\n")
}
