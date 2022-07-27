package steps

import (
	"context"
	pipeline "github.com/ccremer/go-command-pipeline"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// DebugLogger returns a list with a single hook that logs the step name.
// The logger is retrieved from the given context.
func DebugLogger(ctx context.Context) []pipeline.Listener {
	log := controllerruntime.LoggerFrom(ctx)
	hook := func(step pipeline.Step) {
		log.V(2).Info(`Entering step "` + step.Name + `"`)
	}
	return []pipeline.Listener{hook}
}
