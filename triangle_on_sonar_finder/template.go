package triangle_on_sonar_finder

import (
	"image"
	"image/color"
	"math"

	"github.com/nfnt/resize"
)

// TemplateFromImage represents a template created from an image
type TemplateFromImage struct {
	kernel       [][]int16
	kernelWidth  int
	kernelHeight int
	sumKernel    float32
	originalSize image.Point
}

// NewTemplateFromImage creates a new template from an image file
func NewTemplateFromImage(img image.Image, scale float64) (*TemplateFromImage, error) {
	// Resize image, convert to grayscale and normalize to [0,1]
	originalSize := image.Point{X: img.Bounds().Dx(), Y: img.Bounds().Dy()}
	newWidth := uint(float64(originalSize.X) * scale)
	// Resize template proportionally to how we resize input image
	img = resizeImage(img, newWidth)
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	kernel := make([][]int16, height)
	for i := range kernel {
		kernel[i] = make([]int16, width)
	}

	// Convert image to normalized float32 matrix
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get grayscale value and normalize to [0,1]
			c := img.At(x+bounds.Min.X, y+bounds.Min.Y)
			grayValue := int16(0)
			switch c := c.(type) {
			case color.Gray:
				grayValue = int16(c.Y)
			default:
				grayValue = int16(color.GrayModel.Convert(c).(color.Gray).Y)
			}
			kernel[y][x] = grayValue
		}
	}

	// we do the mean so we're looking for shapes, not color similarity

	var kernelSum float32 = 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			kernelSum += float32(kernel[y][x])
		}
	}

	kernelMean := kernelSum / float32(height*width)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			kernel[y][x] = int16(float32(kernel[y][x]) - kernelMean)
		}
	}

	var sumKernel float32 = 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sumKernel += float32(kernel[y][x]) * float32(kernel[y][x])
		}
	}

	return &TemplateFromImage{
		kernel:       kernel,
		kernelWidth:  width,
		kernelHeight: height,
		sumKernel:    sumKernel,
		originalSize: originalSize,
	}, nil
}

// FindMatch finds matches of the template in the given image matrix and scales the matches to the original image size
func (t *TemplateFromImage) FindMatch(image [][]byte, stride int, threshold float32, scale float64) []Match {
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
			var cropSum int = 0
			for y := 0; y < t.kernelHeight; y++ {
				for x := 0; x < t.kernelWidth; x++ {
					cropSum += int(image[i+y][j+x])
				}
			}
			cropMean := cropSum / (t.kernelHeight * t.kernelWidth)

			sumProduct := 0
			sumCropSquared := 0

			for y := 0; y < t.kernelHeight; y++ {
				for x := 0; x < t.kernelWidth; x++ {
					normalizedCrop := int(image[i+y][j+x]) - cropMean
					sumProduct += normalizedCrop * int(t.kernel[y][x])
					sumCropSquared += normalizedCrop * normalizedCrop
				}
			}

			// Calculate correlation coefficient
			denominator := float32(math.Sqrt(float64(float32(sumCropSquared) * t.sumKernel)))
			if denominator > 0 {
				corr := float32(sumProduct) / denominator
				if corr > threshold {
					matches = append(matches, Match{
						X:      int(float64(j) * 1 / scale),
						Y:      int(float64(i) * 1 / scale),
						Width:  t.originalSize.X,
						Height: t.originalSize.Y,
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
func resizeImage(img image.Image, newWidth uint) image.Image {
	return resize.Resize(newWidth, 0, img, resize.Lanczos3) //lanczos3 is best for downsampling
}
