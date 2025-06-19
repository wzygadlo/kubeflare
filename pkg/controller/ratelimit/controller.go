package ratelimit

import (
	"context"
	"fmt"
	"strings"
	"time"

	kubeflarev1alpha1 "github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
	cfratelimit "github.com/replicatedhq/kubeflare/pkg/cloudflare/ratelimit"
	"github.com/replicatedhq/kubeflare/pkg/controller/shared"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	finalizerName = "ratelimit.kubeflare.replicated.com/finalizer"
)

// RateLimitReconciler reconciles a RateLimit object
type RateLimitReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	CFClient *cfratelimit.Client
}

// +kubebuilder:rbac:groups=kubeflare.replicated.com,resources=ratelimits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeflare.replicated.com,resources=ratelimits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeflare.replicated.com,resources=ratelimits/finalizers,verbs=update

// Reconcile handles reconciliation of RateLimit resources
func (r *RateLimitReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	logger := log.FromContext(ctx)

	// Fetch the RateLimit instance
	rateLimit := &kubeflarev1alpha1.RateLimit{}
	err := r.Get(ctx, req.NamespacedName, rateLimit)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, likely deleted
			return ctrl.Result{}, nil
		}
		// Error reading the object
		logger.Error(err, "Failed to get RateLimit")
		return ctrl.Result{}, err
	}

	// Get the zone for this rate limit
	zone, err := shared.GetZone(ctx, rateLimit.Namespace, rateLimit.Spec.ZoneID)
	if err != nil {
		logger.Error(err, "Failed to get zone", "zoneID", rateLimit.Spec.ZoneID)
		rateLimit.Status.Status = "Error"
		rateLimit.Status.Message = fmt.Sprintf("Failed to get zone: %v", err)
		if updateErr := r.Status().Update(ctx, rateLimit); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return reconcile.Result{RequeueAfter: time.Minute * 5}, err
	}

	// Initialize Cloudflare client for this zone
	cf, err := shared.GetCloudflareAPI(ctx, rateLimit.Namespace, zone.Spec.APIToken)
	if err != nil {
		logger.Error(err, "Failed to initialize Cloudflare API client")
		rateLimit.Status.Status = "Error"
		rateLimit.Status.Message = fmt.Sprintf("Failed to initialize Cloudflare API: %v", err)
		if updateErr := r.Status().Update(ctx, rateLimit); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return reconcile.Result{RequeueAfter: time.Minute * 5}, err
	}

	// Initialize rate limit client if not already
	if r.CFClient == nil {
		r.CFClient = cfratelimit.NewClient(cf)
	}

	// Check if the resource is being deleted
	if !rateLimit.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, rateLimit)
	}

	// Add finalizer if it doesn't exist
	if !containsString(rateLimit.ObjectMeta.Finalizers, finalizerName) {
		rateLimit.ObjectMeta.Finalizers = append(rateLimit.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, rateLimit); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// If no ID is present, or if the resource has been modified, sync with Cloudflare
	if rateLimit.Status.ID == "" || rateLimit.Generation != rateLimit.Status.ObservedGeneration {
		return r.reconcileSync(ctx, rateLimit)
	}

	// Periodically check that the rate limit still exists and is correctly configured
	// Try to get the rate limit from Cloudflare to verify it still exists
	cfRateLimit, err := r.CFClient.Get(ctx, rateLimit.Spec.ZoneID, rateLimit.Status.ID)
	if err != nil {
		// If the rate limit doesn't exist in Cloudflare anymore, clear the ID
		// Check if it's a 404 error by looking at the error string or using a different approach
		if isNotFoundError(err) {
			logger.Info("Rate limit not found in Cloudflare, will recreate", "id", rateLimit.Status.ID)
			rateLimit.Status.ID = ""
			rateLimit.Status.Status = "NotFound"
			rateLimit.Status.Message = "Rate limit not found in Cloudflare, will recreate"
			if updateErr := r.Status().Update(ctx, rateLimit); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return reconcile.Result{Requeue: true}, nil
		}

		logger.Error(err, "Failed to get rate limit from Cloudflare")
		return reconcile.Result{RequeueAfter: time.Minute * 5}, nil
	}

	logger.V(1).Info("Rate limit exists in Cloudflare", "id", cfRateLimit.ID)
	return reconcile.Result{RequeueAfter: time.Hour * 24}, nil
}

// reconcileDelete handles deletion of a RateLimit resource
func (r *RateLimitReconciler) reconcileDelete(ctx context.Context, rateLimit *kubeflarev1alpha1.RateLimit) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	// If the resource has a Cloudflare ID, delete it from Cloudflare
	if rateLimit.Status.ID != "" {
		err := r.CFClient.Delete(ctx, rateLimit.Spec.ZoneID, rateLimit.Status.ID)
		if err != nil {
			// If the rate limit doesn't exist (404), ignore the error
			if isNotFoundError(err) {
				logger.Info("Rate limit already deleted from Cloudflare", "id", rateLimit.Status.ID)
			} else {
				logger.Error(err, "Failed to delete rate limit from Cloudflare")
				return reconcile.Result{}, err
			}
		} else {
			logger.Info("Successfully deleted rate limit from Cloudflare", "id", rateLimit.Status.ID)
		}
	}

	// Remove finalizer
	rateLimit.ObjectMeta.Finalizers = removeString(rateLimit.ObjectMeta.Finalizers, finalizerName)
	if err := r.Update(ctx, rateLimit); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// reconcileSync handles creation or update of a RateLimit resource
func (r *RateLimitReconciler) reconcileSync(ctx context.Context, rateLimit *kubeflarev1alpha1.RateLimit) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	// If there's no ID, create a new rate limit
	if rateLimit.Status.ID == "" {
		id, err := r.CFClient.Create(ctx, rateLimit)
		if err != nil {
			logger.Error(err, "Failed to create rate limit in Cloudflare")
			rateLimit.Status.Status = "Error"
			rateLimit.Status.Message = fmt.Sprintf("Failed to create: %v", err)
			if updateErr := r.Status().Update(ctx, rateLimit); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return reconcile.Result{RequeueAfter: time.Minute * 5}, err
		}

		// Update status with new ID and generation
		rateLimit.Status.ID = id
		rateLimit.Status.Status = "Active"
		rateLimit.Status.Message = "Successfully created rate limit"
		rateLimit.Status.ObservedGeneration = rateLimit.Generation
		if err := r.Status().Update(ctx, rateLimit); err != nil {
			logger.Error(err, "Failed to update status")
			return reconcile.Result{}, err
		}

		logger.Info("Created rate limit in Cloudflare", "id", id)
		return reconcile.Result{}, nil
	}

	// Otherwise, update the existing rate limit
	err := r.CFClient.Update(ctx, rateLimit)
	if err != nil {
		// If the rate limit doesn't exist anymore, clear the ID so it will be recreated
		if isNotFoundError(err) {
			logger.Info("Rate limit not found in Cloudflare during update, will recreate", "id", rateLimit.Status.ID)
			rateLimit.Status.ID = ""
			rateLimit.Status.Status = "NotFound"
			rateLimit.Status.Message = "Rate limit not found in Cloudflare, will recreate"
			if updateErr := r.Status().Update(ctx, rateLimit); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return reconcile.Result{Requeue: true}, nil
		}

		logger.Error(err, "Failed to update rate limit in Cloudflare")
		rateLimit.Status.Status = "Error"
		rateLimit.Status.Message = fmt.Sprintf("Failed to update: %v", err)
		if updateErr := r.Status().Update(ctx, rateLimit); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return reconcile.Result{RequeueAfter: time.Minute * 5}, err
	}

	// Update observed generation
	rateLimit.Status.Status = "Active"
	rateLimit.Status.Message = "Successfully updated rate limit"
	rateLimit.Status.ObservedGeneration = rateLimit.Generation
	if err := r.Status().Update(ctx, rateLimit); err != nil {
		logger.Error(err, "Failed to update status")
		return reconcile.Result{}, err
	}

	logger.Info("Updated rate limit in Cloudflare", "id", rateLimit.Status.ID)
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *RateLimitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubeflarev1alpha1.RateLimit{}).
		Complete(r)
}

// Helper functions

// isNotFoundError checks if the error indicates a resource was not found (404)
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error contains "404" or "not found" in the message
	errStr := err.Error()
	return strings.Contains(strings.ToLower(errStr), "404") ||
		strings.Contains(strings.ToLower(errStr), "not found") ||
		strings.Contains(strings.ToLower(errStr), "resource not found")
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
