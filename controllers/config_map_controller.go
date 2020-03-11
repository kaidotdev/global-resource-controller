package controllers

import (
	"context"
	grV1 "global-resource-controller/api/v1"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GlobalConfigMapReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *GlobalConfigMapReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	gcm := &grV1.GlobalConfigMap{}
	ctx := context.Background()
	logger := r.Log.WithValues("gcm", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, gcm); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.cleanupOwnedResources(ctx, gcm); err != nil {
		return ctrl.Result{}, err
	}

	if gcm.ObjectMeta.DeletionTimestamp.IsZero() {
		if len(gcm.Status.TargetNamespaces) == 0 {
			var namespaces v1.NamespaceList
			if err := r.List(
				ctx,
				&namespaces,
			); err != nil {
				return ctrl.Result{}, err
			}
			for _, namespace := range namespaces.Items {
				gcm.Status.TargetNamespaces = append(gcm.Status.TargetNamespaces, namespace.Name)
			}
			if !containsString(gcm.ObjectMeta.Finalizers, myFinalizerName) {
				gcm.ObjectMeta.Finalizers = append(gcm.ObjectMeta.Finalizers, myFinalizerName)
			}
			if err := r.Update(ctx, gcm); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			m := excludeNamespaces(gcm)
			for _, namespaceName := range gcm.Status.TargetNamespaces {
				if m[namespaceName] {
					continue
				}
				var namespace v1.Namespace
				if err := r.Client.Get(ctx, client.ObjectKey{Name: namespaceName}, &namespace); err != nil {
					if errors.IsNotFound(err) {
						return ctrl.Result{}, nil
					}
				}

				var configMap v1.ConfigMap
				if err := r.Client.Get(
					ctx,
					client.ObjectKey{
						Name:      req.Name + "-global",
						Namespace: namespace.Name,
					},
					&configMap,
				); errors.IsNotFound(err) {
					configMap = *r.buildConfigMap(gcm, &namespace)
					if err := controllerutil.SetControllerReference(gcm, &configMap, r.Scheme); err != nil {
						return ctrl.Result{}, err
					}
					if err := r.Create(ctx, &configMap); err != nil {
						return ctrl.Result{}, err
					}
					r.Recorder.Eventf(gcm, coreV1.EventTypeNormal, "SuccessfulCreated", "Created config map: %q(%q)", configMap.Name, namespace.Name)
					logger.V(1).Info("create", "config map", configMap)
				} else if err != nil {
					return ctrl.Result{}, err
				} else {
					expectedConfigMap := r.buildConfigMap(gcm, &namespace)
					if !reflect.DeepEqual(configMap.Data, expectedConfigMap.Data) ||
						!reflect.DeepEqual(configMap.BinaryData, expectedConfigMap.BinaryData) {
						configMap.Data = expectedConfigMap.Data
						configMap.BinaryData = expectedConfigMap.BinaryData

						if err := r.Update(ctx, &configMap); err != nil {
							return ctrl.Result{}, err
						}
						r.Recorder.Eventf(gcm, coreV1.EventTypeNormal, "SuccessfulUpdated", "Updated config map: %q(%q)", configMap.Name, namespace.Name)
						logger.V(1).Info("update", "config map", configMap)
					}
				}
			}
		}
	} else {
		if containsString(gcm.ObjectMeta.Finalizers, myFinalizerName) {
			var configMaps v1.ConfigMapList
			if err := r.List(
				ctx,
				&configMaps,
				client.MatchingFields{ownerKey: gcm.Name},
			); err != nil {
				return ctrl.Result{}, err
			}

			if err := r.Client.Delete(ctx, &configMaps); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(gcm, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted config maps: %q", configMaps)

			gcm.ObjectMeta.Finalizers = removeString(gcm.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(ctx, gcm); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func excludeNamespaces(gcm *grV1.GlobalConfigMap) map[string]bool {
	m := make(map[string]bool)
	for _, namespace := range gcm.Spec.ExcludeNamespaces {
		m[namespace] = true
	}
	return m
}

func (r *GlobalConfigMapReconciler) buildConfigMap(gcm *grV1.GlobalConfigMap, namespace *v1.Namespace) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      gcm.Name + "-global",
			Namespace: namespace.Name,
		},
		Data:       gcm.Spec.Template.Data,
		BinaryData: gcm.Spec.Template.BinaryData,
	}
}

func (r *GlobalConfigMapReconciler) cleanupOwnedResources(ctx context.Context, gcm *grV1.GlobalConfigMap) error {
	var configMaps v1.ConfigMapList
	if err := r.List(
		ctx,
		&configMaps,
		client.MatchingFields{ownerKey: gcm.Name},
	); err != nil {
		return err
	}

	for _, configMap := range configMaps.Items {
		configMap := configMap

		if configMap.Name == gcm.Name+"-global" {
			continue
		}

		if err := r.Client.Delete(ctx, &configMap); err != nil {
			return err
		}
		r.Recorder.Eventf(gcm, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted config map: %q(%q)", configMap.Name, configMap.Namespace)
	}

	return nil
}

func (r *GlobalConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&v1.ConfigMap{}, ownerKey, func(rawObj runtime.Object) []string {
		configMap := rawObj.(*v1.ConfigMap)
		owner := metaV1.GetControllerOf(configMap)
		if owner == nil {
			return nil
		}
		if owner.Kind != "GlobalConfigMap" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&grV1.GlobalConfigMap{}).
		Complete(r)
}
