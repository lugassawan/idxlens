package cli

// SilentError is an error that should cause the process to exit
// with the given exit code without printing any message.
type SilentError struct {
	ExitCode int
}

func (e *SilentError) Error() string {
	return ""
}
