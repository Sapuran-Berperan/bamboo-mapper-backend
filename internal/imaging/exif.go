package imaging

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	jis "github.com/dsoprea/go-jpeg-image-structure/v2"
)

// writeExif writes GPS EXIF metadata to JPEG bytes
func writeExif(jpegBytes []byte, opts ProcessOptions) ([]byte, error) {
	// Parse JPEG structure
	jmp := jis.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(jpegBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JPEG: %w", err)
	}

	sl := intfc.(*jis.SegmentList)

	// Get or create root IFD
	rootIb, err := sl.ConstructExifBuilder()
	if err != nil {
		// No existing EXIF, create new
		im, err := exifcommon.NewIfdMappingWithStandard()
		if err != nil {
			return nil, fmt.Errorf("failed to create IFD mapping: %w", err)
		}
		ti := exif.NewTagIndex()
		rootIb = exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.EncodeDefaultByteOrder)
	}

	// Get or create GPS IFD
	ifdPath := "IFD0/GPSInfo"
	gpsIb, err := exif.GetOrCreateIbFromRootIb(rootIb, ifdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get GPS IFD: %w", err)
	}

	// Parse latitude and longitude
	lat, err := strconv.ParseFloat(opts.Latitude, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}
	lng, err := strconv.ParseFloat(opts.Longitude, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}

	// Set GPS latitude
	latRef := "N"
	if lat < 0 {
		latRef = "S"
		lat = -lat
	}
	if err := gpsIb.SetStandardWithName("GPSLatitudeRef", latRef); err != nil {
		return nil, fmt.Errorf("failed to set GPSLatitudeRef: %w", err)
	}
	latRational := degreesToRational(lat)
	if err := gpsIb.SetStandardWithName("GPSLatitude", latRational); err != nil {
		return nil, fmt.Errorf("failed to set GPSLatitude: %w", err)
	}

	// Set GPS longitude
	lngRef := "E"
	if lng < 0 {
		lngRef = "W"
		lng = -lng
	}
	if err := gpsIb.SetStandardWithName("GPSLongitudeRef", lngRef); err != nil {
		return nil, fmt.Errorf("failed to set GPSLongitudeRef: %w", err)
	}
	lngRational := degreesToRational(lng)
	if err := gpsIb.SetStandardWithName("GPSLongitude", lngRational); err != nil {
		return nil, fmt.Errorf("failed to set GPSLongitude: %w", err)
	}

	// Set DateTimeOriginal if not already present (preserve existing if present)
	exifIfdPath := "IFD/Exif"
	exifIb, err := exif.GetOrCreateIbFromRootIb(rootIb, exifIfdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get Exif IFD: %w", err)
	}

	// Check if DateTimeOriginal exists
	_, err = exifIb.FindTagWithName("DateTimeOriginal")
	if err != nil {
		// Tag doesn't exist, set it
		dateTimeStr := opts.Timestamp.Format("2006:01:02 15:04:05")
		if err := exifIb.SetStandardWithName("DateTimeOriginal", dateTimeStr); err != nil {
			// Non-critical, just log
			fmt.Printf("Warning: failed to set DateTimeOriginal: %v\n", err)
		}
	}

	// Write EXIF back to JPEG
	if err := sl.SetExif(rootIb); err != nil {
		return nil, fmt.Errorf("failed to set EXIF: %w", err)
	}

	var outputBuf bytes.Buffer
	if err := sl.Write(&outputBuf); err != nil {
		return nil, fmt.Errorf("failed to write JPEG: %w", err)
	}

	return outputBuf.Bytes(), nil
}

// degreesToRational converts decimal degrees to EXIF rational format [degrees, minutes, seconds]
// Each value is represented as a rational (numerator/denominator)
func degreesToRational(degrees float64) []exifcommon.Rational {
	// Extract degrees, minutes, seconds
	deg := math.Floor(degrees)
	minFloat := (degrees - deg) * 60
	min := math.Floor(minFloat)
	sec := (minFloat - min) * 60

	// Convert to rational format (multiply by 1000000 for precision, then use as numerator)
	return []exifcommon.Rational{
		{Numerator: uint32(deg), Denominator: 1},
		{Numerator: uint32(min), Denominator: 1},
		{Numerator: uint32(sec * 1000000), Denominator: 1000000},
	}
}
