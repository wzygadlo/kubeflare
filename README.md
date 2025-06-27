# Kubeflare - Simplified Security Controller

Kubeflare is a Kubernetes operator that enables declarative management of Cloudflare security settings through Custom Resource Definitions (CRDs). This streamlined version focuses specifically on security-related features:

- **Rate Limiting**: Protect your applications from abuse and DDoS attacks
- **Web Application Firewall (WAF)**: Configure security rules to block malicious traffic

## Overview

After installing the Kubeflare operator to your Kubernetes cluster, you can define Rate Limit and WAF rules as Kubernetes resources. When these resources are deployed to the cluster, the Kubeflare operator will reconcile them with the Cloudflare API to enforce your security policies.

## Key Features

- **Cloudflare SDK v4.5.1**: Uses the latest Cloudflare Go SDK for API integration
- **Modern Rulesets API**: Implements rate limiting using Cloudflare's recommended Rulesets API
- **GitOps Compatible**: Manage your security rules as code in your Git repositories
- **Kubernetes Native**: Follows standard Kubernetes patterns for resource management

## Installation

Refer to the [COMPLETE_DEPLOYMENT_GUIDE.md](COMPLETE_DEPLOYMENT_GUIDE.md) for detailed installation instructions.

Quick start:
```bash
# Install CRDs
kubectl apply -f config/crds/v1/crds.kubeflare.io_ratelimits.yaml
kubectl apply -f config/crds/v1/crds.kubeflare.io_webapplicationfirewallrules.yaml

# Create Cloudflare API token secret
kubectl create secret generic cloudflare-api-token \
  --from-literal=token=YOUR_CLOUDFLARE_API_TOKEN

# Deploy the controller
kubectl apply -f deploy/ratelimit/deployment.yaml
```

## Examples

Below is an example of a Rate Limit resource:

```yaml
apiVersion: crds.kubeflare.io/v1alpha1
kind: RateLimit
metadata:
  name: api-rate-limit
spec:
  zoneID: "your-cloudflare-zone-id"
  apiTokenSecretRef:
    name: cloudflare-api-token
    key: token
  description: "API Rate Limiting"
  disabled: false
  threshold: 100
  period: 60
  match:
    request:
      methods: ["POST", "PUT", "DELETE"]
      urlPattern: "/api/*"
    response:
      originTraffic: false
  action:
    mode: "challenge"
    timeout: 86400
```

Here's an example of a WAF rule:

```yaml
apiVersion: crds.kubeflare.io/v1alpha1
kind: WebApplicationFirewallRule
metadata:
  name: security-rules
spec:
  zoneName: "example.com"
  rules:
    - id: "rule-1"
      description: "Block SQL Injection Attempts"
      mode: "challenge"
      priority: "high"
      packageID: "sqli"
      group: "OWASP"
      status: "enabled"
```

## Security Features Supported

This streamlined version of Kubeflare focuses on security features:

- **Rate Limiting**: Protect your API endpoints and web applications from abuse
- **WAF Rules**: Configure Web Application Firewall rules to block common attack patterns

## Documentation

- [Deployment Guide](COMPLETE_DEPLOYMENT_GUIDE.md): Complete instructions for deployment
- [Rate Limits Documentation](docs/rate-limits.md): Details on Rate Limiting configuration

This project is independent of Cloudflare and built using their public APIs. This is not a Cloudflare project.
