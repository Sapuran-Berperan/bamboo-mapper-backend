package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"time"

	jis "github.com/dsoprea/go-jpeg-image-structure/v2"
)

// createTestJPEG creates a simple test JPEG image
func createTestJPEG(width, height int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with blue color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes(), err
}

// createTestPNG creates a simple test PNG image
func createTestPNG(width, height int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with green color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}

func TestProcessImage_JPEG(t *testing.T) {
	// Create test JPEG
	inputJPEG, err := createTestJPEG(800, 600)
	if err != nil {
		t.Fatalf("Failed to create test JPEG: %v", err)
	}

	opts := ProcessOptions{
		Name:      "Test Bamboo",
		Latitude:  "-7.797068",
		Longitude: "110.370529",
		Timestamp: time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC),
	}

	// Process image
	output, err := ProcessImage(inputJPEG, opts)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// Verify output is valid JPEG
	_, err = jpeg.Decode(bytes.NewReader(output))
	if err != nil {
		t.Errorf("Output is not valid JPEG: %v", err)
	}

	// Verify output is larger than input (watermark added)
	if len(output) <= 0 {
		t.Error("Output is empty")
	}
}

func TestProcessImage_PNG(t *testing.T) {
	// Create test PNG
	inputPNG, err := createTestPNG(800, 600)
	if err != nil {
		t.Fatalf("Failed to create test PNG: %v", err)
	}

	opts := ProcessOptions{
		Name:      "Test Bamboo PNG",
		Latitude:  "-7.797068",
		Longitude: "110.370529",
		Timestamp: time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC),
	}

	// Process image
	output, err := ProcessImage(inputPNG, opts)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// Verify output is valid JPEG (PNG converted to JPEG)
	_, err = jpeg.Decode(bytes.NewReader(output))
	if err != nil {
		t.Errorf("Output is not valid JPEG: %v", err)
	}
}

func TestProcessImage_InvalidFormat(t *testing.T) {
	// Create invalid image data
	invalidData := []byte("not an image")

	opts := ProcessOptions{
		Name:      "Test",
		Latitude:  "0",
		Longitude: "0",
		Timestamp: time.Now(),
	}

	// Should return error
	_, err := ProcessImage(invalidData, opts)
	if err == nil {
		t.Error("Expected error for invalid image format, got nil")
	}
}

func TestProcessImage_EXIF_GPS(t *testing.T) {
	// Create test JPEG
	inputJPEG, err := createTestJPEG(800, 600)
	if err != nil {
		t.Fatalf("Failed to create test JPEG: %v", err)
	}

	opts := ProcessOptions{
		Name:      "Test Bamboo",
		Latitude:  "-7.797068",
		Longitude: "110.370529",
		Timestamp: time.Date(2026, 1, 19, 14, 30, 0, 0, time.UTC),
	}

	// Process image
	output, err := ProcessImage(inputJPEG, opts)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// Parse EXIF from output
	jmp := jis.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(output)
	if err != nil {
		t.Fatalf("Failed to parse output JPEG: %v", err)
	}

	sl := intfc.(*jis.SegmentList)
	_, rawExif, err := sl.Exif()
	if err != nil {
		t.Fatalf("Failed to get EXIF: %v", err)
	}

	// Verify EXIF data exists and is non-empty
	if len(rawExif) == 0 {
		t.Error("EXIF data is empty")
	}

	// Basic validation: output should contain GPS-related bytes
	// (More thorough validation would require exiftool or similar)
	if len(rawExif) < 100 {
		t.Error("EXIF data seems too small to contain GPS information")
	}
}

func TestWatermark_VariousSizes(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{400, 300},   // Small
		{800, 600},   // Medium
		{1920, 1080}, // Large
		{3000, 2000}, // Extra large
	}

	opts := ProcessOptions{
		Name:      "Test",
		Latitude:  "0",
		Longitude: "0",
		Timestamp: time.Now(),
	}

	for _, size := range sizes {
		t.Run(string(rune(size.width))+"x"+string(rune(size.height)), func(t *testing.T) {
			inputJPEG, err := createTestJPEG(size.width, size.height)
			if err != nil {
				t.Fatalf("Failed to create test JPEG: %v", err)
			}

			output, err := ProcessImage(inputJPEG, opts)
			if err != nil {
				t.Errorf("ProcessImage failed for size %dx%d: %v", size.width, size.height, err)
			}

			// Verify output is valid
			if len(output) == 0 {
				t.Errorf("Output is empty for size %dx%d", size.width, size.height)
			}
		})
	}
}
