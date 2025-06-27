package shared

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

func GetZone(ctx context.Context, namespace string, zoneName string) (*v1alpha1.Zone, error) {
	k8sClient, err := GetK8sClient()
	if err != nil {
		return nil, err
	}

	zone := &v1alpha1.Zone{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      zoneName,
		Namespace: namespace,
	}, zone)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get zone")
	}

	return zone, nil
}
