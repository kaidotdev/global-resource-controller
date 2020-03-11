package controllers

import (
	"context"
	grV1 "global-resource-controller/api/v1"

	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *NamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	namespace := &v1.Namespace{}
	ctx := context.Background()
	logger := r.Log.WithValues("namespace", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, namespace); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var gcms grV1.GlobalConfigMapList
	if err := r.List(
		ctx,
		&gcms,
	); err != nil {
		return ctrl.Result{}, err
	}
	for _, gcm := range gcms.Items {
		exists := false
		for _, n := range gcm.Status.TargetNamespaces {
			if n == namespace.Name {
				exists = true
			}
		}
		if !exists {
			gcm.Status.TargetNamespaces = append(gcm.Status.TargetNamespaces, namespace.Name)
			if err := r.Update(ctx, &gcm); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(&gcm, coreV1.EventTypeNormal, "SuccessfulUpdated", "Updated global config map: %q", gcm.Name)
			logger.V(1).Info("update", "global config map", gcm)
		}
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Namespace{}).
		Complete(r)
}
