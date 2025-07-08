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
	templates, err := loadTemplates()
	test.That(t, err, test.ShouldBeNil)
	test.That(t, len(templates), test.ShouldEqual, 3)

	img, err := openImage("inputs/input_1.jpg")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, img.Bounds().Empty(), test.ShouldBeFalse)

	imgMatrix := ImageToMatrix(img)
	detections := findTriangles(templates, imgMatrix, 2, .7)
	test.That(t, len(detections), test.ShouldEqual, 2)

}

func BenchmarkTriangls(t *testing.B) {
	templates, err := loadTemplates()
	test.That(t, err, test.ShouldBeNil)
	test.That(t, len(templates), test.ShouldEqual, 3)

	img, err := openImage("inputs/input_1.jpg")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, img.Bounds().Empty(), test.ShouldBeFalse)

	t.ResetTimer()

	for t.Loop() {
		imgMatrix := ImageToMatrix(img)
		detections := findTriangles(templates, imgMatrix, 2, .7)
		test.That(t, len(detections), test.ShouldEqual, 2)
	}

}
