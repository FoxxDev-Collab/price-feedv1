package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/otiai10/gosseract/v2"
)

// OCRService handles optical character recognition
type OCRService struct {
	client *gosseract.Client
}

// OCRResult contains the OCR processing result
type OCRResult struct {
	Text       string
	Confidence int
}

// NewOCRService creates a new OCR service
func NewOCRService() (*OCRService, error) {
	client := gosseract.NewClient()

	// Set English language
	if err := client.SetLanguage("eng"); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to set OCR language: %w", err)
	}

	// Configure for receipt scanning
	// PSM 6 = Assume a single uniform block of text
	if err := client.SetPageSegMode(gosseract.PSM_SINGLE_BLOCK); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to set page segmentation mode: %w", err)
	}

	return &OCRService{
		client: client,
	}, nil
}

// ProcessImage processes an image from bytes and returns extracted text
func (s *OCRService) ProcessImage(imageBytes []byte) (*OCRResult, error) {
	// Create a temporary file for the image
	tmpFile, err := os.CreateTemp("", "receipt-*.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write image bytes to temp file
	if _, err := tmpFile.Write(imageBytes); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Close to flush writes
	tmpFile.Close()

	// Process the image
	return s.ProcessImageFromPath(tmpFile.Name())
}

// ProcessImageFromPath processes an image from a file path
func (s *OCRService) ProcessImageFromPath(imagePath string) (*OCRResult, error) {
	// Verify file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("image file not found: %s", imagePath)
	}

	// Get absolute path
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Set the image
	if err := s.client.SetImage(absPath); err != nil {
		return nil, fmt.Errorf("failed to set image: %w", err)
	}

	// Get the text
	text, err := s.client.Text()
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	return &OCRResult{
		Text:       text,
		Confidence: 0, // gosseract doesn't expose confidence directly in simple mode
	}, nil
}

// Close releases OCR resources
func (s *OCRService) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
