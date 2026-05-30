package controller

import (
	"k8s.io/client-go/util/workqueue"
)

type eventActionKind uint8

const (
	actionAdd eventActionKind = iota
	actionUpdate
	actionDelete
)

type eventAction[T any] struct {
	handlers *QueuedEventHandlers[T]
	action   eventActionKind
	resource *T
}

func (e eventAction[T]) handle(c *CaddyController) error {
	switch e.action {
	case actionAdd:
		if e.handlers.AddFunc != nil {
			return e.handlers.AddFunc(e.resource)
		}
	case actionUpdate:
		if e.handlers.UpdateFunc != nil {
			return e.handlers.UpdateFunc(e.resource)
		}
	case actionDelete:
		if e.handlers.DeleteFunc != nil {
			return e.handlers.DeleteFunc(e.resource)
		}
	default:
		panic("invalid action")
	}
	return nil
}

type QueuedEventHandlers[T any] struct {
	Queue      workqueue.TypedInterface[Action]
	FilterFunc func(obj *T) bool
	AddFunc    func(obj *T) error
	UpdateFunc func(obj *T) error
	DeleteFunc func(obj *T) error
}

func (r *QueuedEventHandlers[T]) queue(action eventActionKind, obj *T) {
	r.Queue.Add(eventAction[T]{
		handlers: r,
		action:   action,
		resource: obj,
	})
}

func (r *QueuedEventHandlers[T]) OnAdd(obj any, isInInitialList bool) {
	if obj, ok := obj.(*T); ok {
		if r.FilterFunc == nil || r.FilterFunc(obj) {
			r.queue(actionAdd, obj)
		}
	}
}

func (r *QueuedEventHandlers[T]) OnUpdate(oldObj, newObj any) {
	oldObjT, _ := oldObj.(*T)
	newObjT, _ := newObj.(*T)

	if r.FilterFunc != nil {
		if oldObjT != nil && !r.FilterFunc(oldObjT) {
			oldObjT = nil
		}
		if newObjT != nil && !r.FilterFunc(newObjT) {
			newObjT = nil
		}
	}

	if oldObjT != nil && newObjT != nil {
		r.queue(actionUpdate, newObjT)
	} else if oldObjT != nil {
		r.queue(actionDelete, oldObjT)
	} else if newObjT != nil {
		r.queue(actionAdd, newObjT)
	}
}

func (r *QueuedEventHandlers[T]) OnDelete(obj any) {
	if r.DeleteFunc != nil {
		if obj, ok := obj.(*T); ok {
			if r.FilterFunc == nil || r.FilterFunc(obj) {
				r.queue(actionDelete, obj)
			}
		}
	}
}
