package controller

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/caddyserver/ingress/internal/k8s"
	apiv1 "k8s.io/api/core/v1"
)

var CertFolder = filepath.FromSlash("/etc/caddy/certs")

// SecretAddedAction provides an implementation of the action interface.
type SecretAddedAction struct {
	resource *apiv1.Secret
}

// SecretUpdatedAction provides an implementation of the action interface.
type SecretUpdatedAction struct {
	resource    *apiv1.Secret
	oldResource *apiv1.Secret
}

// SecretDeletedAction provides an implementation of the action interface.
type SecretDeletedAction struct {
	resource *apiv1.Secret
}

// onSecretAdded runs when a TLS secret resource is added to the cluster.
func (c *CaddyController) onSecretAdded(obj *apiv1.Secret) {
	if k8s.IsManagedTLSSecret(obj, c.resourceStore.Ingresses) {
		c.syncQueue.Add(SecretAddedAction{
			resource: obj,
		})
	}
}

// onSecretUpdated is run when a TLS secret resource is updated in the cluster.
func (c *CaddyController) onSecretUpdated(old *apiv1.Secret, new *apiv1.Secret) {
	if k8s.IsManagedTLSSecret(new, c.resourceStore.Ingresses) {
		c.syncQueue.Add(SecretUpdatedAction{
			resource:    new,
			oldResource: old,
		})
	}
}

// onSecretDeleted is run when a TLS secret resource is deleted from the cluster.
func (c *CaddyController) onSecretDeleted(obj *apiv1.Secret) {
	c.syncQueue.Add(SecretDeletedAction{
		resource: obj,
	})
}

// writeFile writes a secret to a .pem file on disk.
func writeFile(s *apiv1.Secret) error {
	content := make([]byte, 0)

	for _, cert := range s.Data {
		content = append(content, cert...)
	}

	err := ioutil.WriteFile(filepath.Join(CertFolder, s.Name+".pem"), content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r SecretAddedAction) handle(c *CaddyController) error {
	c.logger.Infof("TLS secret created (%s/%s)", r.resource.Namespace, r.resource.Name)
	return writeFile(r.resource)
}

func (r SecretUpdatedAction) handle(c *CaddyController) error {
	c.logger.Infof("TLS secret updated (%s/%s)", r.resource.Namespace, r.resource.Name)
	return writeFile(r.resource)
}

func (r SecretDeletedAction) handle(c *CaddyController) error {
	c.logger.Infof("TLS secret deleted (%s/%s)", r.resource.Namespace, r.resource.Name)
	return os.Remove(filepath.Join(CertFolder, r.resource.Name+".pem"))
}

// watchTLSSecrets Start listening to TLS secrets if at least one ingress needs it.
// It will sync the CertFolder with TLS secrets
func (c *CaddyController) watchTLSSecrets() error {
	if c.informers.TLSSecret == nil && c.resourceStore.HasManagedTLS() {
		// Init informers
		params := k8s.TLSSecretParams{
			InformerFactory: c.factories.WatchedNamespace,
		}
		c.informers.TLSSecret = k8s.WatchTLSSecrets(params, k8s.TLSSecretHandlers{
			AddFunc:    c.onSecretAdded,
			UpdateFunc: c.onSecretUpdated,
			DeleteFunc: c.onSecretDeleted,
		})

		// Run it
		go c.informers.TLSSecret.Run(c.stopChan)

		// Sync secrets
		secrets, err := k8s.ListTLSSecrets(params, c.resourceStore.Ingresses)
		if err != nil {
			return err
		}

		if _, err := os.Stat(CertFolder); os.IsNotExist(err) {
			err = os.MkdirAll(CertFolder, 0755)
			if err != nil {
				return err
			}
		}

		for _, secret := range secrets {
			if err := writeFile(secret); err != nil {
				return err
			}
		}
	}

	return nil
}
