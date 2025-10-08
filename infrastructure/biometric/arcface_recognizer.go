package biometric

import (
	"fmt"
	"image"
	"math"
	"os"
	"sync"

	"gateman.io/infrastructure/logger"
	"gocv.io/x/gocv"
)

// ArcFaceRecognizer provides face recognition using ArcFace model
type ArcFaceRecognizer struct {
	net          gocv.Net
	inputSize    image.Point
	modelsLoaded bool
	mutex        sync.RWMutex
}

// ArcFaceConfig holds configuration for ArcFace model
type ArcFaceConfig struct {
	ModelPath string
	InputSize image.Point
	Backend   gocv.NetBackendType
	Target    gocv.NetTargetType
}

// NewArcFaceRecognizer creates a new ArcFace recognizer
func NewArcFaceRecognizer(config ArcFaceConfig) *ArcFaceRecognizer {
	recognizer := &ArcFaceRecognizer{
		inputSize: config.InputSize,
	}

	// Load ArcFace model
	if err := recognizer.loadModel(config); err != nil {
		logger.Error("Failed to load ArcFace model", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return recognizer
	}

	recognizer.modelsLoaded = true
	logger.Info("ArcFace recognizer initialized successfully", logger.LoggerOptions{
		Key: "config",
		Data: map[string]interface{}{
			"model_path": config.ModelPath,
			"input_size": fmt.Sprintf("%dx%d", config.InputSize.X, config.InputSize.Y),
		},
	})

	return recognizer
}

// loadModel loads the ArcFace ONNX model
func (af *ArcFaceRecognizer) loadModel(config ArcFaceConfig) error {
	// Check if model file exists
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", config.ModelPath)
	}

	// Load the network
	af.net = gocv.ReadNet(config.ModelPath, "")
	if af.net.Empty() {
		return fmt.Errorf("failed to load ArcFace model from %s", config.ModelPath)
	}

	// Set backend and target
	af.net.SetPreferableBackend(config.Backend)
	af.net.SetPreferableTarget(config.Target)

	logger.Info("ArcFace model loaded successfully", logger.LoggerOptions{
		Key: "model_info",
		Data: map[string]interface{}{
			"path":    config.ModelPath,
			"backend": config.Backend,
			"target":  config.Target,
		},
	})

	return nil
}

// ExtractEmbedding extracts a 512-dimensional face embedding from a face image
func (af *ArcFaceRecognizer) ExtractEmbedding(face gocv.Mat) ([]float32, error) {
	af.mutex.RLock()
	defer af.mutex.RUnlock()

	if !af.modelsLoaded {
		return nil, fmt.Errorf("ArcFace model not loaded")
	}

	if face.Empty() {
		return nil, fmt.Errorf("empty face image")
	}

	// Preprocess face image
	preprocessed := af.preprocessFace(face)
	defer preprocessed.Close()

	// Create blob from image
	// ArcFace expects input: [1, 3, 112, 112] with normalization
	blob := gocv.BlobFromImage(
		preprocessed,
		1.0/127.5,                                    // Scale factor
		af.inputSize,                                  // Size
		gocv.NewScalar(127.5, 127.5, 127.5, 0),       // Mean subtraction
		true,                                          // Swap RB channels
		false,                                         // Crop
	)
	defer blob.Close()

	// Set input and run forward pass
	af.net.SetInput(blob, "")
	output := af.net.Forward("")
	defer output.Close()

	// Extract embedding vector (512 dimensions)
	embeddingSize := 512
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

// preprocessFace preprocesses face image for ArcFace model
func (af *ArcFaceRecognizer) preprocessFace(face gocv.Mat) gocv.Mat {
	// Resize to model input size (112x112 for ArcFace)
	resized := gocv.NewMat()
	gocv.Resize(face, &resized, af.inputSize, 0, 0, gocv.InterpolationLinear)

	// Convert to RGB if needed (ArcFace expects RGB)
	if resized.Channels() == 1 {
		rgb := gocv.NewMat()
		gocv.CvtColor(resized, &rgb, gocv.ColorGrayToBGR)
		resized.Close()
		return rgb
	}

	return resized
}

// CompareFaces compares two face embeddings using cosine similarity
func (af *ArcFaceRecognizer) CompareFaces(embedding1, embedding2 []float32) float64 {
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
			"match_threshold":   0.4, // Typical ArcFace threshold
		},
	})

	return similarity
}

// cosineSimilarity calculates cosine similarity between two embeddings
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	
	// Clamp to [-1, 1] range
	if similarity > 1.0 {
		similarity = 1.0
	}
	if similarity < -1.0 {
		similarity = -1.0
	}

	return similarity
}

// normalizeEmbedding performs L2 normalization on embedding
func normalizeEmbedding(embedding []float32) []float32 {
	norm := 0.0
	for _, val := range embedding {
		norm += float64(val * val)
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return embedding
	}

	normalized := make([]float32, len(embedding))
	for i, val := range embedding {
		normalized[i] = float32(float64(val) / norm)
	}

	return normalized
}

// calculateNorm calculates the L2 norm of an embedding
func calculateNorm(embedding []float32) float64 {
	norm := 0.0
	for _, val := range embedding {
		norm += float64(val * val)
	}
	return math.Sqrt(norm)
}

// Close releases resources
func (af *ArcFaceRecognizer) Close() error {
	af.mutex.Lock()
	defer af.mutex.Unlock()

	if !af.net.Empty() {
		if err := af.net.Close(); err != nil {
			return fmt.Errorf("failed to close ArcFace network: %v", err)
		}
	}

	af.modelsLoaded = false
	logger.Info("ArcFace recognizer closed")
	return nil
}

// GetDefaultArcFaceConfig returns default configuration for ArcFace
func GetDefaultArcFaceConfig() ArcFaceConfig {
	// Try to find model in common locations
	modelPaths := []string{
		"./models/arcface/arcface_r50.onnx",
		"./models/arcface/arcface_resnet50.onnx",
		"./models/arcface/mobilefacenet.onnx",
		"./models/arcface/arcface.onnx",
		"/usr/local/share/arcface/arcface.onnx",
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
		modelPath = "./models/arcface/arcface.onnx"
		logger.Info("ArcFace model not found, using default path", logger.LoggerOptions{
			Key:  "default_path",
			Data: modelPath,
		})
	}

	return ArcFaceConfig{
		ModelPath: modelPath,
		InputSize: image.Pt(112, 112), // Standard ArcFace input size
		Backend:   gocv.NetBackendDefault,
		Target:    gocv.NetTargetCPU,
	}
}

// IsModelAvailable checks if ArcFace model is available
func IsArcFaceModelAvailable() bool {
	config := GetDefaultArcFaceConfig()
	_, err := os.Stat(config.ModelPath)
	return err == nil
}
