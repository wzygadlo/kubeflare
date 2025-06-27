package shared

import (
	"context"
	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
	"github.com/replicatedhq/kubeflare/pkg/logger"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var HasDependenciesError = errors.New("dependency detected")

func GetCloudflareAPI(ctx context.Context, namespace string, apiTokenName string) (*cloudflare.Client, error) {
	k8sClient, err := GetK8sClient()
	if err != nil {
		return nil, err
	}

	apiToken := &v1alpha1.APIToken{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      apiTokenName,
		Namespace: namespace,
	}, apiToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get api token")
	}

	tokenValue, err := apiToken.GetTokenValue(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token value")
	}

	logger.Debug("creating cloudflare api object",
		zap.String("email", apiToken.Spec.Email),
		zap.Int("tokenLength", len(tokenValue)))

	api := cloudflare.NewClient(option.WithAPIToken(tokenValue))
	if api == nil {
		return nil, errors.New("failed to create cloudflare api instance")
	}

	return api, nil
}

func GetK8sClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
	}

	k8sClient, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes client")
	}

	return k8sClient, nil
}

// GetCrdClient returns a placeholder client for kubeflare CRDs
// This is a temporary placeholder for MVP - we'll implement it properly in Phase 4
func GetCrdClient() (interface{}, error) {
	// Return a placeholder value - the controllers using this are disabled in MVP
	return nil, nil
}
