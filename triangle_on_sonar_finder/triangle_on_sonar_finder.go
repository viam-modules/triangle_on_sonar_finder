package triangle_on_sonar_finder

import (
	"context"

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
	"gocv.io/x/gocv"
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

	name           resource.Name
	logger         logging.Logger
	cam            camera.Camera
	config         *TriangleFinderConfig
	templateImages []gocv.Mat
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
	tf.templateImages, err = loadTemplates(newConf.PathToTemplatesDirectory)
	if err != nil {
		return nil, errors.Errorf("failed to load template images for %s got: %s", ModelName, err)
	}

	if len(tf.templateImages) == 0 {
		return nil, errors.Errorf("no valid template images found in %s", newConf.PathToTemplatesDirectory)
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

// getAndDecodeImage captures an image from the camera and decodes it into a gocv.Mat.
// It returns the decoded image as a grayscale gocv.Mat.
func (tf *myTriangleFinder) getAndDecodeImage(ctx context.Context) (gocv.Mat, error) {
	// Default mime type and extra parameters
	mimeType := "image/jpeg"
	imageBytes, _, err := tf.cam.Image(ctx, mimeType, nil)
	if err != nil {
		return gocv.Mat{}, errors.Errorf("failed to capture image from camera for %s got: %s", ModelName, err)
	}

	img, err := gocv.IMDecode(imageBytes, gocv.IMReadGrayScale)
	if err != nil {
		return gocv.Mat{}, errors.Errorf("failed to decode image for %s got: %s", ModelName, err)
	}

	if img.Empty() {
		return gocv.Mat{}, errors.New("decoded image is empty")
	}

	return img, nil
}

func (tf *myTriangleFinder) getGreyScaleMat(img image.Image) (gocv.Mat, error) {
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
	if err != nil {
		return gocv.Mat{}, errors.Errorf("failed to convert image to grayscale mat: %s", err)
	}
	if mat.Empty() {
		return gocv.Mat{}, errors.New("converted grayscale mat is empty")
	}
	return mat, nil
}

func (tf *myTriangleFinder) findTriangle(img gocv.Mat) []objdet.Detection {
	// Use template matching to find triangles in the image and return detections.
	boxes := FindMatchingPatterns(img, tf.templateImages, tf.config.Threshold)
	detections := make([]objdet.Detection, 0, len(boxes))
	for _, box := range boxes {
		det := objdet.NewDetectionWithoutImgBounds(box, 1.0, "triangle")
		detections = append(detections, det)
	}
	return detections
}

func (tf *myTriangleFinder) DetectionsFromCamera(
	ctx context.Context,
	cameraName string,
	extra map[string]interface{},
) ([]objdet.Detection, error) {
	img, err := tf.getAndDecodeImage(ctx)
	if err != nil {
		return nil, errors.Errorf("failed to get and decode image for %s got: %s", ModelName, err)
	}
	return tf.findTriangle(img), nil
}

func (tf *myTriangleFinder) Detections(ctx context.Context, img image.Image, extra map[string]interface{}) ([]objdet.Detection, error) {
	mat, err := tf.getGreyScaleMat(img)
	if err != nil {
		return nil, errors.Errorf("failed to get grey scale mat for %s got: %s", ModelName, err)
	}

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
	tf.logger.Errorw("REACHING HERE")
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
		dets, err := tf.DetectionsFromCamera(ctx, cameraName, extra)
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
