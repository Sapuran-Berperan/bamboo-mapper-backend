package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"time"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/assets"
)

// ProcessOptions contains metadata for image processing
type ProcessOptions struct {
	Name      string
	Latitude  string
	Longitude string
	Timestamp time.Time
}

// ProcessImage applies watermark and EXIF to an image.
// Input: raw image bytes (JPEG or PNG)
// Output: processed JPEG bytes with watermark and EXIF
func ProcessImage(input []byte, opts ProcessOptions) ([]byte, error) {
	// Decode input image
	img, format, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Validate format
	if format != "jpeg" && format != "png" {
		return nil, fmt.Errorf("unsupported image format: %s (only JPEG and PNG supported)", format)
	}

	// Apply watermark
	watermarkedImg, err := applyWatermark(img, opts)
	if err != nil {
		log.Printf("WARNING: watermark failed: %v - continuing without watermark", err)
		watermarkedImg = img // Graceful degradation
	}

	// Encode to JPEG
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, watermarkedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	// Write EXIF metadata
	processedBytes, err := writeExif(buf.Bytes(), opts)
	if err != nil {
		log.Printf("WARNING: EXIF write failed: %v - continuing without EXIF", err)
		processedBytes = buf.Bytes() // Graceful degradation
	}

	return processedBytes, nil
}

// init validates that the font file is embedded correctly
func init() {
	if len(assets.RobotoRegular) == 0 {
		panic("Roboto font not embedded - check embed path")
	}
}
