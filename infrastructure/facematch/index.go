package facematch

import (
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"sync"

	"gocv.io/x/gocv"
)

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

func detectSingleFace(imgBytes []byte, detector *FaceDetector) ([]image.Rectangle, gocv.Mat, error) {
	imgMat, err := gocv.IMDecode(imgBytes, gocv.IMReadColor)
	if err != nil {
		return nil, imgMat, err
	}
	if imgMat.Empty() {
		return nil, imgMat, err
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
		return faces, imgMat, nil
	}

	detectionSize := detections.Size()

	// Validate detection output dimensions
	if len(detectionSize) < 2 {
		log.Printf("Invalid detection output dimensions: %v", detectionSize)
		return faces, imgMat, nil
	}

	// YuNet output format: [1, num_detections, 15] where 15 = [x, y, w, h, confidence, landmarks...]
	rows := detectionSize[1]
	if rows <= 0 {
		log.Printf("No detection rows found")
		return faces, imgMat, nil
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

	// If multiple faces detected, select the one with highest confidence
	if len(faces) > 1 {
		log.Printf("Multiple faces detected (%d), selecting the best one", len(faces))
		bestIdx := 0
		bestScore := scores[0]
		for i, score := range scores {
			if score > bestScore {
				bestScore = score
				bestIdx = i
			}
		}
		log.Printf("Selected face %d with confidence %.3f", bestIdx+1, bestScore)
		faces = []image.Rectangle{faces[bestIdx]}
	}

	log.Printf("Final result: %d faces in image (%dx%d)", len(faces), imgWidth, imgHeight)

	return faces, imgMat, nil
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

func Compare(img1 string, img2 string) bool {
	var wg sync.WaitGroup
	type faceResult struct {
		faces []image.Rectangle
		mat   gocv.Mat
		err   error
	}
	results := make([]faceResult, 2)

	detector, err := NewFaceDetector()
	if err != nil {
		log.Printf("Error initializing face detector: %v\n", err)
		return false
	}
	defer detector.Close()

	urls := []string{img1, img2}
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			imgBytes, err := downloadImage(urls[idx])
			if err != nil {
				results[idx] = faceResult{err: err}
				return
			}
			faces, mat, err := detectSingleFace(imgBytes, detector)
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

// TestFaceDetection is a helper function to test face detection on a single image
func TestFaceDetection(imgURL string) {
	detector, err := NewFaceDetector()
	if err != nil {
		log.Printf("Error initializing face detector: %v\n", err)
		return
	}
	defer detector.Close()

	imgBytes, err := downloadImage(imgURL)
	if err != nil {
		log.Printf("Error downloading image: %v\n", err)
		return
	}

	faces, mat, err := detectSingleFace(imgBytes, detector)
	if err != nil {
		log.Printf("Error detecting faces: %v\n", err)
		return
	}
	defer mat.Close()

	log.Printf("Test result: Found %d faces in image", len(faces))
	for i, face := range faces {
		log.Printf("Face %d: %v", i+1, face)
	}
}
