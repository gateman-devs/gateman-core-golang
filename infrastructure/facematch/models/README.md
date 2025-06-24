# Face Recognition Models

This directory contains the required ONNX model files for face detection, recognition, and anti-spoofing.

## Required Models

- `yunet.onnx` - YuNet face detection model
- `arcface.onnx` - ArcFace feature extraction model  
- `face_anti_spoofing.onnx` - Anti-spoofing detection model

## Setup

The model files are not committed to git due to their large size. To download them, run:

```bash
# From project root
./download_models.sh
```

This script will automatically download all required models to this directory.

## Note

These files are ignored by git (see `.gitignore`) to keep the repository size manageable. Always use the download script to obtain the models. 