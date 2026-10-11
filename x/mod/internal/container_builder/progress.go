package container_builder

import (
	"strings"

	"github.com/moby/buildkit/client"
	"go.uber.org/zap"
)

// progress transforme le statusCh de statuts de BuildKit en lines de log.
// Graph vertices and command output end up in the same place as the rest of
// the pipeline, hence in the same traces.
type progress struct {
	logger *zap.Logger
}

func (p progress) consume(statusCh <-chan *client.SolveStatus) {
	for status := range statusCh {
		p.reportSteps(status)
		p.reportOutput(status)
	}
}

// reportSteps logs the progress of the graph vertices.
func (p progress) reportSteps(status *client.SolveStatus) {
	for _, vertex := range status.Vertexes {
		switch {
		case vertex.Error != "":
			p.logger.Error("vertex failed",
				zap.String("vertex", vertex.Name),
				zap.String("error", vertex.Error))
		case vertex.Completed != nil:
			p.logger.Debug("vertex completed", zap.String("vertex", vertex.Name))
		case vertex.Started != nil:
			p.logger.Info("vertex", zap.String("name", vertex.Name))
		}
	}
}

// reportOutput logs what the commands write on stdout and stderr.
func (p progress) reportOutput(status *client.SolveStatus) {
	for _, entry := range status.Logs {
		text := strings.TrimRight(string(entry.Data), "\n")
		if text != "" {
			p.logger.Info(text)
		}
	}
}
