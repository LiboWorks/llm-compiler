package pluginapi

// PluginAPI defines extension points that a private/pro module can
// implement and register at runtime. Keep this package small and
// free of any dependencies on private code so it remains fully OSS.

// ProFeatures is the interface that the Pro repo implements to provide
// additional (commercial) functionality. Keep methods minimal and
// focused on behavior rather than types to avoid tight coupling.
type ProFeatures interface {
	// Example: EnhanceText receives input and returns enhanced text.
	EnhanceText(input string) (string, error)

	// Add other pro-only extension points here.
}

var (
	// registered holds the ProFeatures implementation when a Pro
	// module is included in the final binary and calls Register.
	registered ProFeatures
)

// Register is called by the pro module to wire in its implementation.
// The public repo never imports the pro module; the pro module imports
// this package and calls Register in its init() or explicit setup.
func Register(p ProFeatures) {
	registered = p
}

// Available reports whether a Pro implementation was registered.
func Available() bool {
	return registered != nil
}

// EnhanceTextIfAvailable runs the pro enhancement if available,
// otherwise returns the original text unmodified.
func EnhanceTextIfAvailable(input string) (string, error) {
	if registered == nil {
		return input, nil
	}
	return registered.EnhanceText(input)
}
