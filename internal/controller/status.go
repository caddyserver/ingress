package controller

import (
	"fmt"

	"k8s.io/api/extensions/v1beta1"
)

// dispatchSync is run every syncInterval duration to sync ingress source address fields.
func (c *CaddyController) dispatchSync() {
	c.syncQueue.Add(SyncStatusAction{})
}

// SyncStatusAction provides an implementation of the action interface
type SyncStatusAction struct {
}

func (r SyncStatusAction) handle(c *CaddyController) error {
	c.syncStatus(c.resourceStore.Ingresses)
	return nil
}

// syncStatus ensures that the ingress source address points to this ingress controller's IP address.
func (c *CaddyController) syncStatus(ings []*v1beta1.Ingress) {
	// TODO :- update source address to ingress controller published address

	fmt.Println("Handle Synching Ingress Source Address")
}
