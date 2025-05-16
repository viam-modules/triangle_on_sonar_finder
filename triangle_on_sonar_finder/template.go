package triangle_on_sonar_finder

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
)

// TemplateFromImage represents a template created from an image
type TemplateFromImage struct {
	kernel       [][]float32
	kernelWidth  int
	kernelHeight int
	sumKernel    float32
}

// NewTemplateFromImage creates a new template from an image file
func NewTemplateFromImage(imagePath string) (*TemplateFromImage, error) {
	// Open and decode the image
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("error opening image: %v", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	// Convert to grayscale and normalize to [0,1]
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	kernel := make([][]float32, height)
	for i := range kernel {
		kernel[i] = make([]float32, width)
	}

	// Convert image to normalized float32 matrix
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get grayscale value and normalize to [0,1]
			c := img.At(x+bounds.Min.X, y+bounds.Min.Y)
			grayValue := float32(0)
			switch c := c.(type) {
			case color.Gray:
				grayValue = float32(c.Y)
			case color.Gray16:
				grayValue = float32(c.Y) / 65535.0
			default:
				// Convert to grayscale using standard formula
				r, g, b, _ := c.RGBA()
				grayValue = float32((r*299+g*587+b*114)/1000) / 65535.0
			}
			kernel[y][x] = grayValue
		}
	}

	// Calculate kernel mean
	var kernelSum float32 = 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			kernelSum += kernel[y][x]
		}
	}
	kernelMean := kernelSum / float32(height*width)

	// Normalize kernel by subtracting mean
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			kernel[y][x] = kernel[y][x] - kernelMean
		}
	}

	// Calculate sum of squared kernel values
	var sumKernel float32 = 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sumKernel += kernel[y][x] * kernel[y][x]
		}
	}

	return &TemplateFromImage{
		kernel:       kernel,
		kernelWidth:  width,
		kernelHeight: height,
		sumKernel:    sumKernel,
	}, nil
}

// FindMatch finds matches of the template in the given image matrix
func (t *TemplateFromImage) FindMatch(image [][]float32, stride int, threshold float32) []Match {
	height := len(image)
	if height == 0 {
		return nil
	}
	width := len(image[0])

	// Find matches
	var matches []Match
	for i := 0; i < height-t.kernelHeight; i += stride {
		for j := 0; j < width-t.kernelWidth; j += stride {
			// Calculate crop mean
			var cropSum float32 = 0
			for y := 0; y < t.kernelHeight; y++ {
				for x := 0; x < t.kernelWidth; x++ {
					cropSum += image[i+y][j+x]
				}
			}
			cropMean := cropSum / float32(t.kernelHeight*t.kernelWidth)

			// Calculate normalized correlation
			var sumProduct float32 = 0
			var sumCropSquared float32 = 0
			for y := 0; y < t.kernelHeight; y++ {
				for x := 0; x < t.kernelWidth; x++ {
					normalizedCrop := image[i+y][j+x] - cropMean
					sumProduct += normalizedCrop * t.kernel[y][x]
					sumCropSquared += normalizedCrop * normalizedCrop
				}
			}

			// Calculate correlation coefficient
			denominator := float32(math.Sqrt(float64(sumCropSquared * t.sumKernel)))
			if denominator > 0 {
				corr := sumProduct / denominator
				if corr > threshold {
					matches = append(matches, Match{
						X:      j,
						Y:      i,
						Width:  t.kernelWidth,
						Height: t.kernelHeight,
						Score:  corr,
					})
				}
			}
		}
	}

	return matches
}

// Match represents a found match with its position and correlation score
type Match struct {
	X      int
	Y      int
	Width  int
	Height int
	Score  float32
}

// GetBoundingBox returns the bounding box of the match
func (m *Match) GetBoundingBox() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{X: m.X, Y: m.Y},
		Max: image.Point{X: m.X + m.Width, Y: m.Y + m.Height},
	}
}
