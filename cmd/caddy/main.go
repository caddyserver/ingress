package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/caddyserver/ingress/internal/controller"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// high enough QPS to fit all expected use cases.
	defaultQPS = 1e6

	// high enough Burst to fit all expected use cases.
	defaultBurst = 1e6
)

func main() {
	// parse any flags required to configure the caddy ingress controller
	cfg := parseFlags()

	if cfg.WatchNamespace == "" {
		cfg.WatchNamespace = v1.NamespaceAll
		logrus.Warning("-namespace flag is unset, caddy ingress controller will monitor ingress resources in all namespaces.")
	}

	// get client to access the kubernetes service api
	kubeClient, err := createApiserverClient()
	if err != nil {
		msg := "Could not establish a connection to the Kubernetes API Server."
		logrus.Fatalf(msg, err)
	}

	restClient := kubeClient.ExtensionsV1beta1().RESTClient()
	c := controller.NewCaddyController(kubeClient, restClient, cfg)

	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{
		PidFn:        func() (int, error) { return os.Getpid(), nil },
		ReportErrors: true,
	}))

	// create http server to expose controller health metrics
	go startMetricsServer(reg)

	// start the ingress controller
	stopCh := make(chan struct{}, 1)
	defer close(stopCh)

	logrus.Info("Starting the caddy ingress controller")
	go c.Run(stopCh)

	// TODO :- listen to sigterm
	select {}
}

func startMetricsServer(reg *prometheus.Registry) {
	mux := http.NewServeMux()
	mux.Handle(
		"/metrics",
		promhttp.InstrumentMetricHandler(
			reg,
			promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
		),
	)

	logrus.Info("Exporting metrics on :9090")
	server := &http.Server{
		Addr:              fmt.Sprintf(":%v", 9090),
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	logrus.Fatal(server.ListenAndServe())
}

// createApiserverClient creates a new Kubernetes REST client. We assume the
// controller runs inside Kubernetes and use the in-cluster config.
func createApiserverClient() (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}

	logrus.Infof("Creating API client for %s", cfg.Host)

	cfg.QPS = defaultQPS
	cfg.Burst = defaultBurst
	cfg.ContentType = "application/vnd.kubernetes.protobuf"
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

	var v *version.Info
	var retries int
	var lastErr error

	err = wait.ExponentialBackoff(defaultRetry, func() (bool, error) {
		v, err = client.Discovery().ServerVersion()
		if err == nil {
			return true, nil
		}

		lastErr = err
		logrus.Infof("Unexpected error discovering Kubernetes version (attempt %v): %v", retries, err)
		retries++
		return false, nil
	})

	// err is returned in case of timeout in the exponential backoff (ErrWaitTimeout)
	if err != nil {
		return nil, lastErr
	}

	if retries > 0 {
		logrus.Warningf("Initial connection to the Kubernetes API server was retried %d times.", retries)
	}

	return client, nil
}
