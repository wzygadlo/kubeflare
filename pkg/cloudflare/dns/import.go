package dns

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"strings"
)

// sanitizeResourceName creates a valid Kubernetes resource name
func sanitizeResourceName(name string) string {
	// Replace dots and other invalid characters with dashes
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-]`)
	sanitized := reg.ReplaceAllString(name, "-")

	// Ensure it starts and ends with alphanumeric
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		sanitized = "record"
	}

	// Kubernetes names must be lowercase
	return strings.ToLower(sanitized)
}

func FetchDNSRecordsForZone(token string, zone string, zoneID string) ([]*v1alpha1.DNSRecord, error) {
	cf, err := cloudflare.NewWithAPIToken(token)
	if err != nil {
		return nil, errors.Wrap(err, "create clouflare client")
	}

	rc := cloudflare.ZoneIdentifier(zoneID)
	resources, _, err := cf.ListDNSRecords(context.Background(), rc, cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return nil, errors.Wrap(err, "fetch resources")
	}

	dnsRecords := []*v1alpha1.DNSRecord{}
	for _, resource := range resources {
		// Skip records with empty essential fields
		if resource.Name == "" || resource.Type == "" || resource.Content == "" {
			continue
		}

		var priority *int
		if resource.Priority != nil {
			p := int(*resource.Priority)
			priority = &p
		}

		// Create a unique resource name based on DNS record name and type
		resourceName := fmt.Sprintf("%s-%s", sanitizeResourceName(resource.Name), strings.ToLower(resource.Type))

		dnsRecord := v1alpha1.DNSRecord{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "crds.kubeflare.io/v1alpha1",
				Kind:       "DNSRecord",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        resourceName,
				Namespace:   "default",
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			},
			Spec: v1alpha1.DNSRecordSpec{
				Zone: zone,
				Record: &v1alpha1.Record{
					Type:     resource.Type,
					Name:     resource.Name,
					Content:  resource.Content,
					TTL:      func() *int { ttl := int(resource.TTL); return &ttl }(),
					Priority: priority,
					Proxied:  resource.Proxied,
				},
			},
		}

		dnsRecords = append(dnsRecords, &dnsRecord)
	}

	return dnsRecords, nil
}
