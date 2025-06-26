// pkg/apis/kubeflare/v1alpha1/ratelimit_types.go

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RateLimitSpec defines the desired state of a Cloudflare Rate Limit
type RateLimitSpec struct {
	// ZoneID is the Cloudflare Zone ID
	ZoneID string `json:"zoneId"`

	// APITokenSecretRef is an optional reference to a secret containing the API token
	// If provided, this will be used instead of looking up the zone
	APITokenSecretRef *corev1.SecretKeySelector `json:"apiTokenSecretRef,omitempty"`

	// Description for the rate limit
	Description string `json:"description,omitempty"`

	// Threshold defines the number of requests that will trigger the rate limit
	Threshold int `json:"threshold"`

	// Period defines the time in seconds to count matching traffic
	// Valid values: 1, 60, 300, 600, 1800, 3600, 7200, 10800, 21600, 43200, 86400
	Period int `json:"period"`

	// Match defines the characteristics of traffic that will be counted towards the threshold
	Match RateLimitMatch `json:"match"`

	// Action defines the response when the threshold is exceeded
	Action RateLimitAction `json:"action"`

	// Disabled is a flag to disable the rate limit
	Disabled bool `json:"disabled,omitempty"`
}

// RateLimitMatch defines criteria for matching traffic
type RateLimitMatch struct {
	// Methods are the HTTP methods to match
	Methods []string `json:"methods,omitempty"`

	// Schemes are the HTTP schemes to match (HTTP|HTTPS)
	Schemes []string `json:"schemes,omitempty"`

	// URL defines characteristics of the URL to match
	URL RateLimitMatchURL `json:"url,omitempty"`
}

// RateLimitMatchURL defines URL matching criteria
type RateLimitMatchURL struct {
	// Patterns are URL patterns to match
	Patterns []string `json:"patterns,omitempty"`
}

// RateLimitAction defines the action when the rate limit is triggered
type RateLimitAction struct {
	// Mode defines the action taken when the threshold is reached
	// Valid values: simulate, ban, challenge, js_challenge
	Mode string `json:"mode"`

	// Timeout defines how long the action will be in effect (in seconds)
	Timeout int `json:"timeout,omitempty"`

	// Response is a custom response when the rate limit is triggered
	Response *RateLimitActionResponse `json:"response,omitempty"`
}

// RateLimitActionResponse defines a custom response
type RateLimitActionResponse struct {
	// ContentType is the Content-Type of the response
	ContentType string `json:"contentType"`

	// Body is the body of the response
	Body string `json:"body"`
}

// RateLimitStatus defines the observed state of a Rate Limit
type RateLimitStatus struct {
	// ID is the Cloudflare-assigned ID of the rate limit
	ID string `json:"id,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed RateLimit
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Status indicates the current status of the rate limit
	// "Active", "Error", etc.
	Status string `json:"status,omitempty"`

	// Message provides additional information about the status
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RateLimit is the Schema for the ratelimits API
type RateLimit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RateLimitSpec   `json:"spec,omitempty"`
	Status RateLimitStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RateLimitList contains a list of RateLimit resources
type RateLimitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RateLimit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RateLimit{}, &RateLimitList{})
}
