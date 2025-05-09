package triangle_on_sonar_finder

import (
	"image"
	"os"
	"testing"

	"go.viam.com/test"
	"gocv.io/x/gocv"
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
	templates, err := loadTemplates("/Users/robin@viam.com/triangle_finder/patterns")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, len(templates), test.ShouldEqual, 3)
	img, err := openImage("/Users/robin@viam.com/triangle_on_sonar_finder/triangle_on_sonar_finder/inputs/input_1.jpg")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, img.Bounds().Empty(), test.ShouldBeFalse)
	grayImg, ok := img.(*image.Gray)
	if !ok {
		// If not already grayscale, convert to grayscale
		bounds := img.Bounds()
		grayImg = image.NewGray(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				grayImg.Set(x, y, img.At(x, y))
			}
		}
	}
	mat, err := gocv.ImageGrayToMatGray(grayImg)
	test.That(t, err, test.ShouldBeNil)
	detections := FindMatchingPatterns(mat, templates, 0.7)
	test.That(t, len(detections), test.ShouldEqual, 3)

	// // Create a new image for visualization
	// resultImage := DrawBoundingBoxes(mat, detections, 2)
	// // Save the result image to the output directory if provided
	// // Construct the output file path
	// outputPath := filepath.Join("./", "result_input_1.jpg")

	// // Save the image
	// ok = gocv.IMWrite(outputPath, resultImage)
	// test.That(t, ok, test.ShouldBeTrue)

}
