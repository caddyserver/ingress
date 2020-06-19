package controller

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/internal/caddy"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/networking/v1beta1"
)

// loadConfigMap runs when a config map with caddy config is loaded on app start.
func (c *CaddyController) onLoadConfig(obj io.Reader) {
	c.syncQueue.Add(LoadConfigAction{
		config: obj,
	})
}

// onResourceAdded runs when an ingress resource is added to the cluster.
func (c *CaddyController) onResourceAdded(obj interface{}) {
	c.syncQueue.Add(ResourceAddedAction{
		resource: obj,
	})
}

// onResourceUpdated is run when an ingress resource is updated in the cluster.
func (c *CaddyController) onResourceUpdated(old interface{}, new interface{}) {
	c.syncQueue.Add(ResourceUpdatedAction{
		resource:    new,
		oldResource: old,
	})
}

// onResourceDeleted is run when an ingress resource is deleted from the cluster.
func (c *CaddyController) onResourceDeleted(obj interface{}) {
	c.syncQueue.Add(ResourceDeletedAction{
		resource: obj,
	})
}

// onSyncStatus is run every sync interval to update the source address on ingresses.
func (c *CaddyController) onSyncStatus(obj interface{}) {
	c.syncQueue.Add(SyncStatusAction{})
}

// Action is an interface for ingress actions.
type Action interface {
	handle(c *CaddyController) error
}

// LoadConfigAction provides an implementation of the action interface.
type LoadConfigAction struct {
	config io.Reader
}

// ResourceAddedAction provides an implementation of the action interface.
type ResourceAddedAction struct {
	resource interface{}
}

// ResourceUpdatedAction provides an implementation of the action interface.
type ResourceUpdatedAction struct {
	resource    interface{}
	oldResource interface{}
}

// ResourceDeletedAction provides an implementation of the action interface.
type ResourceDeletedAction struct {
	resource interface{}
}

func (r LoadConfigAction) handle(c *CaddyController) error {
	logrus.Info("Config file detected, updating Caddy config...")
	return c.loadConfigFromFile(r.config)
}

func (r ResourceAddedAction) handle(c *CaddyController) error {
	logrus.Info("New ingress resource detected, updating Caddy config...")

	// configure caddy to handle this resource
	ing, ok := r.resource.(*v1beta1.Ingress)
	if !ok {
		return fmt.Errorf("ResourceAddedAction: incoming resource is not of type ingress")
	}

	// add this ingress to the internal store
	c.resourceStore.AddIngress(ing)

	err := updateConfig(c)
	if err != nil {
		return err
	}

	// ensure that ingress source is updated to point to this ingress controller's ip
	err = c.syncStatus([]*v1beta1.Ingress{ing})
	if err != nil {
		return errors.Wrapf(err, "syncing ingress source address name: %v", ing.GetName())
	}

	logrus.Info("Caddy reloaded successfully.")
	return nil
}

func (r ResourceUpdatedAction) handle(c *CaddyController) error {
	logrus.Info("Ingress resource update detected, updating Caddy config...")

	// update caddy config regarding this ingress
	ing, ok := r.resource.(*v1beta1.Ingress)
	if !ok {
		return fmt.Errorf("ResourceAddedAction: incoming resource is not of type ingress")
	}

	// add or update this ingress in the internal store
	c.resourceStore.AddIngress(ing)

	err := updateConfig(c)
	if err != nil {
		return err
	}

	logrus.Info("Caddy reloaded successfully.")
	return nil
}

func (r ResourceDeletedAction) handle(c *CaddyController) error {
	logrus.Info("Ingress resource deletion detected, updating Caddy config...")

	// delete all resources from caddy config that are associated with this resource
	// reload caddy config
	ing, ok := r.resource.(*v1beta1.Ingress)
	if !ok {
		return fmt.Errorf("ResourceAddedAction: incoming resource is not of type ingress")
	}

	// add this ingress to the internal store
	c.resourceStore.PluckIngress(ing)

	err := updateConfig(c)
	if err != nil {
		return err
	}

	logrus.Info("Caddy reloaded successfully.")
	return nil
}

// updateConfig updates internal caddy config with new ingress info.
func updateConfig(c *CaddyController) error {
	apps := c.resourceStore.CaddyConfig.Apps

	// if certs are defined on an ingress resource we need to handle them.
	tlsCfg, err := c.HandleOwnCertManagement(c.resourceStore.Ingresses)
	if err != nil {
		return errors.Wrap(err, "caddy config reload")
	}

	// after TLS secrets are synched we should load them in the cert pool.
	if tlsCfg != nil {
		apps["tls"].(caddytls.TLS).CertificatesRaw["load_folders"] = tlsCfg["load_folders"].(json.RawMessage)
	} else {
		// reset cert loading
		apps["tls"].(caddytls.TLS).CertificatesRaw["load_folders"] = json.RawMessage(`[]`)
	}

	// skip auto https for hosts with certs provided
	if tlsCfg != nil {
		if hosts, ok := tlsCfg["hosts"].([]string); ok {
			apps["http"].(caddyhttp.App).Servers["ingress_server"].AutoHTTPS.Skip = hosts
		}
	} else {
		// reset any skipped hosts set
		apps["http"].(caddyhttp.App).Servers["ingress_server"].AutoHTTPS.Skip = make([]string, 0)
	}

	if !c.usingConfigMap {
		serverRoutes, err := caddy.ConvertToCaddyConfig(c.resourceStore.Ingresses)
		if err != nil {
			return errors.Wrap(err, "converting ingress resources to caddy config")
		}

		// set the http server routes
		apps["http"].(caddyhttp.App).Servers["ingress_server"].Routes = serverRoutes
	}

	// reload caddy with new config
	err = c.reloadCaddy()
	if err != nil {
		return errors.Wrap(err, "caddy config reload")
	}

	return nil
}
