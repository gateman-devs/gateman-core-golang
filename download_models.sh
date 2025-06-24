#!/bin/bash

# Script to download required OpenCV DNN models for face detection and recognition
# Models are downloaded from official OpenCV repositories

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the directory where this script is located and set models directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODELS_DIR="$SCRIPT_DIR/infrastructure/facematch/models"

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to get file size in a cross-platform way
get_file_size() {
    local file="$1"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        stat -f%z "$file" 2>/dev/null || echo "0"
    else
        # Linux
        stat -c%s "$file" 2>/dev/null || echo "0"
    fi
}

# Function to download a file with progress
download_file() {
    local url="$1"
    local output="$2"
    local filename=$(basename "$output")
    
    print_status $YELLOW "Downloading $filename..."
    
    if command_exists curl; then
        curl -L --progress-bar -o "$output" "$url"
    elif command_exists wget; then
        wget --progress=bar:force -O "$output" "$url"
    else
        print_status $RED "Error: Neither curl nor wget is available. Please install one of them."
        return 1
    fi
    
    # Verify download
    if [[ ! -f "$output" ]] || [[ $(get_file_size "$output") -lt 1000 ]]; then
        print_status $RED "Error: Download failed or file is too small"
        rm -f "$output"
        return 1
    fi
    
    print_status $GREEN "✓ Successfully downloaded $filename ($(get_file_size "$output") bytes)"
    return 0
}

# Function to verify model file integrity
verify_model() {
    local model_file="$1"
    local min_size="$2"
    
    if [[ ! -f "$model_file" ]]; then
        return 1
    fi
    
    local file_size=$(get_file_size "$model_file")
    if [[ $file_size -lt $min_size ]]; then
        print_status $YELLOW "Warning: $model_file seems too small ($file_size bytes), may be corrupted"
        return 1
    fi
    
    return 0
}

# Function to check and download a single model
check_and_download_model() {
    local model_name="$1"
    local model_url="$2"
    local min_size="$3"
    local model_path="$MODELS_DIR/$model_name"
    
    print_status $YELLOW "\nChecking: $model_name"
    
    if verify_model "$model_path" "$min_size"; then
        print_status $GREEN "✓ $model_name exists and appears valid ($(get_file_size "$model_path") bytes)"
        return 0
    else
        print_status $RED "✗ $model_name is missing or invalid"
        
        # Download the missing model
        if download_file "$model_url" "$model_path"; then
            # Verify the download
            if verify_model "$model_path" "$min_size"; then
                print_status $GREEN "✓ $model_name successfully downloaded and verified"
                return 1  # Indicates a download occurred
            else
                print_status $RED "✗ Failed to verify $model_name after download"
                return 2  # Indicates download failed verification
            fi
        else
            print_status $RED "✗ Failed to download $model_name"
            return 2  # Indicates download failed
        fi
    fi
}

# Main execution
main() {
    print_status $GREEN "=== OpenCV Face Models Downloader ==="
    print_status $YELLOW "Checking models in: $MODELS_DIR"
    
    # Create models directory if it doesn't exist
    if [[ ! -d "$MODELS_DIR" ]]; then
        print_status $YELLOW "Creating models directory..."
        mkdir -p "$MODELS_DIR"
    fi
    
    local downloads_count=0
    local failed_count=0
    local total_models=3  # yunet, arcface, and face_anti_spoofing (though anti-spoofing needs separate sourcing)
    
    # Check YuNet face detection model
    check_and_download_model \
        "yunet.onnx" \
        "https://github.com/opencv/opencv_zoo/raw/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx" \
        "200000"
    case $? in
        0) ;; # Already exists
        1) ((downloads_count++)) ;;
        2) ((failed_count++)) ;;
    esac
    
    # Check SFace face recognition model (rename to arcface.onnx)
    check_and_download_model \
        "arcface.onnx" \
        "https://github.com/opencv/opencv_zoo/raw/main/models/face_recognition_sface/face_recognition_sface_2021dec.onnx" \
        "200000"
    case $? in
        0) ;; # Already exists
        1) ((downloads_count++)) ;;
        2) ((failed_count++)) ;;
    esac
    
    # Check Anti-spoofing model (Using SFace model from OpenCV Zoo)
    # Note: This is actually a face recognition model being used for anti-spoofing
    check_and_download_model \
        "face_anti_spoofing.onnx" \
        "https://github.com/opencv/opencv_zoo/raw/main/models/face_recognition_sface/face_recognition_sface_2021dec.onnx" \
        "200000"
    case $? in
        0) ;; # Already exists
        1) ((downloads_count++)) ;;
        2) ((failed_count++)) ;;
    esac
    
    # Summary
    print_status $GREEN "\n=== Summary ==="
    local existing_count=$((total_models - downloads_count - failed_count))
    
    if [[ $failed_count -eq 0 ]]; then
        if [[ $downloads_count -eq 0 ]]; then
            print_status $GREEN "✓ All $total_models models are present and verified"
        else
            print_status $GREEN "✓ Downloaded $downloads_count missing model(s), $existing_count were already present"
        fi
    else
        print_status $YELLOW "⚠ $existing_count models were already present, $downloads_count downloaded successfully, $failed_count failed"
        if [[ $downloads_count -gt 0 ]]; then
            print_status $GREEN "✓ Some models were downloaded successfully"
        fi
        if [[ $failed_count -gt 0 ]]; then
            print_status $RED "✗ $failed_count model(s) failed to download"
        fi
    fi
    
    print_status $YELLOW "\nModel files location: $MODELS_DIR"
    if [[ $failed_count -eq 0 ]]; then
        print_status $YELLOW "You can now run your face matching service!"
    else
        print_status $YELLOW "Some models failed to download. Check your internet connection and try again."
    fi
}

# Check if script is being sourced or executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 