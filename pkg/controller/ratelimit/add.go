package ratelimit

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Add creates a new RateLimit Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconciler := newReconciler(mgr)
	return reconciler.SetupWithManager(mgr)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *RateLimitReconciler {
	return &RateLimitReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		CFClient: nil, // Will be initialized during reconciliation for each zone
	}
}
