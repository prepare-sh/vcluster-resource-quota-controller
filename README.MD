# Kubernetes Admission Controller for Resource Quotas

## Overview

This repository contains a Kubernetes Admission Controller that enforces resource management policies on Pods in a cluster. The primary use case for this admission controller is to control resource quotas for vCluster-based environments, where resource quotas need to be enforced on pods that are synced to the main cluster via the syncer. This controller does not control the vCluster system itself.

## Features

- **Label-based Resource Quotas:** Controls Pods based on specific labels, allowing for flexible resource management for various use cases.
- **Dynamic Configuration:** Configurable settings for CPU and memory limits, as well as mandatory resource requests and limits, through Kubernetes ConfigMaps.
- **Compatibility with Kubernetes:** Works seamlessly with Kubernetes' existing admission controller interfaces and resource management systems.

## How It Works

The admission controller intercepts Pod creation requests and validates them against predefined resource limits and requirements. If a Pod does not comply with the specified resource policies, it is rejected. This process ensures that resource quotas are enforced consistently across vCluster environments.

## Configuration

The admission controller uses a ConfigMap for dynamic configuration. Below is an example of the ConfigMap used to set the resource limits and mandatory resource requirements:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: webhook-config
  namespace: default
data:
  limitCPU: "500m"
  limitMemory: "500Mi"
  requireCPURequests: "true"
  requireMemoryRequests: "true"
  requireCPULimits: "true"
  requireMemoryLimits: "true"
```

### ConfigMap Fields

- **limitCPU:** Maximum CPU limit for a pod.
- **limitMemory:** Maximum memory limit for a pod.
- **requireCPURequests:** Whether CPU requests are mandatory.
- **requireMemoryRequests:** Whether memory requests are mandatory.
- **requireCPULimits:** Whether CPU limits are mandatory.
- **requireMemoryLimits:** Whether memory limits are mandatory.

## Usage

### Prerequisites

- Kubernetes cluster
- kubectl configured to interact with your cluster
- Certificate and key for TLS communication

### Deploying the Admission Controller

1. **Create ConfigMap:**
    ```sh
    kubectl apply -f webhook-config.yaml
    ```

2. **Deploy the Admission Controller:**
    Apply the deployment, service, and MutatingWebhookConfiguration files. Ensure that these files are correctly configured for your environment. 

    Example deployment file:
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: admission-controller
      namespace: default
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: admission-controller
      template:
        metadata:
          labels:
            app: admission-controller
        spec:
          containers:
          - name: admission-controller
            image: your-image-repository/admission-controller:latest
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
      name: webhook-config
    webhooks:
    - name: webhook.example.com
      clientConfig:
        service:
          name: admission-controller-service
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

- This admission controller specifically targets pods syncing to the main cluster via the `syncer` mechanism of vCluster.
- The policies are enforced based on labels. Ensure that the correct labels are applied to the desired pods.

## Extending for Other Use Cases

Although this admission controller is designed for vCluster environments, it can be adapted to enforce label-based resource quotas for other use cases. By changing the label key and value in the configuration, different resource management policies can be applied based on various labels.

## Kubernetes Support

Kubernetes currently supports quotas based on `PriorityClass`, but this admission controller adds the flexibility of label-based quotas, making it useful for more granular resource management scenarios.

## Acknowledgments

This project uses the `k8s.io` client libraries for interacting with Kubernetes APIs and the `admissionv1` library for handling admission controller webhooks.

## Contact

For any questions or support, please contact alex@prepare.sh