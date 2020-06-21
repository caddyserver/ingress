package controller

import (
	"fmt"
	"github.com/caddyserver/ingress/internal/caddy"

	caddy2 "github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type ConfigMapOptions struct {
	Debug  bool   `json:"debug"`
	AcmeCA string `json:"acme-ca"`
	Email  string `json:"email"`
}

// onConfigMapAdded is run when a config map is added to the cluster.
func (c *CaddyController) onConfigMapAdded(obj interface{}) {
	c.syncQueue.Add(ConfigMapAddedAction{
		resource: obj,
	})
}

// onConfigMapUpdated is run when an ingress resource is updated in the cluster.
func (c *CaddyController) onConfigMapUpdated(old interface{}, new interface{}) {
	c.syncQueue.Add(ConfigMapUpdatedAction{
		resource:    new,
		oldResource: old,
	})
}

// onConfigMapDeleted is run when an ingress resource is deleted from the cluster.
func (c *CaddyController) onConfigMapDeleted(obj interface{}) {
	c.syncQueue.Add(ConfigMapDeletedAction{
		resource: obj,
	})
}

// ConfigMapAddedAction provides an implementation of the action interface.
type ConfigMapAddedAction struct {
	resource interface{}
}

// ConfigMapUpdatedAction provides an implementation of the action interface.
type ConfigMapUpdatedAction struct {
	resource    interface{}
	oldResource interface{}
}

// ConfigMapDeletedAction provides an implementation of the action interface.
type ConfigMapDeletedAction struct {
	resource interface{}
}

func (r ConfigMapAddedAction) handle(c *CaddyController) error {
	cfgMap, ok := r.resource.(*v1.ConfigMap)
	if !ok {
		return fmt.Errorf("ConfigMapAddedAction: incoming resource is not of type configmap")
	}

	// only care about the caddy config map
	if !changeTriggerUpdate(c, cfgMap) {
		return nil
	}

	logrus.Info("New configmap detected, updating Caddy config...")
	// save to the store the current config map to use
	c.resourceStore.ConfigMap = cfgMap

	err := regenerateConfig(c)
	if err != nil {
		return err
	}

	logrus.Info("Caddy reloaded successfully.")
	return nil
}

func (r ConfigMapUpdatedAction) handle(c *CaddyController) error {
	cfgMap, ok := r.resource.(*v1.ConfigMap)
	if !ok {
		return fmt.Errorf("ConfigMapUpdatedAction: incoming resource is not of type configmap")
	}

	// only care about the caddy config map
	if !changeTriggerUpdate(c, cfgMap) {
		return nil
	}

	logrus.Info("ConfigMap resource updated, updating Caddy config...")

	// save to the store the current config map to use
	c.resourceStore.ConfigMap = cfgMap

	err := regenerateConfig(c)
	if err != nil {
		return err
	}

	logrus.Info("Caddy reloaded successfully.")
	return nil
}

func (r ConfigMapDeletedAction) handle(c *CaddyController) error {
	cfgMap, ok := r.resource.(*v1.ConfigMap)
	if !ok {
		return fmt.Errorf("ConfigMapDeletedAction: incoming resource is not of type configmap")
	}

	// only care about the caddy config map
	if !changeTriggerUpdate(c, cfgMap) {
		return nil
	}

	logrus.Info("ConfigMap resource deleted, updating Caddy config...")

	// delete config map from internal store
	c.resourceStore.ConfigMap = nil

	err := regenerateConfig(c)
	if err != nil {
		return err
	}

	logrus.Info("Caddy reloaded successfully.")
	return nil
}

func setConfigMapOptions(c *CaddyController, cfg *caddy.Config) error {
	// parse configmap
	cfgMap := ConfigMapOptions{}
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		WeaklyTypedInput: true,
		Result:           &cfgMap,
		TagName:          "json",
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		logrus.Warningf("unexpected error creating decoder: %v", err)
	}
	err = decoder.Decode(c.resourceStore.ConfigMap.Data)
	if err != nil {
		logrus.Warningf("unexpected error parsing configmap: %v", err)
	}

	logrus.Infof("using config map options: %+v to %+v", c.resourceStore.ConfigMap.Data, cfgMap)

	// merge configmap options to CaddyConfig
	tlsApp := cfg.Apps["tls"].(*caddytls.TLS)
	//httpApp := cfg.Apps["http"].(*caddyhttp.App)

	if cfgMap.Debug {
		cfg.Logging.Logs = map[string]*caddy2.CustomLog{"default": {Level: "DEBUG"}}
	}

	if cfgMap.AcmeCA != "" || cfgMap.Email != "" {
		acmeIssuer := caddytls.ACMEIssuer{}

		if cfgMap.AcmeCA != "" {
			acmeIssuer.CA = cfgMap.AcmeCA
		}

		if cfgMap.Email != "" {
			acmeIssuer.Email = cfgMap.Email
		}

		tlsApp.Automation = &caddytls.AutomationConfig{
			Policies: []*caddytls.AutomationPolicy{
				{IssuerRaw: caddyconfig.JSONModuleObject(acmeIssuer, "module", "acme", nil)},
			},
		}
	}

	return nil
}

func changeTriggerUpdate(c *CaddyController, cfgMap *v1.ConfigMap) bool {
	return cfgMap.Namespace == c.podInfo.Namespace && cfgMap.Name == c.config.ConfigMapName
}
