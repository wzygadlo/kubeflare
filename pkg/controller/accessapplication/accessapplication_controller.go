/*
Copyright 2019 Replicated, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package accessapplication

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	crdsv1alpha1 "github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
	"github.com/replicatedhq/kubeflare/pkg/controller/shared"
	"github.com/replicatedhq/kubeflare/pkg/logger"
)

// Add creates a new AccessApplication Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAccessApplication{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create controller using new builder pattern
	err := builder.
		ControllerManagedBy(mgr).
		For(&crdsv1alpha1.AccessApplication{}).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed to create accessapplication controller")
	}

	generatedClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	generatedInformers := kubeinformers.NewSharedInformerFactory(generatedClient, time.Minute)
	err = mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		generatedInformers.Start(ctx.Done())
		<-ctx.Done()
		return nil
	}))
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAccessApplication{}

// ReconcileAccessApplication reconciles a AccessApplication object
type ReconcileAccessApplication struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ReconcileAccessApplication object and makes changes based on the state read
// and what is in the Zone.Spec
// +kubebuilder:rbac:groups=crds.kubeflare.io,resources=accessapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crds.kubeflare.io,resources=accessapplications/status,verbs=get;update;patch
func (r *ReconcileAccessApplication) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// This reconcile loop will be called for all ReconcileAccessApplication objects
	// because of the informer that we have set up
	instance := crdsv1alpha1.AccessApplication{}
	err := r.Get(ctx, request.NamespacedName, &instance)
	if err != nil {
		logger.Error(err)
		return reconcile.Result{}, err
	}

	zone, err := shared.GetZone(ctx, instance.Namespace, instance.Spec.Zone)
	if err != nil {
		logger.Error(err)
		return reconcile.Result{}, err
	}

	cf, err := shared.GetCloudflareAPI(ctx, instance.Namespace, zone.Spec.APIToken)
	if err != nil {
		logger.Error(err)
		return reconcile.Result{}, err
	}

	// if the instanace status subresource doesn't contain an application id, update it now
	if instance.Status.ApplicationID == "" {
		existingApplication, err := findExistingAccessApplication(instance, zone, cf)
		if err != nil {
			logger.Error(err)
			return reconcile.Result{}, nil
		}

		if existingApplication != nil {
			instance.Status.ApplicationID = existingApplication.ID
			err := r.Status().Update(ctx, &instance)
			if err != nil {
				logger.Error(err)
				return reconcile.Result{}, nil
			}
		}
	}

	_, err = ReconcileAccessApplicationInstance(ctx, instance, zone, cf)
	if err != nil {
		logger.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
