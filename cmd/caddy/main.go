package main

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/lightcodelabs/ingress/internal/controller"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const (
	// high enough QPS to fit all expected use cases.
	defaultQPS = 1e6

	// high enough Burst to fit all expected use cases.
	defaultBurst = 1e6
)

func main() {
	klog.InitFlags(nil)

	// parse any flags required to configure the caddy ingress controller
	cfg := parseFlags()

	if cfg.WatchNamespace == "" {
		cfg.WatchNamespace = v1.NamespaceAll
		klog.Warning("-namespace flag is unset, caddy ingress controller will monitor ingress resources in all namespaces.")
	}

	// get client to access the kubernetes service api
	kubeClient, err := createApiserverClient()
	if err != nil {
		msg := `
		Error while initiating a connection to the Kubernetes API server.
		This could mean the cluster is misconfigured (e.g. it has invalid
		API server certificates or Service Accounts configuration)
		`

		klog.Fatalf(msg, err)
	}

	restClient := kubeClient.ExtensionsV1beta1().RESTClient()

	// start ingress controller
	c := controller.NewCaddyController(kubeClient, restClient, cfg)

	// create http server to expose controller health metrics
	healthPort := 9090
	go startMetricsServer(healthPort)

	klog.Info("Starting the caddy ingress controller")

	// start the ingress controller
	stopCh := make(chan struct{}, 1)
	defer close(stopCh)

	go c.Run(stopCh)

	select {}
}

type healthChecker struct{}

func (h *healthChecker) Name() string {
	return "caddy-ingress-controller"
}

func (h *healthChecker) Check(_ *http.Request) error {
	return nil
}

func startMetricsServer(port int) {
	mux := http.NewServeMux()

	// reg := prometheus.NewRegistry()

	// reg.MustRegister(prometheus.NewGoCollector())
	// reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{
	// 	PidFn:        func() (int, error) { return os.Getpid(), nil },
	// 	ReportErrors: true,
	// }))

	// // healthz.InstallHandler(mux,
	// // 	healthz.PingHealthz,
	// // 	healthChecker,
	// // )

	server := &http.Server{
		Addr:              fmt.Sprintf(":%v", port),
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	klog.Fatal(server.ListenAndServe())
}

// createApiserverClient creates a new Kubernetes REST client. We assume the
// controller runs inside Kubernetes and use the in-cluster config.
func createApiserverClient() (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}

	cfg.QPS = defaultQPS
	cfg.Burst = defaultBurst
	cfg.ContentType = "application/vnd.kubernetes.protobuf"

	klog.Infof("Creating API client for %s", cfg.Host)

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	// The client may fail to connect to the API server on the first request
	defaultRetry := wait.Backoff{
		Steps:    10,
		Duration: 1 * time.Second,
		Factor:   1.5,
		Jitter:   0.1,
	}

	klog.V(2).Info("Attempting to discover Kubernetes version")

	var v *version.Info
	var retries int
	var lastErr error

	err = wait.ExponentialBackoff(defaultRetry, func() (bool, error) {
		v, err = client.Discovery().ServerVersion()
		if err == nil {
			return true, nil
		}

		lastErr = err
		klog.V(2).Infof("Unexpected error discovering Kubernetes version (attempt %v): %v", retries, err)
		retries++
		return false, nil
	})

	// err is returned in case of timeout in the exponential backoff (ErrWaitTimeout)
	if err != nil {
		return nil, lastErr
	}

	// this should not happen, warn the user
	if retries > 0 {
		klog.Warningf("Initial connection to the Kubernetes API server was retried %d times.", retries)
	}

	msg := "Running in Kubernetes cluster version v%v.%v (%v) - git (%v) commit %v - platform %v"
	klog.Infof(msg, v.Major, v.Minor, v.GitVersion, v.GitTreeState, v.GitCommit, v.Platform)

	return client, nil
}
