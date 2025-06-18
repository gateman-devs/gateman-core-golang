package facematch

import (
	"fmt"
	"image"
	"math"
	"sync"

	"gateman.io/application/utils"
	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

type FaceDetector struct {
	detectionNet   gocv.Net
	recognitionNet gocv.Net
}

type DetectorPool struct {
	detectors chan *FaceDetector
}

func (fd *FaceDetector) Close() {
	fd.detectionNet.Close()
	fd.recognitionNet.Close()
}

func loadImage(input *string) ([]byte, error) {
	if utils.IsBase64Image(*input) {
		logger.Info("loading image from base64 data")
		return utils.DecodeBase64Image(*input)
	}

	logger.Info("loading image from URL", logger.LoggerOptions{
		Key: "url", Data: input,
	})
	return utils.DownloadImage(*input)
}

func NewFaceDetector() (*FaceDetector, error) {
	detectionModelPath := "infrastructure/facematch/models/yunet.onnx"
	recognitionModelPath := "infrastructure/facematch/models/arcface.onnx"

	detectionNet := gocv.ReadNet(detectionModelPath, "")
	if detectionNet.Empty() {
		return nil, fmt.Errorf("failed to load YuNet detection model: %s", detectionModelPath)
	}

	recognitionNet := gocv.ReadNet(recognitionModelPath, "")
	if recognitionNet.Empty() {
		return nil, fmt.Errorf("failed to load ArcFace recognition model: %s", recognitionModelPath)
	}

	logger.Info("YuNet detection model and ArcFace recognition model loaded successfully")

	return &FaceDetector{
		detectionNet:   detectionNet,
		recognitionNet: recognitionNet,
	}, nil
}

var globalPool *DetectorPool
var poolSize = 15

func Init() {
	globalPool = &DetectorPool{
		detectors: make(chan *FaceDetector, poolSize),
	}

	for i := 0; i < poolSize; i++ {
		detector, err := NewFaceDetector()
		if err != nil {
			logger.Error(fmt.Sprintf("failed to initialize detector %d", i), logger.LoggerOptions{
				Key: "err", Data: err,
			})
		}
		globalPool.detectors <- detector
	}

	logger.Info("initialized detector pool", logger.LoggerOptions{
		Key: "poolSize", Data: poolSize,
	})
}

func (dp *DetectorPool) Get() *FaceDetector {
	return <-dp.detectors
}

func (dp *DetectorPool) Put(detector *FaceDetector) {
	dp.detectors <- detector
}

// iou calculates Intersection over Union between two rectangles
func iou(a, b image.Rectangle) float32 {
	inter := a.Intersect(b)
	if inter.Empty() {
		return 0
	}
	interArea := float32(inter.Dx() * inter.Dy())
	unionArea := float32(a.Dx()*a.Dy() + b.Dx()*b.Dy() - int(interArea))
	return interArea / unionArea
}

// nms applies Non-Maximum Suppression to merge overlapping detections
func nms(rects []image.Rectangle, scores []float32, threshold float32) ([]image.Rectangle, []float32) {
	if len(rects) == 0 {
		return rects, scores
	}

	picked := []int{}
	used := make([]bool, len(rects))

	for i := 0; i < len(rects); i++ {
		if used[i] {
			continue
		}
		maxIdx := i
		for j := i + 1; j < len(rects); j++ {
			if used[j] {
				continue
			}
			if scores[j] > scores[maxIdx] {
				maxIdx = j
			}
		}
		used[maxIdx] = true
		picked = append(picked, maxIdx)

		// Suppress overlapping detections
		for j := 0; j < len(rects); j++ {
			if used[j] {
				continue
			}
			if iou(rects[maxIdx], rects[j]) > threshold {
				used[j] = true
			}
		}
	}

	nmsRects := []image.Rectangle{}
	nmsScores := []float32{}
	for _, idx := range picked {
		nmsRects = append(nmsRects, rects[idx])
		nmsScores = append(nmsScores, scores[idx])
	}

	return nmsRects, nmsScores
}

func detectSingleFace(imgBytes []byte, detector *FaceDetector) ([]image.Rectangle, []float32, gocv.Mat, error) {
	fmt.Println("detectSingleFace running")
	imgMat, err := gocv.IMDecode(imgBytes, gocv.IMReadColor)
	if err != nil {
		logger.Error("failed to decode image", logger.LoggerOptions{
			Key: "err", Data: err,
		})
		return nil, nil, imgMat, err
	}
	if imgMat.Empty() {
		logger.Error("decoded image is empty")
		return nil, nil, imgMat, fmt.Errorf("decoded image is empty")
	}

	imgHeight := imgMat.Rows()
	imgWidth := imgMat.Cols()

	logger.Info("processing image", logger.LoggerOptions{
		Key: "width", Data: imgWidth,
	}, logger.LoggerOptions{
		Key: "height", Data: imgHeight,
	})

	// YuNet expects input size of 320x320, BGR, scale=1.0, no mean subtraction
	inputSize := image.Pt(320, 320)
	blob := gocv.BlobFromImage(imgMat, 1.0, inputSize, gocv.NewScalar(0, 0, 0, 0), false, false)
	defer blob.Close()

	logger.Info("created blob with shape", logger.LoggerOptions{
		Key: "shape", Data: blob.Size(),
	})

	detector.detectionNet.SetInput(blob, "")
	detections := detector.detectionNet.Forward("")
	defer detections.Close()

	var faces []image.Rectangle
	var scores []float32

	if detections.Empty() {
		logger.Info("model produced empty output")
		return faces, scores, imgMat, nil
	}

	detectionSize := detections.Size()

	if len(detectionSize) < 2 {
		logger.Info("invalid detection output dimensions", logger.LoggerOptions{
			Key: "detectionSize", Data: detectionSize,
		})
		return faces, scores, imgMat, nil
	}

	// YuNet output format: [batch_size, num_detections, 15]
	rows := detectionSize[1]
	if rows <= 0 {
		logger.Info("no detection rows found")
		return faces, scores, imgMat, nil
	}

	if len(detectionSize) >= 2 {
		cols := 15
		if len(detectionSize) == 3 {
			cols = detectionSize[2]
		}

		for i := 0; i < rows; i++ {
			if cols < 5 {
				continue
			}

			x_center_norm := detections.GetFloatAt(0, i*cols+0)
			y_center_norm := detections.GetFloatAt(0, i*cols+1)
			width_norm := detections.GetFloatAt(0, i*cols+2)
			height_norm := detections.GetFloatAt(0, i*cols+3)
			confidence := detections.GetFloatAt(0, i*cols+4)

			// Log all raw detection values for debugging/monitoring
			logger.Info("raw detection", logger.LoggerOptions{
				Key: "x_center_norm", Data: x_center_norm,
			}, logger.LoggerOptions{
				Key: "y_center_norm", Data: y_center_norm,
			}, logger.LoggerOptions{
				Key: "width_norm", Data: width_norm,
			}, logger.LoggerOptions{
				Key: "height_norm", Data: height_norm,
			}, logger.LoggerOptions{
				Key: "confidence", Data: confidence,
			})

			if confidence > 0.7 {
				x_center := x_center_norm * float32(imgWidth)
				y_center := y_center_norm * float32(imgHeight)
				width := width_norm * float32(imgWidth)
				height := height_norm * float32(imgHeight)

				x := int(x_center - width/2)
				y := int(y_center - height/2)
				w := int(width)
				h := int(height)

				// Boundary checks
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

				// Minimum face size for production (30x30)
				fmt.Println("face sizes")
				fmt.Println("w", w)
				fmt.Println("h", h)
				if w > 30 && h > 30 {
					rect := image.Rect(x, y, x+w, y+h)
					faces = append(faces, rect)
					scores = append(scores, confidence)
					logger.Info("added face rectangle", logger.LoggerOptions{
						Key: "rect", Data: rect,
					}, logger.LoggerOptions{
						Key: "score", Data: confidence,
					})
				} else {
					logger.Info("face too small, skipping", logger.LoggerOptions{
						Key: "width", Data: w,
					}, logger.LoggerOptions{
						Key: "height", Data: h,
					})
				}
			}
		}
	} else {
		logger.Info("unexpected detection output format", logger.LoggerOptions{
			Key: "detectionSize", Data: detectionSize,
		})
	}

	// Apply NMS to merge overlapping detections
	if len(faces) > 1 {
		logger.Info("applying NMS to merge overlapping detections", logger.LoggerOptions{
			Key: "beforeNMS", Data: len(faces),
		})
		faces, scores = nms(faces, scores, 0.5) // IoU threshold of 0.5
		logger.Info("NMS completed", logger.LoggerOptions{
			Key: "afterNMS", Data: len(faces),
		})
	}

	return faces, scores, imgMat, nil
}

// extractFaceEmbedding extracts face embedding using ArcFace model
func extractFaceEmbedding(imgMat gocv.Mat, faceRect image.Rectangle, detector *FaceDetector) ([]float32, error) {
	faceMat := imgMat.Region(faceRect)
	defer faceMat.Close()

	// Resize face to 112x112 (standard input size for ArcFace)
	resized := gocv.NewMat()
	gocv.Resize(faceMat, &resized, image.Pt(112, 112), 0, 0, gocv.InterpolationLinear)
	defer resized.Close()

	// ArcFace preprocessing: Convert BGR to RGB and normalize
	rgb := gocv.NewMat()
	gocv.CvtColor(resized, &rgb, gocv.ColorBGRToRGB)
	defer rgb.Close()

	// ArcFace normalization: (pixel_value - 127.5) / 128.0
	// This is the standard preprocessing for most ArcFace models
	blob := gocv.BlobFromImage(rgb, 1.0/128.0, image.Pt(112, 112), gocv.NewScalar(127.5, 127.5, 127.5, 0), false, false)
	defer blob.Close()

	// Set input and get embedding
	detector.recognitionNet.SetInput(blob, "")
	embedding := detector.recognitionNet.Forward("")
	defer embedding.Close()

	if embedding.Empty() {
		return nil, fmt.Errorf("failed to extract face embedding")
	}

	// Convert embedding to float32 slice
	embeddingSize := embedding.Size()
	if len(embeddingSize) != 2 || embeddingSize[0] != 1 {
		return nil, fmt.Errorf("unexpected embedding shape: %v", embeddingSize)
	}

	embeddingDim := embeddingSize[1]
	result := make([]float32, embeddingDim)

	for i := 0; i < embeddingDim; i++ {
		result[i] = embedding.GetFloatAt(0, i)
	}

	// L2 normalization - essential for ArcFace embeddings
	norm := float32(0)
	for _, val := range result {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range result {
			result[i] /= norm
		}
	}

	return result, nil
}

// cosineSimilarity calculates cosine similarity between two normalized vectors
func cosineSimilarity(vec1, vec2 []float32) float32 {
	if len(vec1) != len(vec2) {
		return 0
	}

	dotProduct := float32(0)
	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
	}

	// Since vectors are already normalized, dot product equals cosine similarity
	return dotProduct
}

func Compare(img1 *string, img2 *string) bool {
	logger.Info("starting face comparison")
	var wg sync.WaitGroup
	type faceResult struct {
		faces []image.Rectangle
		mat   gocv.Mat
		err   error
	}
	results := make([]faceResult, 2)

	urls := []*string{img1, img2}
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			imgBytes, err := loadImage(urls[idx])
			if err != nil {
				results[idx] = faceResult{err: err}
				return
			}

			detector := globalPool.Get()
			defer globalPool.Put(detector)

			faces, _, mat, err := detectSingleFace(imgBytes, detector)
			results[idx] = faceResult{faces: faces, mat: mat, err: err}
		}(i)
	}
	wg.Wait()

	for i, res := range results {
		if res.err != nil {
			logger.Error(fmt.Sprintf("error processing image %d", i+1), logger.LoggerOptions{
				Key: "err", Data: res.err,
			})
			return false
		}
		defer res.mat.Close()
		if len(res.faces) != 1 {
			logger.Error(fmt.Sprintf("image %d: expected 1 face, found %d", i+1, len(res.faces)), logger.LoggerOptions{
				Key: "faceCount", Data: len(res.faces),
			})
			return false
		}
	}

	// Extract face embeddings using ArcFace
	var embeddings [2][]float32
	var embeddingWg sync.WaitGroup
	embeddingWg.Add(2)

	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer embeddingWg.Done()
			detector := globalPool.Get()
			defer globalPool.Put(detector)

			embedding, err := extractFaceEmbedding(results[idx].mat, results[idx].faces[0], detector)
			if err != nil {
				logger.Error(fmt.Sprintf("failed to extract embedding for image %d", idx+1), logger.LoggerOptions{
					Key: "err", Data: err,
				})
				return
			}
			embeddings[idx] = embedding
		}(i)
	}
	embeddingWg.Wait()

	// Check if both embeddings were extracted successfully
	if embeddings[0] == nil || embeddings[1] == nil {
		logger.Error("failed to extract embeddings from one or both images")
		return false
	}

	similarity := cosineSimilarity(embeddings[0], embeddings[1])
	logger.Info("face comparison completed", logger.LoggerOptions{
		Key: "similarity", Data: similarity,
	})

	// ArcFace similarity threshold: 0.4-0.5 is typically used for verification
	// 0.6 was too high - most genuine matches fall between 0.3-0.7
	return similarity > 0.4
}

func TestFaceDetection(imgURL *string) {
	detector := globalPool.Get()
	defer globalPool.Put(detector)

	imgBytes, err := loadImage(imgURL)
	if err != nil {
		logger.Error("error loading image", logger.LoggerOptions{
			Key: "err", Data: err,
		})
		return
	}

	faces, scores, mat, err := detectSingleFace(imgBytes, detector)
	if err != nil {
		logger.Error("error detecting faces", logger.LoggerOptions{
			Key: "err", Data: err,
		})
		return
	}
	defer mat.Close()

	logger.Info("test result: found faces in image", logger.LoggerOptions{
		Key: "faceCount", Data: len(faces),
	})
	for i, face := range faces {
		confidence := float32(0.0)
		if i < len(scores) {
			confidence = scores[i]
		}
		logger.Info(fmt.Sprintf("face %d", i+1), logger.LoggerOptions{
			Key: "rect", Data: face,
		}, logger.LoggerOptions{
			Key: "confidence", Data: confidence,
		})
	}
}

// ImageQualityResult represents the result of image quality verification
type ImageQualityResult struct {
	IsValid    bool
	Reason     string
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

	// Calculate brightness and contrast using OpenCV functions for better accuracy
	mean := gocv.NewMat()
	stddev := gocv.NewMat()
	defer mean.Close()
	defer stddev.Close()

	gocv.MeanStdDev(gray, &mean, &stddev)

	brightness := mean.GetDoubleAt(0, 0)
	contrast := stddev.GetDoubleAt(0, 0)

	// More realistic lighting thresholds based on 8-bit grayscale (0-255)
	var lightingQuality string
	if brightness < 50 { // Very dark
		lightingQuality = "too_dark"
	} else if brightness > 220 { // Very bright/overexposed
		lightingQuality = "too_bright"
	} else if contrast < 15 { // Very low contrast (flat lighting)
		lightingQuality = "low_contrast"
	} else if contrast > 80 { // Very high contrast (harsh lighting)
		lightingQuality = "too_contrasty"
	} else {
		lightingQuality = "good"
	}

	return lightingQuality, brightness, contrast
}

// findMostProminentFace selects the most prominent face when multiple faces are detected
// Returns false if there are multiple equally dominant faces
func findMostProminentFace(faces []image.Rectangle, scores []float32, imgWidth, imgHeight int) (image.Rectangle, float32, bool) {
	if len(faces) == 0 {
		return image.Rectangle{}, 0, false
	}

	// If only one face after NMS, accept it as the dominant face
	if len(faces) == 1 {
		logger.Info("single face detected after NMS - accepting as dominant", logger.LoggerOptions{
			Key: "rect", Data: faces[0],
		}, logger.LoggerOptions{
			Key: "score", Data: scores[0],
		})
		return faces[0], scores[0], true
	}

	// Calculate combined scores for all faces
	type faceScore struct {
		index int
		score float32
		area  int
	}

	centerX := imgWidth / 2
	centerY := imgHeight / 2
	maxDistance := math.Sqrt(float64(imgWidth*imgWidth + imgHeight*imgHeight))

	var faceScores []faceScore

	for i, face := range faces {
		// Calculate face area
		faceArea := face.Dx() * face.Dy()

		// Calculate distance from center
		faceCenterX := face.Min.X + face.Dx()/2
		faceCenterY := face.Min.Y + face.Dy()/2
		dx := float64(faceCenterX - centerX)
		dy := float64(faceCenterY - centerY)
		distance := math.Sqrt(dx*dx + dy*dy)
		centralityScore := 1.0 - (distance / maxDistance)

		// Combined scoring: size (60%), confidence (25%), centrality (15%)
		// Emphasize size more as larger faces are typically the main subject
		sizeScore := float64(faceArea) / float64(imgWidth*imgHeight)
		combinedScore := float32(sizeScore*0.6 + float64(scores[i])*0.25 + centralityScore*0.15)

		faceScores = append(faceScores, faceScore{
			index: i,
			score: combinedScore,
			area:  faceArea,
		})

		logger.Info(fmt.Sprintf("face %d analysis", i+1), logger.LoggerOptions{
			Key: "confidence", Data: scores[i],
		}, logger.LoggerOptions{
			Key: "area", Data: faceArea,
		}, logger.LoggerOptions{
			Key: "combinedScore", Data: combinedScore,
		})
	}

	// Sort by combined score (highest first)
	for i := 0; i < len(faceScores)-1; i++ {
		for j := i + 1; j < len(faceScores); j++ {
			if faceScores[j].score > faceScores[i].score {
				faceScores[i], faceScores[j] = faceScores[j], faceScores[i]
			}
		}
	}

	bestFace := faceScores[0]

	// More lenient threshold for multiple faces after NMS
	// If there's a second face with >85% of the best score, reject
	// This is more lenient than the previous 75% threshold
	dominanceThreshold := bestFace.score * 0.85

	for i := 1; i < len(faceScores); i++ {
		if faceScores[i].score > dominanceThreshold {
			logger.Info("multiple dominant faces detected after NMS - no clear primary face", logger.LoggerOptions{
				Key: "bestScore", Data: bestFace.score,
			}, logger.LoggerOptions{
				Key: "competitorScore", Data: faceScores[i].score,
			}, logger.LoggerOptions{
				Key: "threshold", Data: dominanceThreshold,
			})
			return image.Rectangle{}, 0, false
		}
	}

	// More lenient size comparison for faces after NMS
	// The dominant face should be at least 1.2x larger than others
	// unless it has very high confidence (>0.85)
	if scores[bestFace.index] < 0.85 {
		for i := 1; i < len(faceScores); i++ {
			otherArea := faceScores[i].area
			if float64(bestFace.area) < float64(otherArea)*1.2 {
				logger.Info("faces are too similar in size after NMS - no clear primary face", logger.LoggerOptions{
					Key: "bestArea", Data: bestFace.area,
				}, logger.LoggerOptions{
					Key: "competitorArea", Data: otherArea,
				})
				return image.Rectangle{}, 0, false
			}
		}
	}

	logger.Info("selected dominant face after NMS", logger.LoggerOptions{
		Key: "rect", Data: faces[bestFace.index],
	}, logger.LoggerOptions{
		Key: "score", Data: bestFace.score,
	}, logger.LoggerOptions{
		Key: "originalConfidence", Data: scores[bestFace.index],
	})

	return faces[bestFace.index], bestFace.score, true
}

func VerifyImageQuality(imgURL *string) ImageQualityResult {
	result := ImageQualityResult{
		IsValid: false,
		Reason:  "unknown_error",
	}

	imgBytes, err := loadImage(imgURL)
	if err != nil {
		result.Reason = "load_failed"
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

	// Additional quality checks with more realistic parameters
	faceRect := faces[0]
	faceWidth := faceRect.Dx()
	faceHeight := faceRect.Dy()

	// Log face dimensions for debugging
	logger.Info("face dimensions", logger.LoggerOptions{
		Key: "width", Data: faceWidth,
	}, logger.LoggerOptions{
		Key: "height", Data: faceHeight,
	})

	// Check face size - more reasonable thresholds
	imgArea := mat.Rows() * mat.Cols()
	faceArea := faceWidth * faceHeight
	faceAreaRatio := float64(faceArea) / float64(imgArea)

	if faceAreaRatio < 0.005 { // Face should be at least 0.5% of image
		result.Reason = "face_too_small"
		return result
	}

	if faceAreaRatio > 0.6 { // Face shouldn't be more than 60% (allows for closer shots)
		result.Reason = "face_too_large"
		return result
	}

	// Check face aspect ratio - more lenient for real faces
	// Handle edge cases where height might be very small or negative
	if faceHeight <= 0 {
		logger.Info("invalid face height detected", logger.LoggerOptions{
			Key: "height", Data: faceHeight,
		})
		// If height is invalid, skip aspect ratio check but log it
	} else {
		aspectRatio := float64(faceWidth) / float64(faceHeight)
		logger.Info("face aspect ratio", logger.LoggerOptions{
			Key: "aspectRatio", Data: aspectRatio,
		})
		if aspectRatio < 0.5 || aspectRatio > 4.0 { // Much more lenient range for face proportions
			result.Reason = "face_aspect_ratio_unusual"
			return result
		}
	}

	// All checks passed
	result.IsValid = true
	result.Reason = "valid"

	logger.Info("image quality verification passed", logger.LoggerOptions{
		Key: "faceCount", Data: result.FaceCount,
	}, logger.LoggerOptions{
		Key: "lighting", Data: result.Lighting,
	}, logger.LoggerOptions{
		Key: "brightness", Data: result.Brightness,
	}, logger.LoggerOptions{
		Key: "contrast", Data: result.Contrast,
	})

	return result
}

func TestImageQuality(imgURL *string) {
	result := VerifyImageQuality(imgURL)

	logger.Info("image quality test results", logger.LoggerOptions{
		Key: "valid", Data: result.IsValid,
	}, logger.LoggerOptions{
		Key: "reason", Data: result.Reason,
	}, logger.LoggerOptions{
		Key: "faceCount", Data: result.FaceCount,
	}, logger.LoggerOptions{
		Key: "confidence", Data: result.Confidence,
	}, logger.LoggerOptions{
		Key: "lighting", Data: result.Lighting,
	}, logger.LoggerOptions{
		Key: "brightness", Data: result.Brightness,
	}, logger.LoggerOptions{
		Key: "contrast", Data: result.Contrast,
	})
}
