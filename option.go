package saiyan

import (
	"github.com/sohaha/zlsgo/zfile"
)

type Option func(conf *Config)

// WithCommand Custom start command
func WithCommand(command string) Option {
	return func(c *Config) {
		c.Command = command
	}
}

// WithProjectPath Custom project dir
func WithProjectPath(path string) Option {
	return func(c *Config) {
		c.ProjectPath = zfile.RealPath(path)
	}
}

// WithWorkerSum Custom WorkerSum
func WithWorkerSum(i uint64) Option {
	return func(c *Config) {
		c.WorkerSum = i
	}
}

// WithMaxWorkerSum Custom MaxWorkerSum
func WithMaxWorkerSum(i uint64) Option {
	return func(c *Config) {
		c.MaxWorkerSum = i
	}
}

// ManualAllOption Manually configure all configurations
func ManualAllOption(o Option) Option {
	return func(c *Config) {
		o(c)
	}
}
