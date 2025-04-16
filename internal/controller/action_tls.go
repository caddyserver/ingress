package controller

import (
	"os"
	"path/filepath"

	apiv1 "k8s.io/api/core/v1"
)

var certFolder = ""

// GetCertFolder returns the staging path for storing certificates as files.
func GetCertFolder() string {
	if certFolder == "" {
		// Use the systemd cache directory if possible.
		runtimeDir := os.Getenv("RUNTIME_DIRECTORY")
		if runtimeDir != "" {
			certFolder = filepath.Join(runtimeDir, "certs")
		} else {
			certFolder = filepath.FromSlash("/etc/caddy/certs")
		}
	}
	return certFolder
}

// onSecretAdded runs when a TLS secret resource is added to the cluster.
func (c *CaddyController) onSecretAdded(obj *apiv1.Secret) error {
	c.logger.Infof("TLS secret created (%s/%s)", obj.Namespace, obj.Name)
	return writeFile(obj)
}

// onSecretUpdated is run when a TLS secret resource is updated in the cluster.
func (c *CaddyController) onSecretUpdated(obj *apiv1.Secret) error {
	c.logger.Infof("TLS secret updated (%s/%s)", obj.Namespace, obj.Name)
	return writeFile(obj)
}

// onSecretDeleted is run when a TLS secret resource is deleted from the cluster.
func (c *CaddyController) onSecretDeleted(obj *apiv1.Secret) error {
	c.logger.Infof("TLS secret deleted (%s/%s)", obj.Namespace, obj.Name)
	return os.Remove(filepath.Join(GetCertFolder(), obj.Name+".pem"))
}

// writeFile writes a secret to a .pem file on disk.
func writeFile(s *apiv1.Secret) error {
	content := make([]byte, 0)

	for _, cert := range s.Data {
		content = append(content, cert...)
	}

	err := os.WriteFile(filepath.Join(GetCertFolder(), s.Name+".pem"), content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (c *CaddyController) isManagedTLSSecret(secret *apiv1.Secret) bool {
	for _, ing := range c.resourceStore.Ingresses() {
		for _, tlsRule := range ing.Spec.TLS {
			if tlsRule.SecretName == secret.Name && ing.Namespace == secret.Namespace {
				return true
			}
		}
	}
	return false
}

// watchTLSSecrets starts listening to TLS secrets and syncs CertFolder.
func (c *CaddyController) watchTLSSecrets() error {
	if err := os.MkdirAll(GetCertFolder(), 0755); err != nil && !os.IsExist(err) {
		return err
	}

	// Init informers
	c.informers.Secret = c.factories.WatchedNamespace.Core().V1().Secrets().Informer()
	c.informers.Secret.AddEventHandler(&QueuedEventHandlers[apiv1.Secret]{
		Queue:      c.syncQueue,
		FilterFunc: c.isManagedTLSSecret,
		AddFunc:    c.onSecretAdded,
		UpdateFunc: c.onSecretUpdated,
		DeleteFunc: c.onSecretDeleted,
	})

	// Run it
	go c.informers.Secret.Run(c.stopChan)
	c.factories.WatchedNamespace.WaitForCacheSync(c.stopChan)

	// Sync secrets
	for _, secret := range c.informers.Secret.GetStore().List() {
		if err := writeFile(secret.(*apiv1.Secret)); err != nil {
			return err
		}
	}

	return nil
}
