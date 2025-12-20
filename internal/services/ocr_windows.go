//go:build windows

package services

import (
	"errors"
)

// OCRService handles optical character recognition (stub for Windows)
type OCRService struct{}

// NewOCRService creates a new OCR service (not available on Windows)
func NewOCRService() (*OCRService, error) {
	return nil, errors.New("OCR service is not available on Windows - run in Docker container")
}

// ProcessImage processes an image from bytes and returns extracted text
func (s *OCRService) ProcessImage(imageBytes []byte) (*OCRResult, error) {
	return nil, errors.New("OCR service is not available on Windows")
}

// ProcessImageFromPath processes an image from a file path
func (s *OCRService) ProcessImageFromPath(imagePath string) (*OCRResult, error) {
	return nil, errors.New("OCR service is not available on Windows")
}

// Close releases OCR resources
func (s *OCRService) Close() error {
	return nil
}
