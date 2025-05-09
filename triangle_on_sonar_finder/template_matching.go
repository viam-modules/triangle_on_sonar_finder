package triangle_on_sonar_finder

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"gocv.io/x/gocv"
)

// Box represents a bounding box with position and dimensions
type Box struct {
	X      int
	Y      int
	Width  int
	Height int
}

// LoadTemplates loads all template images from a directory for pattern matching.
// Returns a slice of template images rather than a map since we only care about the matches.
func loadTemplates(templateDir string) ([]gocv.Mat, error) {
	templates := []gocv.Mat{}

	// Check if directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory not found: %s", templateDir)
	}

	// Define valid extensions
	validExtensions := []string{".png", ".jpg", ".jpeg", ".bmp", ".tif", ".tiff"}

	// Read files from directory
	files, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, fmt.Errorf("error reading template directory: %v", err)
	}

	// Process each file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		isValidExt := false
		for _, ext := range validExtensions {
			if strings.HasSuffix(strings.ToLower(filename), ext) {
				isValidExt = true
				break
			}
		}

		if isValidExt {
			templatePath := filepath.Join(templateDir, filename)
			template := gocv.IMRead(templatePath, gocv.IMReadGrayScale)
			if !template.Empty() {
				templates = append(templates, template)
			}
		}
	}

	fmt.Printf("Loaded %d templates from %s\n", len(templates), templateDir)
	return templates, nil
}

// CalculateIOU calculates Intersection over Union (IoU) between two boxes
func CalculateIOU(box1, box2 image.Rectangle) float64 {
	// Calculate intersection rectangle
	intersection := box1.Intersect(box2)

	// Check if there is an intersection
	if intersection.Empty() {
		return 0.0
	}

	// Calculate areas
	intersectionArea := intersection.Dx() * intersection.Dy()
	box1Area := box1.Dx() * box1.Dy()
	box2Area := box2.Dx() * box2.Dy()
	unionArea := box1Area + box2Area - intersectionArea

	// Calculate IoU
	return float64(intersectionArea) / float64(unionArea)
}

// NonMaxSuppression filters out overlapping boxes using non-maximum suppression
func NonMaxSuppression(boxes []image.Rectangle, iouThreshold float64) []image.Rectangle {
	if len(boxes) == 0 {
		return []image.Rectangle{}
	}

	filteredBoxes := []image.Rectangle{}

	for _, box := range boxes {
		shouldKeep := true

		// Check against all already filtered boxes
		for _, filteredBox := range filteredBoxes {
			if CalculateIOU(box, filteredBox) > iouThreshold {
				shouldKeep = false
				break
			}
		}

		if shouldKeep {
			filteredBoxes = append(filteredBoxes, box)
		}
	}

	return filteredBoxes
}

// FindMatchingPatterns performs template matching to find patterns in the image
func FindMatchingPatterns(img gocv.Mat, templates []gocv.Mat, threshold float32) []image.Rectangle {
	allBoxes := []image.Rectangle{}

	for _, templateImg := range templates {
		// Skip invalid templates
		if templateImg.Empty() {
			continue
		}

		templateH := templateImg.Rows()
		templateW := templateImg.Cols()

		// Perform template matching
		result := gocv.NewMat()
		gocv.MatchTemplate(img, templateImg, &result, gocv.TmCcoeffNormed, gocv.NewMat())

		// Find locations where the matching score exceeds the threshold
		for r := 0; r < result.Rows(); r++ {
			for c := 0; c < result.Cols(); c++ {
				val := result.GetFloatAt(r, c)
				if val >= threshold {
					// Create image.Rectangle
					rect := image.Rect(c, r, c+templateW, r+templateH)
					allBoxes = append(allBoxes, rect)
				}
			}
		}

		// Clean up
		result.Close()
	}

	// Apply non-maximum suppression to filter out overlapping boxes
	return NonMaxSuppression(allBoxes, 0.5)
}

// DrawBoundingBoxes draws bounding boxes around the matched regions
func DrawBoundingBoxes(img gocv.Mat, rectangles []image.Rectangle, thickness int) gocv.Mat {
	// Convert grayscale to color if needed
	resultImage := gocv.NewMat()
	if img.Channels() == 1 {
		gocv.CvtColor(img, &resultImage, gocv.ColorGrayToBGR)
	} else {
		img.CopyTo(&resultImage)
	}

	// Draw bounding boxes
	for _, rect := range rectangles {

		// Draw rectangle with red color
		color := color.RGBA{255, 0, 0, 255}
		gocv.Rectangle(&resultImage, rect, color, thickness)
	}

	return resultImage
}
