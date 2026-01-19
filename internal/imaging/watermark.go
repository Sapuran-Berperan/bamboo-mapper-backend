package imaging

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/assets"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
)

// applyWatermark draws a watermark strip at the bottom of the image
func applyWatermark(img image.Image, opts ProcessOptions) (image.Image, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate watermark strip height: 8% of image height (min 40px, max 80px)
	stripHeight := int(math.Round(float64(height) * 0.08))
	if stripHeight < 40 {
		stripHeight = 40
	}
	if stripHeight > 80 {
		stripHeight = 80
	}

	// Create drawing context
	dc := gg.NewContextForImage(img)

	// Draw semi-transparent black background strip
	dc.SetColor(color.RGBA{R: 0, G: 0, B: 0, A: 179}) // 70% opacity (179/255 ≈ 0.70)
	dc.DrawRectangle(0, float64(height-stripHeight), float64(width), float64(stripHeight))
	dc.Fill()

	// Load and parse font
	font, err := truetype.Parse(assets.RobotoRegular)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	// Calculate font size: proportional to strip height
	fontSize := float64(stripHeight) * 0.4 // 40% of strip height
	face := truetype.NewFace(font, &truetype.Options{
		Size: fontSize,
	})
	dc.SetFontFace(face)

	// Format watermark text: "{name} | {lat}, {lng} | {YYYY-MM-DD HH:mm}"
	timestamp := opts.Timestamp.Format("2006-01-02 15:04")
	text := fmt.Sprintf("%s | %s, %s | %s", opts.Name, opts.Latitude, opts.Longitude, timestamp)

	// Set text color to white
	dc.SetColor(color.White)

	// Draw text centered vertically in the strip, with 10px horizontal padding
	textX := 10.0
	textY := float64(height-stripHeight/2) + fontSize*0.35 // Vertical center with baseline adjustment
	dc.DrawString(text, textX, textY)

	return dc.Image(), nil
}
