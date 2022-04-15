package store

// Options represents ingress controller config received through cli arguments.
type Options struct {
	WatchNamespace string
	ConfigMapName  string
	Verbose        bool
	LeaseId        string
	PluginsOrder   []string
}
