package triangle_on_sonar_finder

import (
	"context"
	"sort"

	"image"

	"github.com/pkg/errors"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"
	vis "go.viam.com/rdk/vision"
	"go.viam.com/rdk/vision/classification"
	objdet "go.viam.com/rdk/vision/objectdetection"
	"go.viam.com/rdk/vision/viscapture"
)

const (
	ModelName = "triangle-finder"
)

var (
	Model            = resource.NewModel("viam", "vision", ModelName)
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterService(vision.API, Model, resource.Registration[vision.Service, *TriangleFinderConfig]{
		Constructor: newTriangleFinder,
	})
}

// TriangleFinderConfig contains the configuration for the triangle finder.
type TriangleFinderConfig struct {
	// Camera is the name of the camera to use for triangle detection.
	Camera string `json:"camera_name"`

	// PathToTemplatesDirectory is the path to the directory containing template images.
	PathToTemplatesDirectory string `json:"path_to_templates_directory"`

	// Threshold is the matching threshold value used for template matching.
	Threshold float32 `json:"threshold,omitempty"`
}

// TODO: implement Validate
func (cfg TriangleFinderConfig) Validate(path string) ([]string, error) {
	return []string{cfg.Camera}, nil
}

type myTriangleFinder struct {
	resource.AlwaysRebuild

	name      resource.Name
	logger    logging.Logger
	cam       camera.Camera
	config    *TriangleFinderConfig
	templates []TemplateFromImage
}

func newTriangleFinder(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (vision.Service, error) {
	newConf, err := resource.NativeConfig[*TriangleFinderConfig](conf)
	if err != nil {
		return nil, errors.Errorf("failed to parse config for %s got: %s", ModelName, err)
	}

	tf := &myTriangleFinder{
		name:   conf.ResourceName(),
		logger: logger,
		config: newConf,
	}
	// get camera
	tf.cam, err = camera.FromDependencies(deps, newConf.Camera)
	if err != nil {
		return nil, errors.Errorf("failed to get camera from dependencies for %s got: %s", ModelName, err)
	}

	// load template images
	tf.templates, err = loadTemplates(newConf.PathToTemplatesDirectory)
	if err != nil {
		return nil, errors.Errorf("failed to load template images for %s got: %s", ModelName, err)
	}

	if len(tf.templates) == 0 {
		return nil, errors.Errorf("no valid template found in %s", newConf.PathToTemplatesDirectory)
	}
	return tf, nil
}

func (tf *myTriangleFinder) Name() resource.Name {
	return tf.name
}

func (tf *myTriangleFinder) GetProperties(ctx context.Context, extra map[string]interface{}) (*vision.Properties, error) {
	return &vision.Properties{
		DetectionSupported:      true,
		ClassificationSupported: false,
		ObjectPCDsSupported:     false,
	}, nil
}

// calculateIoU calculates the Intersection over Union between two rectangles
func calculateIoU(box1, box2 *image.Rectangle) float64 {
	if box1 == nil || box2 == nil {
		return 0
	}

	// Calculate intersection rectangle
	intersection := box1.Intersect(*box2)
	if intersection.Empty() {
		return 0
	}

	// Calculate areas
	intersectionArea := intersection.Dx() * intersection.Dy()
	box1Area := box1.Dx() * box1.Dy()
	box2Area := box2.Dx() * box2.Dy()

	// Calculate IoU
	unionArea := box1Area + box2Area - intersectionArea
	return float64(intersectionArea) / float64(unionArea)
}

func (tf *myTriangleFinder) findTriangle(imgMatrix [][]float32) []objdet.Detection {
	// Find matches using all templates
	var allMatches []Match
	for _, template := range tf.templates {
		matches := template.FindMatch(imgMatrix, 2, tf.config.Threshold) //TODO: stride configurable
		allMatches = append(allMatches, matches...)
	}

	// Convert matches to detections
	detections := make([]objdet.Detection, 0, len(allMatches))
	for _, match := range allMatches {
		box := match.GetBoundingBox()
		det := objdet.NewDetectionWithoutImgBounds(box, float64(match.Score), "triangle")
		detections = append(detections, det)
	}

	// Sort detections by score in descending order
	sort.Slice(detections, func(i, j int) bool {
		return detections[i].Score() > detections[j].Score()
	})

	// Apply Non-Maximum Suppression
	var filteredDetections []objdet.Detection
	used := make([]bool, len(detections))

	for i := 0; i < len(detections); i++ {
		if used[i] {
			continue
		}

		// Keep the current detection
		filteredDetections = append(filteredDetections, detections[i])
		used[i] = true

		// Check overlap with remaining detections
		for j := i + 1; j < len(detections); j++ {
			if used[j] {
				continue
			}

			// Calculate IoU between current and remaining detection
			iou := calculateIoU(detections[i].BoundingBox(), detections[j].BoundingBox())

			// If IoU is greater than threshold, mark as used
			if iou > 0.3 { // You can adjust this threshold
				used[j] = true
			}
		}
	}

	return filteredDetections
}

func (tf *myTriangleFinder) DetectionsFromCamera(
	ctx context.Context,
	cameraName string,
	extra map[string]interface{},
) ([]objdet.Detection, error) {
	mimeType := "image/jpeg"
	image, err := camera.DecodeImageFromCamera(ctx, mimeType, nil, tf.cam)
	if err != nil {
		return nil, errors.Errorf("failed to get and decode image for %s got: %s", ModelName, err)
	}

	imgMatrix := ImageToMatrix(image)
	return tf.findTriangle(imgMatrix), nil
}

func (tf *myTriangleFinder) Detections(ctx context.Context, img image.Image, extra map[string]interface{}) ([]objdet.Detection, error) {
	// Convert image to grayscale
	mat := ImageToMatrix(img)
	return tf.findTriangle(mat), nil
}

func (tf *myTriangleFinder) Classifications(ctx context.Context, img image.Image,
	n int, extra map[string]interface{},
) (classification.Classifications, error) {
	return nil, errUnimplemented
}

func (tf *myTriangleFinder) ClassificationsFromCamera(
	ctx context.Context,
	cameraName string,
	n int,
	extra map[string]interface{},
) (classification.Classifications, error) {
	return nil, errUnimplemented
}

func (tf *myTriangleFinder) GetObjectPointClouds(
	ctx context.Context,
	cameraName string,
	extra map[string]interface{},
) ([]*vis.Object, error) {
	return nil, errUnimplemented
}

func (tf *myTriangleFinder) CaptureAllFromCamera(
	ctx context.Context,
	cameraName string,
	opt viscapture.CaptureOptions,
	extra map[string]interface{},
) (viscapture.VisCapture, error) {
	res := viscapture.VisCapture{}
	mimeType := "image/jpeg"
	image, err := camera.DecodeImageFromCamera(ctx, mimeType, nil, tf.cam)
	if err != nil {
		return viscapture.VisCapture{}, errors.Errorf("failed to get image from camera for %s got: %s", ModelName, err)
	}
	if opt.ReturnImage {
		res.Image = image
	}
	if opt.ReturnDetections {
		dets, err := tf.Detections(ctx, image, extra)
		if err != nil {
			return viscapture.VisCapture{}, errors.Errorf("failed to get detections from camera for %s got: %s", ModelName, err)
		}
		res.Detections = dets
	}
	return res, nil
}

func (tf *myTriangleFinder) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, errUnimplemented
}

func (tf *myTriangleFinder) Close(ctx context.Context) error {
	return nil
}
