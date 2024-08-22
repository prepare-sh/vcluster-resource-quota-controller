# Kubernetes Admission Controller for vCluster Resource Quotas

## Overview

This repository contains a Kubernetes Admission Controller designed to enforce resource quotas on Pods in a vCluster environment. The controller ensures that resource limits are properly adhered to for Pods that are synced to the main cluster via the syncer and not for the vCluster system itself. Currently, Kubernetes supports only `PriorityClass` based quotas, but this controller provides label-based quotas, making it flexible for various use cases.

## Features

- **Label-based Resource Quotas:** Controls Pods based on specific labels, enabling resource management for various scenarios.
- **Dynamic Configuration:** Resource limits and mandatory resource requirements are configurable via Kubernetes ConfigMaps.
- **Seamless Integration:** Works seamlessly with Kubernetes' admission controller interfaces and resource management systems.

## Configuration

The admission controller uses a ConfigMap for dynamic configuration. Below is an example of the ConfigMap used to set the resource limits and mandatory resource requirements:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vcluster-resource-quota-controller-config
  namespace: default
data:
  limitCPU: "500m"
  limitMemory: "500Mi"
```

### ConfigMap Fields

- **limitCPU:** Maximum CPU limit for a pod.
- **limitMemory:** Maximum memory limit for a pod.

## How It Works

The admission controller intercepts Pod creation requests and validates them against predefined resource limits and requirements. If a Pod does not comply with the specified resource policies, it is rejected, ensuring that resource quotas are enforced consistently across vCluster environments.

## Usage

### Prerequisites

- Kubernetes cluster
- `kubectl` configured to interact with your cluster
- Certificate and key for TLS communication

### Deploying the Admission Controller

1. **Create ConfigMap:**

    Apply the ConfigMap configuration file.

    ```sh
    kubectl apply -f webhook-config.yaml
    ```

2. **Deploy the Admission Controller:**

    Apply the deployment, service, and MutatingWebhookConfiguration files. Ensure these files are correctly configured for your environment.
    
    Example deployment file:
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: vcluster-resource-quota-controller
      namespace: default
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: vcluster-resource-quota-controller
      template:
        metadata:
          labels:
            app: vcluster-resource-quota-controller
        spec:
          containers:
          - name: webhook
            image: your-image-repository/webhook:latest
            ports:
            - containerPort: 8443
            volumeMounts:
            - name: webhook-certs
              mountPath: "/etc/webhook/certs"
              readOnly: true
          volumes:
          - name: webhook-certs
            secret:
              secretName: webhook-server-cert
    ```

3. **Create the TLS Secret:**

    Ensure you have a valid TLS certificate and key stored in a Kubernetes Secret named `webhook-server-cert`.

4. **Register the Webhook:**

    Apply the MutatingWebhookConfiguration to register your webhook with the API server.

    Example configuration:
    ```yaml
    apiVersion: admissionregistration.k8s.io/v1
    kind: MutatingWebhookConfiguration
    metadata:
      name: vcluster-resource-quota-controller
    webhooks:
    - name: webhook.vcluster.example.com
      clientConfig:
        service:
          name: vcluster-resource-quota-controller
          namespace: default
          path: "/validate"
        caBundle: <base64-encoded-CA-cert>
      rules:
      - operations: ["CREATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
      admissionReviewVersions: ["v1"]
    ```

## Limitations

- This admission controller specifically targets pods with the label `vcluster.loft.sh/managed-by` syncing to the main cluster via the `syncer` mechanism of vCluster.
- Policies are enforced based on labels. Ensure that the correct labels are applied to the desired pods.

## Extending for Other Use Cases

Although this admission controller is designed for vCluster environments, it can be adapted to enforce label-based resource quotas for other use cases. By changing the label key and value in the configuration, different resource management policies can be applied.

## Kubernetes Support

Kubernetes currently supports quotas based on `PriorityClass`, but this admission controller adds the flexibility of label-based quotas, making it useful for more granular resource management scenarios.

## Acknowledgments

This project uses the `k8s.io` client libraries for interacting with Kubernetes APIs and the `admissionv1` library for handling admission controller webhooks.

## Contact

For any questions or support, please contact alex@prepare.sh