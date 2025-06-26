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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AccessPolicy struct {
	Decision   string   `json:"decision"`
	Name       string   `json:"name"`
	Include    []string `json:"include"` // TODO
	Precedence *int     `json:"precedence,omitempty"`
	Exclude    []string `json:"exclude,omitempty"` // TODO
	Require    []string `json:"require,omitempty"` // TODO
}

type CORSHeader struct {
	AllowedMethods   []string `json:"allowedMethods"`
	AllowedOrigins   []string `json:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"`
	AllowAllMethods  bool     `json:"allowAllMethods"`
	AllowAllOrigins  bool     `json:"allowAllOrigins"`
	AllowAllHeaders  bool     `json:"allowAllHeaders"`
	AllowCredentials bool     `json:"allowCredentials"`
	MaxAge           int      `json:"maxAge"`
}

// AccessApplicationSpec defines the desired state of AccessApplication
type AccessApplicationSpec struct {
	Zone                   string         `json:"zone"`
	Name                   string         `json:"name"`
	Domain                 string         `json:"domain"`
	SessionDuration        string         `json:"sessionDuration,omitempty"`
	AllowedIdPs            []string       `json:"allowedIdPs,omitempty"`
	AutoRedirectToIdentity *bool          `json:"autoRedirectToIdentity,omitempty"`
	CORSHeaders            *CORSHeader    `json:"corsHeaders,omitempty"`
	AccessPolicies         []AccessPolicy `json:"accessPolicies,omitempty"`
}

// AccessApplicationStatus defines the observed state of AccessApplication
type AccessApplicationStatus struct {
	ApplicationID string `json:"applicationID,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AccessApplication is the Schema for the accessapplication API
type AccessApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessApplicationSpec   `json:"spec,omitempty"`
	Status AccessApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AccessApplicationList contains a list of AccessApplication
type AccessApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AccessApplication{}, &AccessApplicationList{})
}
