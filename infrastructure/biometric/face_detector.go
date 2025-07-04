package biometric

import (
	"encoding/base64"
	"fmt"
	"image"
	"net/http"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

type FaceDetector struct {
	net         *gocv.Net
	initialized bool
	mu          sync.RWMutex
}

func NewFaceDetector() *FaceDetector {
	return &FaceDetector{}
}

func (fd *FaceDetector) Initialize(modelPath string) error {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	if fd.initialized {
		return nil
	}

	net := gocv.ReadNet(modelPath, "")
	if net.Empty() {
		net.Close() // Clean up if empty
		return fmt.Errorf("failed to load YuNet model from %s", modelPath)
	}

	fd.net = &net
	fd.initialized = true
	return nil
}

func (fd *FaceDetector) DetectFace(input string) FaceDetectionResult {
	start := time.Now()

	fd.mu.RLock()
	defer fd.mu.RUnlock()

	if !fd.initialized {
		return FaceDetectionResult{
			Success:     false,
			Error:       "face detector not initialized",
			ProcessTime: time.Since(start).Milliseconds(),
		}
	}

	img, err := fd.loadImage(input)
	if err != nil {
		return FaceDetectionResult{
			Success:     false,
			Error:       fmt.Sprintf("failed to load image: %v", err),
			ProcessTime: time.Since(start).Milliseconds(),
		}
	}
	defer img.Close()

	if img.Empty() {
		return FaceDetectionResult{
			Success:     false,
			Error:       "loaded image is empty",
			ProcessTime: time.Since(start).Milliseconds(),
		}
	}

	// Prepare image for YuNet (requires specific input size)
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(img, &resized, image.Pt(320, 320), 0, 0, gocv.InterpolationLinear)

	// Create blob from image
	blob := gocv.BlobFromImage(resized, 1.0, image.Pt(320, 320), gocv.NewScalar(0, 0, 0, 0), false, false)
	defer blob.Close()

	// Set input to the network
	fd.net.SetInput(blob, "")

	// Run forward pass
	output := fd.net.Forward("")
	defer output.Close()

	// Parse detections
	faces := fd.parseDetections(output, img.Cols(), img.Rows())

	result := FaceDetectionResult{
		Success:     true,
		ProcessTime: time.Since(start).Milliseconds(),
		Metadata:    fd.generateMetadata(img, faces),
	}

	if len(faces) > 0 {
		// Get the face with highest confidence
		bestFace := faces[0]
		for _, face := range faces {
			if face.Score > bestFace.Score {
				bestFace = face
			}
		}

		result.FaceFound = true
		result.FaceRegion = bestFace
		result.Confidence = bestFace.Score
	} else {
		result.FaceFound = false
		result.Confidence = 0.0
	}

	return result
}

func (fd *FaceDetector) parseDetections(output gocv.Mat, imgWidth, imgHeight int) []FaceBox {
	var faces []FaceBox

	// YuNet output format: [batch_size, num_detections, 15]
	// Each detection: [x1, y1, x2, y2, landmarks..., confidence]
	for i := 0; i < output.Rows(); i++ {
		confidence := output.GetFloatAt(i, 14) // confidence is at index 14

		if confidence > 0.5 { // confidence threshold
			x1 := output.GetFloatAt(i, 0)
			y1 := output.GetFloatAt(i, 1)
			x2 := output.GetFloatAt(i, 2)
			y2 := output.GetFloatAt(i, 3)

			// Convert normalized coordinates to pixel coordinates
			x := int(x1 * float32(imgWidth))
			y := int(y1 * float32(imgHeight))
			width := int((x2 - x1) * float32(imgWidth))
			height := int((y2 - y1) * float32(imgHeight))

			// Ensure coordinates are within image bounds
			if x < 0 {
				x = 0
			}
			if y < 0 {
				y = 0
			}
			if x+width > imgWidth {
				width = imgWidth - x
			}
			if y+height > imgHeight {
				height = imgHeight - y
			}

			faces = append(faces, FaceBox{
				X:      x,
				Y:      y,
				Width:  width,
				Height: height,
				Score:  float64(confidence),
			})
		}
	}

	return faces
}

func (fd *FaceDetector) loadImage(input string) (gocv.Mat, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return fd.loadImageFromURL(input)
	}
	return fd.loadImageFromBase64(input)
}

func (fd *FaceDetector) loadImageFromURL(url string) (gocv.Mat, error) {
	done := make(chan struct {
		img gocv.Mat
		err error
	})

	go func() {
		resp, err := http.Get(url)
		if err != nil {
			done <- struct {
				img gocv.Mat
				err error
			}{gocv.NewMat(), err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			done <- struct {
				img gocv.Mat
				err error
			}{gocv.NewMat(), fmt.Errorf("HTTP error: %d", resp.StatusCode)}
			return
		}

		buf := make([]byte, resp.ContentLength)
		_, err = resp.Body.Read(buf)
		if err != nil {
			done <- struct {
				img gocv.Mat
				err error
			}{gocv.NewMat(), err}
			return
		}

		img, err := gocv.IMDecode(buf, gocv.IMReadColor)
		done <- struct {
			img gocv.Mat
			err error
		}{img, err}
	}()

	result := <-done
	return result.img, result.err
}

func (fd *FaceDetector) loadImageFromBase64(data string) (gocv.Mat, error) {
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

func (fd *FaceDetector) generateMetadata(img gocv.Mat, faces []FaceBox) FaceMetadata {
	metadata := FaceMetadata{
		ImageSize:   fmt.Sprintf("%dx%d", img.Cols(), img.Rows()),
		Quality:     "good",
		Lighting:    "adequate",
		Orientation: "upright",
	}

	if len(faces) > 0 {
		face := faces[0]
		metadata.FaceSize = fmt.Sprintf("%dx%d", face.Width, face.Height)

		// Basic quality assessment
		faceArea := face.Width * face.Height
		imageArea := img.Cols() * img.Rows()
		faceRatio := float64(faceArea) / float64(imageArea)

		if faceRatio < 0.01 {
			metadata.Quality = "low"
			metadata.Warnings = append(metadata.Warnings, "Face is very small in the image")
		} else if faceRatio > 0.5 {
			metadata.Quality = "excellent"
		}

		if face.Score < 0.7 {
			metadata.Warnings = append(metadata.Warnings, "Low face detection confidence")
		}
	} else {
		metadata.FaceSize = "0x0"
		metadata.Quality = "poor"
		metadata.Warnings = append(metadata.Warnings, "No face detected")
	}

	return metadata
}

func (fd *FaceDetector) Close() {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	if fd.net != nil {
		fd.net.Close()
		fd.net = nil
	}
	fd.initialized = false
}
