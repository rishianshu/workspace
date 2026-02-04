// Package workflow provides Temporal client and workflow definitions
package workflow

import (
	"fmt"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// TemporalClient wraps the Temporal SDK client
type TemporalClient struct {
	client.Client
	logger *zap.SugaredLogger
}

// NewTemporalClient creates a new Temporal client
func NewTemporalClient(hostPort string, logger *zap.SugaredLogger) (*TemporalClient, error) {
	// Create Temporal client options
	opts := client.Options{
		HostPort: hostPort,
		Logger:   newZapAdapter(logger),
	}

	// Connect to Temporal
	c, err := client.Dial(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	logger.Infow("Connected to Temporal", "host", hostPort)

	return &TemporalClient{
		Client: c,
		logger: logger,
	}, nil
}

// Close closes the Temporal client connection
func (c *TemporalClient) Close() {
	if c.Client != nil {
		c.Client.Close()
	}
}

// zapAdapter adapts zap logger to temporal logger interface
type zapAdapter struct {
	logger *zap.SugaredLogger
}

func newZapAdapter(logger *zap.SugaredLogger) *zapAdapter {
	return &zapAdapter{logger: logger}
}

func (z *zapAdapter) Debug(msg string, keyvals ...interface{}) {
	z.logger.Debugw(msg, keyvals...)
}

func (z *zapAdapter) Info(msg string, keyvals ...interface{}) {
	z.logger.Infow(msg, keyvals...)
}

func (z *zapAdapter) Warn(msg string, keyvals ...interface{}) {
	z.logger.Warnw(msg, keyvals...)
}

func (z *zapAdapter) Error(msg string, keyvals ...interface{}) {
	z.logger.Errorw(msg, keyvals...)
}
