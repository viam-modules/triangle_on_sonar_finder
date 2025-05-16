package triangle_on_sonar_finder

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
)

// Box represents a bounding box with position and dimensions
type Box struct {
	X      int
	Y      int
	Width  int
	Height int
}

// loadTemplates loads template images from the specified directory and returns
// a slice of TemplateFromImage objects. Each template is normalized. Returns an error if the directory cannot be accessed or if
// no valid templates are found.
func loadTemplates(templateDir string) ([]TemplateFromImage, error) {
	templates := []TemplateFromImage{}

	// Check if directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory not found: %s", templateDir)
	}

	// Define valid extensions
	validExtensions := []string{".png", ".jpg", ".jpeg"}

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
			template, err := NewTemplateFromImage(templatePath)
			if err != nil {
				continue
			}
			templates = append(templates, *template)
		}
	}
	return templates, nil
}

// ImageToMatrix converts a grayscale image to a 2D float32 matrix
func ImageToMatrix(img image.Image) [][]float32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create the 2D matrix
	matrix := make([][]float32, height)
	for i := range matrix {
		matrix[i] = make([]float32, width)
	}

	// Convert each pixel to float32
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get the grayscale value and convert to float32
			grayValue := float32(img.At(x+bounds.Min.X, y+bounds.Min.Y).(color.YCbCr).Y)
			matrix[y][x] = grayValue
		}
	}

	return matrix
}
