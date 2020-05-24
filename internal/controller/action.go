package controller

import (
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/caddyserver/ingress/internal/caddy"
	config "github.com/caddyserver/ingress/internal/caddy"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/networking/v1beta1"
)

// loadConfigMap runs when a config map with caddy config is loaded on app start.
func (c *CaddyController) onLoadConfig(obj interface{}) {
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
	config interface{}
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

	c.resourceStore.CaddyConfig = r.config.(*config.Config)

	err := regenerateConfig(c)
	if err != nil {
		return err
	}

	return nil
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

	err := regenerateConfig(c)
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

	err := regenerateConfig(c)
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

	err := regenerateConfig(c)
	if err != nil {
		return err
	}

	logrus.Info("Caddy reloaded successfully.")
	return nil
}

// regenerateConfig regenerate caddy config with updated resources.
func regenerateConfig(c *CaddyController) error {
	logrus.Info("Updating caddy config")

	var cfg *config.Config
	var cfgFile *config.Config = nil
	var err error

	if c.usingConfigMap {
		cfgFile, err = loadCaddyConfigFile("/etc/caddy/config.json")
		if err != nil {
			logrus.Warn("Unable to load config file: %v", err)
		}
	}

	cfg = config.NewConfig(c.podInfo.Namespace, cfgFile)

	tlsApp := cfg.Apps["tls"].(*caddytls.TLS)
	httpApp := cfg.Apps["http"].(*caddyhttp.App)

	if c.resourceStore.ConfigMap != nil {
		err := setConfigMapOptions(c, cfg)
		if err != nil {
			return errors.Wrap(err, "caddy config reload")
		}
	}

	// if certs are defined on an ingress resource we need to handle them.
	tlsCfg, err := c.HandleOwnCertManagement(c.resourceStore.Ingresses)
	if err != nil {
		return errors.Wrap(err, "caddy config reload")
	}

	// after TLS secrets are synched we should load them in the cert pool
	// and skip auto https for hosts with certs provided
	if tlsCfg != nil {
		tlsApp.CertificatesRaw["load_folders"] = tlsCfg["load_folders"].(json.RawMessage)

		if hosts, ok := tlsCfg["hosts"].([]string); ok {
			httpApp.Servers["ingress_server"].AutoHTTPS.Skip = hosts
		}
	}

	if !c.usingConfigMap {
		serverRoutes, err := caddy.ConvertToCaddyConfig(c.resourceStore.Ingresses)
		if err != nil {
			return errors.Wrap(err, "converting ingress resources to caddy config")
		}

		// set the http server routes
		httpApp.Servers["ingress_server"].Routes = serverRoutes
	}

	// reload caddy with new config
	err = c.reloadCaddy(cfg)
	if err != nil {
		return errors.Wrap(err, "caddy config reload")
	}

	return nil
}
