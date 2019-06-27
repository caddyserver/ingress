package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var certDir = filepath.FromSlash("/etc/caddy/certs")

// CertManager manager user defined certs on ingress resources for caddy.
type CertManager struct {
	certInformer cache.Controller
	certs        []certificate
	syncQueue    workqueue.RateLimitingInterface
	synced       bool
}

type certificate struct {
	name      string
	namespace string
}

// HandleOwnCertManagement handles whether we need to watch for user defined
// certs and update caddy.
func (c *CaddyController) HandleOwnCertManagement(ings []*v1beta1.Ingress) (map[string]interface{}, error) {
	var certs []certificate
	var hosts []string

	// do we have any ingresses with TLS certificates and secrets defined on them?
	for _, ing := range ings {
		for _, tlsRule := range ing.Spec.TLS {
			for _, h := range tlsRule.Hosts {
				hosts = append(hosts, h)
			}

			c := certificate{name: tlsRule.SecretName, namespace: ing.Namespace}
			certs = append(certs, c)
		}
	}

	// run the caddy cert sync now (ONE TIME) but only run it in the future
	// when a cert has been updated (or a new cert has been added)
	if len(certs) > 0 && c.certManager == nil {
		err := syncCertificates(certs, c.kubeClient)
		if err != nil {
			return nil, err
		}

		informer, err := newSecretInformer(c)
		if err != nil {
			return nil, err
		}

		c.certManager = &CertManager{
			certs:        certs,
			certInformer: informer,
			syncQueue:    c.syncQueue,
		}

		// start the informer to listen to secrets
		go informer.Run(c.stopChan)
	}

	fmt.Printf("\nCERTS: %+v - %+v\n", len(certs), certs)

	if len(certs) > 0 {
		return getTLSConfig(hosts), nil
	}

	return nil, nil
}

// newSecretInformer creates an informer to listen to updates to secrets.
func newSecretInformer(c *CaddyController) (cache.Controller, error) {
	secretInformer := sv1.NewSecretInformer(c.kubeClient, c.config.WatchNamespace, secretSyncInterval, cache.Indexers{})
	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onSecretResourceAdded,
		UpdateFunc: c.onSecretResourceUpdated,
		DeleteFunc: c.onSecretResourceDeleted,
	})

	return secretInformer, nil
}

// getTLSConfig returns the caddy config for certificate management to load all certs from certDir.
func getTLSConfig(hosts []string) map[string]interface{} {
	return map[string]interface{}{
		"load_folders": json.RawMessage(`["` + certDir + `"]`),
		"hosts":        hosts,
	}
}

// syncCertificates downloads the certificate files defined on a ingress resource and
// stores it locally in this pod for use by caddy.
func syncCertificates(certs []certificate, kubeClient *kubernetes.Clientset) error {
	logrus.Info("Found TLS certificates on ingress resource. Syncing...")

	certData := make(map[string]map[string][]byte, len(certs))
	for _, cert := range certs {
		s, err := kubeClient.CoreV1().Secrets(cert.namespace).Get(cert.name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		certData[cert.name] = s.Data
	}

	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		err = os.MkdirAll(certDir, 0755)
		if err != nil {
			return err
		}
	}

	// combine crt and key and combine to .pem in cert directory
	for secret, data := range certData {
		content := make([]byte, 0)

		for _, cert := range data {
			content = append(content, cert...)
		}

		err := ioutil.WriteFile(filepath.Join(certDir, secret+".pem"), content, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// SecretResourceAddedAction provides an implementation of the action interface.
type SecretResourceAddedAction struct {
	resource *apiv1.Secret
}

// SecretResourceUpdatedAction provides an implementation of the action interface.
type SecretResourceUpdatedAction struct {
	resource    *apiv1.Secret
	oldResource *apiv1.Secret
}

// SecretResourceDeletedAction provides an implementation of the action interface.
type SecretResourceDeletedAction struct {
	resource *apiv1.Secret
}

// onSecretResourceAdded runs when a secret resource is added to the cluster.
func (c *CaddyController) onSecretResourceAdded(obj interface{}) {
	s, ok := obj.(*apiv1.Secret)
	if ok {
		for _, secret := range c.certManager.certs {
			if s.Name == secret.name {
				c.syncQueue.Add(SecretResourceAddedAction{
					resource: s,
				})
			}
		}
	}
}

// writeFile writes a secret to a .pem file on disk.
func writeFile(s *apiv1.Secret) error {
	content := make([]byte, 0)

	for _, cert := range s.Data {
		content = append(content, cert...)
	}

	err := ioutil.WriteFile(filepath.Join(certDir, s.Name+".pem"), content, 0644)
	if err != nil {
		return err
	}

	return nil
}

// onSecretResourceUpdated is run when a secret resource is updated in the cluster.
func (c *CaddyController) onSecretResourceUpdated(old interface{}, new interface{}) {
	s, ok := old.(*apiv1.Secret)
	if !ok {
		return
	}

	snew, ok := new.(*apiv1.Secret)
	for _, secret := range c.certManager.certs {
		if s.Name == secret.name {
			c.syncQueue.Add(SecretResourceUpdatedAction{
				resource:    snew,
				oldResource: s,
			})
		}
	}
}

// onSecretResourceDeleted is run when a secret resource is deleted from the cluster.
func (c *CaddyController) onSecretResourceDeleted(obj interface{}) {
	s, ok := obj.(*apiv1.Secret)
	if ok {
		for _, secret := range c.certManager.certs {
			if s.Name == secret.name {
				c.syncQueue.Add(SecretResourceDeletedAction{
					resource: s,
				})
			}
		}
	}
}

// handle is run when a SecretResourceDeletedAction appears in the queue.
func (r SecretResourceDeletedAction) handle(c *CaddyController) error {
	return os.Remove(filepath.Join(certDir, r.resource.Name+".pem"))
}

// handle is run when a SecretResourceUpdatedAction appears in the queue.
func (r SecretResourceUpdatedAction) handle(c *CaddyController) error {
	return writeFile(r.resource)
}

// handle is run when a SecretResourceAddedAction appears in the queue.
func (r SecretResourceAddedAction) handle(c *CaddyController) error {
	return writeFile(r.resource)
}
