package saiyan

type Option func(conf *Config)

// WithCommand Custom start command
func WithCommand(command string) Option {
	return func(c *Config) {
		c.Command = command
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
