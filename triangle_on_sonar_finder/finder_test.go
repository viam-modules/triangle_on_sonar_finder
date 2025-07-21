package triangle_on_sonar_finder

import (
	"image"
	"os"
	"testing"

	"go.viam.com/test"
)

func openImage(fn string) (image.Image, error) {
	file, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}

func TestTriangleOnSonarFinder(t *testing.T) {
	scale := 0.5
	templates, err := loadTemplates(scale)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, len(templates), test.ShouldEqual, 3)

	for i, tmpl := range templates {
		t.Logf("Template %d: Original size: %v, Resized size: %dx%d",
			i, tmpl.originalSize, tmpl.kernelWidth, tmpl.kernelHeight)
	}

	img, err := openImage("inputs/input_3.jpg")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, img.Bounds().Empty(), test.ShouldBeFalse)

	imgMatrix := ImageToMatrix(img, scale)
	detections := findTriangles(templates, imgMatrix, 2, .48, scale) // 0.5 confidence threshold
	for i, det := range detections {
		t.Logf("Detection %d: Box=%v, Score=%f", i, det.BoundingBox(), det.Score())
	}
	test.That(t, len(detections), test.ShouldEqual, 3)

}

func BenchmarkTriangls(t *testing.B) {
	scale := 0.7
	templates, err := loadTemplates(scale)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, len(templates), test.ShouldEqual, 3)

	img, err := openImage("inputs/input_2.jpg")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, img.Bounds().Empty(), test.ShouldBeFalse)

	t.ResetTimer()

	for t.Loop() {
		imgMatrix := ImageToMatrix(img, scale)
		detections := findTriangles(templates, imgMatrix, 2, .5, scale)
		test.That(t, len(detections), test.ShouldEqual, 1)
	}

}

// TestTemplateResizing tests that templates are resized proportionally
func TestTemplateResizing(t *testing.T) {
	// Use an actual template image
	img, err := openImage("templates/triangle_1.png")
	test.That(t, err, test.ShouldBeNil)

	originalSize := img.Bounds().Size()
	t.Logf("Original template size: %v", originalSize)

	// Create scaled template image
	template, err := NewTemplateFromImage(img, 0.8)
	test.That(t, err, test.ShouldBeNil)

	test.That(t, template.originalSize, test.ShouldResemble, originalSize)

	// Verify kernel dimensions are scaled correctly
	expectedWidth := int(float64(originalSize.X) * 0.8)
	expectedHeight := int(float64(originalSize.Y) * 0.8)
	test.That(t, template.kernelWidth, test.ShouldEqual, expectedWidth)
	test.That(t, template.kernelHeight, test.ShouldEqual, expectedHeight)
}

// tests that detected coordinates are properly scaled
func TestCoordinateScaling(t *testing.T) {
	templates, err := loadTemplates(0.5)
	test.That(t, err, test.ShouldBeNil)

	img, err := openImage("inputs/input_1.jpg")
	test.That(t, err, test.ShouldBeNil)
	originalSize := img.Bounds().Size()
	t.Logf("Original input image size: %v", originalSize)
	// Process image
	matrix := ImageToMatrix(img, 0.5)
	t.Logf("Resized input image size: %dx%d", len(matrix[0]), len(matrix))
	detections := findTriangles(templates, matrix, 2, 0.65, 0.5)

	for i, det := range detections {
		box := det.BoundingBox()
		t.Logf("Detection %d: Box=%v, Score=%f", i, box, det.Score())

		// Verify coordinates are in original image space
		test.That(t, box.Min.X, test.ShouldBeLessThan, originalSize.X)
		test.That(t, box.Min.Y, test.ShouldBeLessThan, originalSize.Y)
		test.That(t, box.Max.X, test.ShouldBeLessThan, originalSize.X)
		test.That(t, box.Max.Y, test.ShouldBeLessThan, originalSize.Y)

		// verify the box size matches one of our templates
		boxWidth := box.Dx()
		boxHeight := box.Dy()

		// check if this detection matches any template size
		foundMatchingTemplate := false
		for _, template := range templates {
			if boxWidth == template.originalSize.X && boxHeight == template.originalSize.Y {
				foundMatchingTemplate = true
				break
			}
		}
		test.That(t, foundMatchingTemplate, test.ShouldBeTrue)
	}
}
