// Package main is a module which implements the hough transform
package main

import (
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"

	"github.com/viam-modules/triangle_on_sonar_finder/triangle_on_sonar_finder"
)

func main() {
	module.ModularMain(
		resource.APIModel{API: vision.API, Model: triangle_on_sonar_finder.Model},
	)
}
