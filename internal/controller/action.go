package controller

import (
	"fmt"
	"io"

	"github.com/caddyserver/caddy2/modules/caddyhttp"
	"github.com/caddyserver/ingress/internal/caddy"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/extensions/v1beta1"
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

func updateConfig(c *CaddyController) error {
	// update internal caddy config with new ingress info
	// serverRoutes, hosts, err := caddy.ConvertToCaddyConfig(c.resourceStore.Ingresses)
	serverRoutes, _, err := caddy.ConvertToCaddyConfig(c.resourceStore.Ingresses)
	if err != nil {
		return errors.Wrap(err, "converting ingress resources to caddy config")
	}

	// set the http server routes
	if c.resourceStore.CaddyConfig != nil {
		c.resourceStore.CaddyConfig.Apps["http"].(caddyhttp.App).Servers["ingress_server"].Routes = serverRoutes
	}

	// reload caddy2 config with newConfig
	err = c.reloadCaddy()
	if err != nil {
		return errors.Wrap(err, "caddy config reload")
	}

	return nil
}
