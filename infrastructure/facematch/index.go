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
	modelPath := "infrastructure/facematch/models/face_detection_yunet_2023mar.onnx"

	net := gocv.ReadNet(modelPath, "")
	if net.Empty() {
		return nil, fmt.Errorf("failed to load YuNet model: %s", modelPath)
	}
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

	blob := gocv.BlobFromImage(imgMat, 1.0, image.Pt(320, 320), gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	detector.net.SetInput(blob, "")
	detections := detector.net.Forward("")
	defer detections.Close()

	var faces []image.Rectangle
	rows := detections.Size()[1]
	if len(detections.Size()) == 3 && detections.Size()[2] == 15 {
		for i := 0; i < rows; i++ {
			score := detections.GetFloatAt(0, i*15+4)
			if score > 0.5 {
				x := int(detections.GetFloatAt(0, i*15+0))
				y := int(detections.GetFloatAt(0, i*15+1))
				w := int(detections.GetFloatAt(0, i*15+2))
				h := int(detections.GetFloatAt(0, i*15+3))
				rect := image.Rect(x, y, x+w, y+h)
				faces = append(faces, rect)
			}
		}
	}
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
