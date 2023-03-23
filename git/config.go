package git

// ConfigOptions are options for Config.
type ConfigOptions struct {
	File string
	All  bool
	Add  bool
	CommandOptions
}

// Config gets a git configuration.
func Config(key string, opts ...ConfigOptions) (string, error) {
	var opt ConfigOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	cmd := NewCommand("config")
	if opt.File != "" {
		cmd.AddArgs("--file", opt.File)
	}
	if opt.All {
		cmd.AddArgs("--get-all")
	}
	for _, a := range opt.Args {
		cmd.AddArgs(a)
	}
	cmd.AddArgs(key)
	bts, err := cmd.Run()
	if err != nil {
		return "", err
	}
	return string(bts), nil
}

// SetConfig sets a git configuration.
func SetConfig(key string, value string, opts ...ConfigOptions) error {
	var opt ConfigOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	cmd := NewCommand("config")
	if opt.File != "" {
		cmd.AddArgs("--file", opt.File)
	}
	for _, a := range opt.Args {
		cmd.AddArgs(a)
	}
	cmd.AddArgs(key, value)
	_, err := cmd.Run()
	return err
}
