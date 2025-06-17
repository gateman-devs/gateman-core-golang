package facematch

import (
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"gocv.io/x/gocv"
)

// isBase64Image checks if a string is a base64 encoded image
func isBase64Image(input string) bool {
	// Check if it starts with a data URL prefix
	if strings.HasPrefix(input, "data:image/") {
		return true
	}

	// Check if it looks like base64 (alphanumeric + / + = padding)
	// This is a simple heuristic - base64 strings are typically longer and contain specific characters
	if len(input) > 100 && !strings.Contains(input, "http") && !strings.Contains(input, "://") {
		// Try to decode a small portion to see if it's valid base64
		if len(input) > 50 {
			testStr := input[:50]
			_, err := base64.StdEncoding.DecodeString(testStr)
			return err == nil
		}
	}

	return false
}

// decodeBase64Image decodes a base64 encoded image string
func decodeBase64Image(input string) ([]byte, error) {
	// Handle data URLs (e.g., "data:image/jpeg;base64,/9j/4AAQ...")
	if strings.HasPrefix(input, "data:image/") {
		parts := strings.Split(input, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid data URL format")
		}
		return base64.StdEncoding.DecodeString(parts[1])
	}

	// Handle raw base64 strings
	return base64.StdEncoding.DecodeString(input)
}

// loadImage loads an image from either a URL or base64 string
func loadImage(input string) ([]byte, error) {
	if isBase64Image(input) {
		log.Printf("Loading image from base64 data")
		return decodeBase64Image(input)
	}

	log.Printf("Loading image from URL: %s", input)
	return downloadImage(input)
}

func downloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

type FaceDetector struct {
	net gocv.Net
}

func NewFaceDetector() (*FaceDetector, error) {
	modelPath := "infrastructure/facematch/models/yunet.onnx"

	log.Printf("Loading YuNet model from: %s", modelPath)

	net := gocv.ReadNet(modelPath, "")
	if net.Empty() {
		return nil, fmt.Errorf("failed to load YuNet model: %s", modelPath)
	}

	log.Printf("YuNet model loaded successfully")

	return &FaceDetector{net: net}, nil
}

func (fd *FaceDetector) Close() {
	fd.net.Close()
}

// Thread-safe detector pool
type DetectorPool struct {
	detectors chan *FaceDetector
	mu        sync.Mutex
}

var globalPool *DetectorPool

func init() {
	// Initialize the global detector pool
	poolSize := 10 // Number of concurrent detectors
	globalPool = &DetectorPool{
		detectors: make(chan *FaceDetector, poolSize),
	}

	// Pre-load detectors
	for i := 0; i < poolSize; i++ {
		detector, err := NewFaceDetector()
		if err != nil {
			log.Fatalf("Failed to initialize detector %d: %v", i, err)
		}
		globalPool.detectors <- detector
	}

	log.Printf("Initialized detector pool with %d detectors", poolSize)
}

func (dp *DetectorPool) Get() *FaceDetector {
	return <-dp.detectors
}

func (dp *DetectorPool) Put(detector *FaceDetector) {
	dp.detectors <- detector
}

func detectSingleFace(imgBytes []byte, detector *FaceDetector) ([]image.Rectangle, []float32, gocv.Mat, error) {
	imgMat, err := gocv.IMDecode(imgBytes, gocv.IMReadColor)
	if err != nil {
		return nil, nil, imgMat, err
	}
	if imgMat.Empty() {
		return nil, nil, imgMat, err
	}

	// Get original image dimensions
	imgHeight := imgMat.Rows()
	imgWidth := imgMat.Cols()

	log.Printf("Processing image: %dx%d", imgWidth, imgHeight)

	// YuNet expects input size of 320x320 (not 160x120)
	inputSize := image.Pt(320, 320)

	// Create blob with proper preprocessing
	// YuNet expects: scale=1.0, size=(320,320), mean=(0,0,0), swapRB=true, crop=false
	blob := gocv.BlobFromImage(imgMat, 1.0, inputSize, gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	log.Printf("Created blob with shape: %v", blob.Size())

	detector.net.SetInput(blob, "")
	log.Printf("Set input to model")

	detections := detector.net.Forward("")
	defer detections.Close()

	log.Printf("Got model output with shape: %v", detections.Size())

	var faces []image.Rectangle
	var scores []float32

	// Check if detections output is valid
	if detections.Empty() {
		log.Printf("Model produced empty output")
		return faces, scores, imgMat, nil
	}

	detectionSize := detections.Size()

	// Validate detection output dimensions
	if len(detectionSize) < 2 {
		log.Printf("Invalid detection output dimensions: %v", detectionSize)
		return faces, scores, imgMat, nil
	}

	// YuNet output format: [1, num_detections, 15] where 15 = [x, y, w, h, confidence, landmarks...]
	rows := detectionSize[1]
	if rows <= 0 {
		log.Printf("No detection rows found")
		return faces, scores, imgMat, nil
	}

	// Check if we have the expected 15 values per detection
	if len(detectionSize) == 3 && detectionSize[2] >= 5 {
		for i := 0; i < rows; i++ {
			score := detections.GetFloatAt(0, i*detectionSize[2]+4)
			log.Printf("Face %d confidence: %.3f", i, score)

			if score > 0.5 { // Higher threshold to reduce false positives
				// Get normalized coordinates (0-1)
				x_norm := detections.GetFloatAt(0, i*detectionSize[2]+0)
				y_norm := detections.GetFloatAt(0, i*detectionSize[2]+1)
				w_norm := detections.GetFloatAt(0, i*detectionSize[2]+2)
				h_norm := detections.GetFloatAt(0, i*detectionSize[2]+3)

				log.Printf("Normalized coords: x=%.3f, y=%.3f, w=%.3f, h=%.3f", x_norm, y_norm, w_norm, h_norm)

				// Convert to absolute pixel coordinates
				x := int(x_norm * float32(imgWidth))
				y := int(y_norm * float32(imgHeight))
				w := int(w_norm * float32(imgWidth))
				h := int(h_norm * float32(imgHeight))

				log.Printf("Pixel coords: x=%d, y=%d, w=%d, h=%d", x, y, w, h)

				// Ensure coordinates are within image bounds
				if x < 0 {
					x = 0
				}
				if y < 0 {
					y = 0
				}
				if x+w > imgWidth {
					w = imgWidth - x
				}
				if y+h > imgHeight {
					h = imgHeight - y
				}

				// Only add face if it has reasonable dimensions
				if w > 20 && h > 20 { // Increased minimum size
					rect := image.Rect(x, y, x+w, y+h)
					faces = append(faces, rect)
					scores = append(scores, score)
					log.Printf("Added face rectangle: %v (score: %.3f)", rect, score)
				} else {
					log.Printf("Face too small, skipping: w=%d, h=%d", w, h)
				}
			}
		}
	} else {
		log.Printf("Unexpected detection output format: %v", detectionSize)
	}

	log.Printf("Final result: %d faces in image (%dx%d)", len(faces), imgWidth, imgHeight)

	return faces, scores, imgMat, nil
}

func extractFaceFeature(imgMat gocv.Mat, faceRect image.Rectangle) gocv.Mat {
	faceMat := imgMat.Region(faceRect)
	defer faceMat.Close()
	resized := gocv.NewMat()
	gocv.Resize(faceMat, &resized, image.Pt(128, 128), 0, 0, gocv.InterpolationLinear)
	return resized
}

func compareHist(mat1, mat2 gocv.Mat) float32 {
	histSize := []int{256}
	ranges := []float64{0, 256}
	channels := []int{0}
	hist1 := gocv.NewMat()
	hist2 := gocv.NewMat()
	defer hist1.Close()
	defer hist2.Close()
	gocv.CalcHist([]gocv.Mat{mat1}, channels, gocv.NewMat(), &hist1, histSize, ranges, false)
	gocv.CalcHist([]gocv.Mat{mat2}, channels, gocv.NewMat(), &hist2, histSize, ranges, false)
	return gocv.CompareHist(hist1, hist2, gocv.HistCmpCorrel)
}

// Compare compares two images (URLs or base64 strings) and returns true if they contain the same face
func Compare(img1 string, img2 string) bool {
	fmt.Println("compare running")
	var wg sync.WaitGroup
	type faceResult struct {
		faces []image.Rectangle
		mat   gocv.Mat
		err   error
	}
	results := make([]faceResult, 2)

	urls := []string{img1, img2}
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			imgBytes, err := loadImage(urls[idx])
			if err != nil {
				results[idx] = faceResult{err: err}
				return
			}

			// Get a detector from the pool
			detector := globalPool.Get()
			defer globalPool.Put(detector) // Return detector to pool when done

			faces, _, mat, err := detectSingleFace(imgBytes, detector)
			results[idx] = faceResult{faces: faces, mat: mat, err: err}
		}(i)
	}
	wg.Wait()

	for i, res := range results {
		if res.err != nil {
			log.Printf("Error processing image %d: %v\n", i+1, res.err)
			return false
		}
		defer res.mat.Close()
		if len(res.faces) != 1 {
			log.Printf("Image %d: expected 1 face, found %d\n", i+1, len(res.faces))
			return false
		}
	}

	var faceMats [2]gocv.Mat
	var faceWg sync.WaitGroup
	faceWg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer faceWg.Done()
			faceMats[idx] = extractFaceFeature(results[idx].mat, results[idx].faces[0])
		}(i)
	}
	faceWg.Wait()
	defer faceMats[0].Close()
	defer faceMats[1].Close()

	similarity := compareHist(faceMats[0], faceMats[1])
	return similarity > 0.7
}

// TestFaceDetection is a helper function to test face detection on a single image (URL or base64)
func TestFaceDetection(imgURL string) {
	// Get a detector from the pool
	detector := globalPool.Get()
	defer globalPool.Put(detector) // Return detector to pool when done

	imgBytes, err := loadImage(imgURL)
	if err != nil {
		log.Printf("Error loading image: %v\n", err)
		return
	}

	faces, scores, mat, err := detectSingleFace(imgBytes, detector)
	if err != nil {
		log.Printf("Error detecting faces: %v\n", err)
		return
	}
	defer mat.Close()

	log.Printf("Test result: Found %d faces in image", len(faces))
	for i, face := range faces {
		confidence := float32(0.0)
		if i < len(scores) {
			confidence = scores[i]
		}
		log.Printf("Face %d: %v (confidence: %.3f)", i+1, face, confidence)
	}
}

// ImageQualityResult represents the result of image quality verification
type ImageQualityResult struct {
	IsValid bool
	Reason  string // Possible values:
	// - "valid": Image passed all quality checks
	// - "download_failed": Could not download the image
	// - "face_detection_failed": Face detection algorithm failed
	// - "no_face_detected": No faces found in the image
	// - "multiple_faces_no_clear_primary": Multiple faces detected but none is clearly the primary subject
	// - "poor_lighting_too_dark": Image is too dark
	// - "poor_lighting_too_bright": Image is too bright
	// - "poor_lighting_low_contrast": Image has insufficient contrast
	// - "poor_lighting_too_contrasty": Image has excessive contrast
	// - "face_too_small": Detected face is too small relative to image size
	// - "face_too_large": Detected face is too large (likely too close to camera)
	// - "face_aspect_ratio_unusual": Face has unusual proportions
	// - "unknown_error": Unexpected error occurred
	FaceCount  int
	Confidence float32
	Lighting   string
	Brightness float64
	Contrast   float64
}

// verifyLightingQuality analyzes the lighting conditions of the face region
func verifyLightingQuality(imgMat gocv.Mat, faceRect image.Rectangle) (string, float64, float64) {
	faceMat := imgMat.Region(faceRect)
	defer faceMat.Close()

	// Convert to grayscale for lighting analysis
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceMat, &gray, gocv.ColorBGRToGray)

	// Calculate brightness using a simple approach
	// Get the mean value of all pixels
	total := 0.0
	count := 0
	for y := 0; y < gray.Rows(); y++ {
		for x := 0; x < gray.Cols(); x++ {
			val := gray.GetUCharAt(y, x)
			total += float64(val)
			count++
		}
	}
	brightness := total / float64(count)

	// Calculate contrast (standard deviation)
	variance := 0.0
	for y := 0; y < gray.Rows(); y++ {
		for x := 0; x < gray.Cols(); x++ {
			val := gray.GetUCharAt(y, x)
			diff := float64(val) - brightness
			variance += diff * diff
		}
	}
	variance /= float64(count)
	contrast := variance

	// Determine lighting quality
	var lightingQuality string
	if brightness < 70 {
		lightingQuality = "too_dark"
	} else if brightness > 250 {
		lightingQuality = "too_bright"
	} else if contrast < 500 {
		lightingQuality = "low_contrast"
	} else if contrast > 3000 {
		lightingQuality = "too_contrasty"
	} else {
		lightingQuality = "good"
	}

	return lightingQuality, brightness, contrast
}

// findMostProminentFace selects the most prominent face when multiple faces are detected
func findMostProminentFace(faces []image.Rectangle, scores []float32, imgWidth, imgHeight int) (image.Rectangle, float32, bool) {
	if len(faces) == 0 {
		return image.Rectangle{}, 0, false
	}

	if len(faces) == 1 {
		return faces[0], scores[0], true
	}

	// Find the face closest to the center of the image
	centerX := imgWidth / 2
	centerY := imgHeight / 2

	bestIdx := 0
	bestScore := scores[0]

	for i, face := range faces {
		// Calculate center of face
		faceCenterX := face.Min.X + face.Dx()/2
		faceCenterY := face.Min.Y + face.Dy()/2

		// Calculate distance from image center
		dx := faceCenterX - centerX
		dy := faceCenterY - centerY
		distance := float64(dx*dx + dy*dy)

		// Calculate face size (area)
		faceArea := face.Dx() * face.Dy()

		// Combined score: confidence + centrality + size factor
		centralityScore := 1.0 - (distance / float64(imgWidth*imgWidth+imgHeight*imgHeight))
		sizeScore := float64(faceArea) / float64(imgWidth*imgHeight)
		combinedScore := float32(centralityScore*0.4 + sizeScore*0.3 + float64(scores[i])*0.3)

		log.Printf("Face %d: confidence=%.3f, distance=%.1f, area=%d, combined_score=%.3f",
			i+1, scores[i], distance, faceArea, combinedScore)

		if combinedScore > bestScore {
			bestScore = combinedScore
			bestIdx = i
		}
	}

	// Check if the best face is significantly better than others
	for i, face := range faces {
		if i == bestIdx {
			continue
		}

		faceCenterX := face.Min.X + face.Dx()/2
		faceCenterY := face.Min.Y + face.Dy()/2
		dx := faceCenterX - centerX
		dy := faceCenterY - centerY
		distance := float64(dx*dx + dy*dy)

		faceArea := face.Dx() * face.Dy()
		centralityScore := 1.0 - (distance / float64(imgWidth*imgWidth+imgHeight*imgHeight))
		sizeScore := float64(faceArea) / float64(imgWidth*imgHeight)
		combinedScore := float32(centralityScore*0.4 + sizeScore*0.3 + float64(scores[i])*0.3)

		// If another face is within 20% of the best score, consider it ambiguous
		if combinedScore > bestScore*0.8 {
			log.Printf("Multiple prominent faces detected - ambiguous selection")
			return image.Rectangle{}, 0, false
		}
	}

	log.Printf("Selected most prominent face: %v (score: %.3f)", faces[bestIdx], bestScore)
	return faces[bestIdx], bestScore, true
}

// VerifyImageQuality checks if an image is suitable for face verification
func VerifyImageQuality(imgURL string) ImageQualityResult {
	result := ImageQualityResult{
		IsValid: false,
		Reason:  "unknown_error",
	}

	// Download image
	imgBytes, err := loadImage(imgURL)
	if err != nil {
		result.Reason = "download_failed"
		return result
	}

	// Get detector from pool
	detector := globalPool.Get()
	defer globalPool.Put(detector)

	// Detect faces
	faces, scores, mat, err := detectSingleFace(imgBytes, detector)
	if err != nil {
		result.Reason = "face_detection_failed"
		return result
	}
	defer mat.Close()

	// Check if any faces were detected
	if len(faces) == 0 {
		result.Reason = "no_face_detected"
		result.FaceCount = 0
		return result
	}

	result.FaceCount = len(faces)

	// If multiple faces, try to find the most prominent one
	if len(faces) > 1 {
		imgHeight := mat.Rows()
		imgWidth := mat.Cols()

		selectedFace, confidence, found := findMostProminentFace(faces, scores, imgWidth, imgHeight)
		if !found {
			result.Reason = "multiple_faces_no_clear_primary"
			return result
		}

		// Use only the selected face
		faces = []image.Rectangle{selectedFace}
		result.Confidence = confidence
	} else {
		result.Confidence = scores[0]
	}

	// Verify lighting quality
	lightingQuality, brightness, contrast := verifyLightingQuality(mat, faces[0])
	result.Lighting = lightingQuality
	result.Brightness = brightness
	result.Contrast = contrast

	// Check if lighting is acceptable
	if lightingQuality != "good" {
		result.Reason = "poor_lighting_" + lightingQuality
		return result
	}

	// Additional quality checks
	faceRect := faces[0]
	faceWidth := faceRect.Dx()
	faceHeight := faceRect.Dy()

	// Check face size (should be reasonably large)
	imgArea := mat.Rows() * mat.Cols()
	faceArea := faceWidth * faceHeight
	faceAreaRatio := float64(faceArea) / float64(imgArea)

	if faceAreaRatio < 0.01 { // Face should be at least 1% of image
		result.Reason = "face_too_small"
		return result
	}

	if faceAreaRatio > 0.8 { // Face shouldn't be too large (likely too close)
		result.Reason = "face_too_large"
		return result
	}

	// Check face aspect ratio (should be roughly square-ish)
	aspectRatio := float64(faceWidth) / float64(faceHeight)
	if aspectRatio < 0.5 || aspectRatio > 10.0 {
		result.Reason = "face_aspect_ratio_unusual"
		return result
	}

	// All checks passed
	result.IsValid = true
	result.Reason = "valid"

	log.Printf("Image quality verification passed: %d face(s), lighting=%s, brightness=%.1f, contrast=%.1f",
		result.FaceCount, result.Lighting, result.Brightness, result.Contrast)

	return result
}

// TestImageQuality is a helper function to test image quality verification
func TestImageQuality(imgURL string) {
	result := VerifyImageQuality(imgURL)

	log.Printf("Image Quality Test Results:")
	log.Printf("  Valid: %t", result.IsValid)
	log.Printf("  Reason: %s", result.Reason)
	log.Printf("  Face Count: %d", result.FaceCount)
	log.Printf("  Confidence: %.3f", result.Confidence)
	log.Printf("  Lighting: %s", result.Lighting)
	log.Printf("  Brightness: %.1f", result.Brightness)
	log.Printf("  Contrast: %.1f", result.Contrast)
}
