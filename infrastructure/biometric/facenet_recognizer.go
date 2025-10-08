package biometric

import (
	"fmt"
	"image"
	"os"
	"sync"

	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

// FaceNetRecognizer provides face recognition using FaceNet/SFace model
type FaceNetRecognizer struct {
	net          gocv.Net
	inputSize    image.Point
	modelsLoaded bool
	mutex        sync.RWMutex
}

// FaceNetConfig holds configuration for FaceNet model
type FaceNetConfig struct {
	ModelPath string
	InputSize image.Point
	Backend   gocv.NetBackendType
	Target    gocv.NetTargetType
}

// NewFaceNetRecognizer creates a new FaceNet recognizer
func NewFaceNetRecognizer(config FaceNetConfig) *FaceNetRecognizer {
	recognizer := &FaceNetRecognizer{
		inputSize: config.InputSize,
	}

	// Load FaceNet model
	if err := recognizer.loadModel(config); err != nil {
		logger.Error("Failed to load FaceNet model", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return recognizer
	}

	recognizer.modelsLoaded = true
	logger.Info("FaceNet recognizer initialized successfully", logger.LoggerOptions{
		Key: "config",
		Data: map[string]interface{}{
			"model_path": config.ModelPath,
			"input_size": fmt.Sprintf("%dx%d", config.InputSize.X, config.InputSize.Y),
		},
	})

	return recognizer
}

// loadModel loads the FaceNet ONNX model
func (fn *FaceNetRecognizer) loadModel(config FaceNetConfig) error {
	// Check if model file exists
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", config.ModelPath)
	}

	// Load the network
	fn.net = gocv.ReadNet(config.ModelPath, "")
	if fn.net.Empty() {
		return fmt.Errorf("failed to load FaceNet model from %s", config.ModelPath)
	}

	// Set backend and target
	fn.net.SetPreferableBackend(config.Backend)
	fn.net.SetPreferableTarget(config.Target)

	logger.Info("FaceNet model loaded successfully", logger.LoggerOptions{
		Key: "model_info",
		Data: map[string]interface{}{
			"path":    config.ModelPath,
			"backend": config.Backend,
			"target":  config.Target,
		},
	})

	return nil
}

// ExtractEmbedding extracts a 128-dimensional face embedding from a face image
func (fn *FaceNetRecognizer) ExtractEmbedding(face gocv.Mat) ([]float32, error) {
	fn.mutex.RLock()
	defer fn.mutex.RUnlock()

	if !fn.modelsLoaded {
		return nil, fmt.Errorf("FaceNet model not loaded")
	}

	if face.Empty() {
		return nil, fmt.Errorf("empty face image")
	}

	// Preprocess face image
	preprocessed := fn.preprocessFace(face)
	defer preprocessed.Close()

	// Create blob from image
	// SFace expects input: [1, 3, 112, 112] with normalization
	blob := gocv.BlobFromImage(
		preprocessed,
		1.0/127.5,                                    // Scale factor
		fn.inputSize,                                  // Size
		gocv.NewScalar(127.5, 127.5, 127.5, 0),       // Mean subtraction
		true,                                          // Swap RB channels
		false,                                         // Crop
	)
	defer blob.Close()

	// Set input and run forward pass
	fn.net.SetInput(blob, "")
	output := fn.net.Forward("")
	defer output.Close()

	// Extract embedding vector (128 dimensions for SFace)
	embeddingSize := 128
	embedding := make([]float32, embeddingSize)
	
	// Read embedding values
	for i := 0; i < embeddingSize; i++ {
		embedding[i] = output.GetFloatAt(0, i)
	}

	// Normalize embedding (L2 normalization)
	embedding = normalizeEmbedding(embedding)

	logger.Info("Face embedding extracted", logger.LoggerOptions{
		Key: "embedding_info",
		Data: map[string]interface{}{
			"dimensions": embeddingSize,
			"norm":       calculateNorm(embedding),
		},
	})

	return embedding, nil
}

// preprocessFace preprocesses face image for FaceNet model
func (fn *FaceNetRecognizer) preprocessFace(face gocv.Mat) gocv.Mat {
	// Resize to model input size (112x112 for SFace)
	resized := gocv.NewMat()
	gocv.Resize(face, &resized, fn.inputSize, 0, 0, gocv.InterpolationLinear)

	// Convert to RGB if needed (FaceNet expects RGB)
	if resized.Channels() == 1 {
		rgb := gocv.NewMat()
		gocv.CvtColor(resized, &rgb, gocv.ColorGrayToBGR)
		resized.Close()
		return rgb
	}

	return resized
}

// CompareFaces compares two face embeddings using cosine similarity
func (fn *FaceNetRecognizer) CompareFaces(embedding1, embedding2 []float32) float64 {
	if len(embedding1) != len(embedding2) {
		logger.Error("Embedding dimension mismatch", logger.LoggerOptions{
			Key: "dimensions",
			Data: map[string]interface{}{
				"embedding1": len(embedding1),
				"embedding2": len(embedding2),
			},
		})
		return 0.0
	}

	// Calculate cosine similarity
	similarity := cosineSimilarity(embedding1, embedding2)

	logger.Info("Face comparison completed", logger.LoggerOptions{
		Key: "similarity",
		Data: map[string]interface{}{
			"cosine_similarity": similarity,
			"match_threshold":   0.363, // SFace recommended threshold
		},
	})

	return similarity
}

// Close releases resources
func (fn *FaceNetRecognizer) Close() error {
	fn.mutex.Lock()
	defer fn.mutex.Unlock()

	if !fn.net.Empty() {
		if err := fn.net.Close(); err != nil {
			return fmt.Errorf("failed to close FaceNet network: %v", err)
		}
	}

	fn.modelsLoaded = false
	logger.Info("FaceNet recognizer closed")
	return nil
}

// GetDefaultFaceNetConfig returns default configuration for FaceNet
func GetDefaultFaceNetConfig() FaceNetConfig {
	// Try to find model in common locations
	modelPaths := []string{
		"./models/facenet/facenet.onnx",
		"./models/facenet/sface.onnx",
		"/usr/local/share/facenet/facenet.onnx",
	}

	modelPath := ""
	for _, path := range modelPaths {
		if _, err := os.Stat(path); err == nil {
			modelPath = path
			break
		}
	}

	// If no model found, use default path
	if modelPath == "" {
		modelPath = "./models/facenet/facenet.onnx"
		logger.Info("FaceNet model not found, using default path", logger.LoggerOptions{
			Key:  "default_path",
			Data: modelPath,
		})
	}

	return FaceNetConfig{
		ModelPath: modelPath,
		InputSize: image.Pt(112, 112), // Standard SFace input size
		Backend:   gocv.NetBackendDefault,
		Target:    gocv.NetTargetCPU,
	}
}

// IsFaceNetModelAvailable checks if FaceNet model is available
func IsFaceNetModelAvailable() bool {
	config := GetDefaultFaceNetConfig()
	_, err := os.Stat(config.ModelPath)
	return err == nil
}
