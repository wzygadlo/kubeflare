package zone

import (
	"testing"

	"github.com/cloudflare/cloudflare-go/v4"
	crdsv1alpha1 "github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
	"github.com/stretchr/testify/assert"
)

var (
	trueValue  = true
	falseValue = false
	stringM    = "m"
)

func Test_compareAndUpdateMobileRedirectZoneSetting(t *testing.T) {
	tests := []struct {
		name         string
		zoneSetting  cloudflare.ZoneSetting
		desiredValue *crdsv1alpha1.MobileRedirect
		expected     bool
	}{
		{
			name: "no change",
			zoneSetting: cloudflare.ZoneSetting{
				Value: map[string]interface{}{
					"status":           "on",
					"mobile_subdomain": "m",
					"strip_uri":        false,
				},
			},
			desiredValue: &crdsv1alpha1.MobileRedirect{
				Status:          &trueValue,
				MobileSubdomain: &stringM,
				StripURI:        &falseValue,
			},
			expected: false,
		},
		{
			name: "changed subdomin only",
			zoneSetting: cloudflare.ZoneSetting{
				Value: map[string]interface{}{
					"status":           "on",
					"mobile_subdomain": "mm",
					"strip_uri":        false,
				},
			},
			desiredValue: &crdsv1alpha1.MobileRedirect{
				Status:          &trueValue,
				MobileSubdomain: &stringM,
				StripURI:        &falseValue,
			},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := compareAndUpdateMobileRedirectZoneSetting(&test.zoneSetting, test.desiredValue)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_compareAndUpdateStringArrayZoneSetting(t *testing.T) {
	tests := []struct {
		name         string
		zoneSetting  []interface{}
		desiredValue []string
		expected     bool
	}{
		{
			name: "no change",
			zoneSetting: []interface{}{
				"A",
			},
			desiredValue: []string{
				"A",
			},
			expected: false,
		},
		{
			name: "change",
			zoneSetting: []interface{}{
				"A",
				"B",
			},
			desiredValue: []string{
				"A",
			},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			zoneSetting := cloudflare.ZoneSetting{
				Value: test.zoneSetting,
			}
			desiredValue := []*string{}
			for _, d := range test.desiredValue {
				desiredValue = append(desiredValue, &d)
			}

			actual := compareAndUpdateStringArrayZoneSetting(&zoneSetting, desiredValue)
			assert.Equal(t, test.expected, actual)
		})
	}
}
