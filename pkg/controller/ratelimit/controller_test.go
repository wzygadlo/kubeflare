package ratelimit

import (
	"context"
	"testing"
	"time"

	kubeflarev1alpha1alpha1 "github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
	cfratelimit "github.com/replicatedhq/kubeflare/pkg/cloudflare/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MockCFClient mocks the Cloudflare rate limit client
type MockCFClient struct {
	mock.Mock
}

func (m *MockCFClient) Create(ctx context.Context, rateLimit *kubeflarev1alpha1.RateLimit) (string, error) {
	args := m.Called(ctx, rateLimit)
	return args.String(0), args.Error(1)
}

func (m *MockCFClient) Get(ctx context.Context, zoneID, ruleID string) (*cfratelimit.RateLimitRule, error) {
	args := m.Called(ctx, zoneID, ruleID)
	return args.Get(0).(*cfratelimit.RateLimitRule), args.Error(1)
}

func (m *MockCFClient) Update(ctx context.Context, rateLimit *kubeflarev1alpha1.RateLimit) error {
	args := m.Called(ctx, rateLimit)
	return args.Error(0)
}

func (m *MockCFClient) Delete(ctx context.Context, zoneID, ruleID string) error {
	args := m.Called(ctx, zoneID, ruleID)
	return args.Error(0)
}

func (m *MockCFClient) List(ctx context.Context, zoneID string) ([]*cfratelimit.RateLimitRule, error) {
	args := m.Called(ctx, zoneID)
	return args.Get(0).([]*cfratelimit.RateLimitRule), args.Error(1)
}

func TestRateLimitReconciler_ReconcileCreate(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	kubeflarev1alpha1.AddToScheme(scheme)

	rateLimit := &kubeflarev1alpha1.RateLimit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rate-limit",
			Namespace: "default",
		},
		Spec: kubeflarev1alpha1.RateLimitSpec{
			ZoneID:      "test-zone-id",
			Description: "Test rate limit",
			Threshold:   100,
			Period:      60,
		},
	}

	client := fake.NewFakeClient(rateLimit)

	mockCF := &MockCFClient{}
	mockCF.On("Create", mock.Anything, rateLimit).Return("test-rule-id", nil)

	reconciler := &RateLimitReconciler{
		Client:   client,
		Scheme:   scheme,
		CFClient: mockCF,
	}

	// Test
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-rate-limit",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
	mockCF.AssertExpectations(t)
}

func TestRateLimitReconciler_ReconcileUpdate(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	kubeflarev1alpha1.AddToScheme(scheme)

	rateLimit := &kubeflarev1alpha1.RateLimit{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-rate-limit",
			Namespace:  "default",
			Generation: 2,
		},
		Spec: kubeflarev1alpha1.RateLimitSpec{
			ZoneID:      "test-zone-id",
			Description: "Updated rate limit",
			Threshold:   200,
			Period:      120,
		},
		Status: kubeflarev1alpha1.RateLimitStatus{
			ID:                 "existing-rule-id",
			ObservedGeneration: 1,
		},
	}

	client := fake.NewFakeClientWithScheme(scheme, rateLimit)

	mockCF := &MockCFClient{}
	mockCF.On("Update", mock.Anything, rateLimit).Return(nil)

	reconciler := &RateLimitReconciler{
		Client:   client,
		Scheme:   scheme,
		CFClient: mockCF,
	}

	// Test
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-rate-limit",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
	mockCF.AssertExpectations(t)
}

func TestRateLimitReconciler_ReconcileDelete(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	kubeflarev1alpha1.AddToScheme(scheme)

	now := metav1.NewTime(time.Now())
	rateLimit := &kubeflarev1alpha1.RateLimit{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-rate-limit",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{finalizerName},
		},
		Spec: kubeflarev1alpha1.RateLimitSpec{
			ZoneID: "test-zone-id",
		},
		Status: kubeflarev1alpha1.RateLimitStatus{
			ID: "test-rule-id",
		},
	}

	client := fake.NewFakeClientWithScheme(scheme, rateLimit)

	mockCF := &MockCFClient{}
	mockCF.On("Delete", mock.Anything, "test-zone-id", "test-rule-id").Return(nil)

	reconciler := &RateLimitReconciler{
		Client:   client,
		Scheme:   scheme,
		CFClient: mockCF,
	}

	// Test
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-rate-limit",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
	mockCF.AssertExpectations(t)
}
