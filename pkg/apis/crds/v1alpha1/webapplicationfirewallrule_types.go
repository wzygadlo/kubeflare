/*
Copyright 2020 Replicated, Inc.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WAFRule struct {
	ID        string `json:"id"`
	PackageID string `json:"packageid,omitempty"`
	Mode      string `json:"mode"`
}

// WebApplicationFirewallRuleSpec defines the desired state of WebApplicationFirewallRule
type WebApplicationFirewallRuleSpec struct {
	Zone  string     `json:"zone"`
	Rules []*WAFRule `json:"rules,omitempty"`
}

// WebApplicationFirewallRuleStatus defines the observed state of WebApplicationFirewallRule
type WebApplicationFirewallRuleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WebApplicationFirewallRule is the Schema for the webapplicationfirewallrules API
type WebApplicationFirewallRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebApplicationFirewallRuleSpec   `json:"spec,omitempty"`
	Status WebApplicationFirewallRuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WebApplicationFirewallRuleList contains a list of WebApplicationFirewallRule
type WebApplicationFirewallRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebApplicationFirewallRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WebApplicationFirewallRule{}, &WebApplicationFirewallRuleList{})
}
