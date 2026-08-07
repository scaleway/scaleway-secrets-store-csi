# Scaleway Secret Manager Provider for Secrets Store CSI Driver

[Scaleway Secret Manager](https://www.scaleway.com/fr/secret-manager/) provider for
[Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/) allows you to get secrets
stored in Scaleway Secret Manager and use the Secrets Store CSI driver interface to mount them into
Kubernetes pods.

## Installation

### Prerequisites

Install the
[Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/getting-started/installation.html)
in your Kubernetes cluster.

### Using Helm

The recommended installation method is via Helm 3:
```bash
helm repo add scaleway https://helm.scw.cloud/
helm repo update
helm upgrade --install scaleway-secrets-store-csi --namespace kube-system scaleway/scaleway-secrets-store-csi
```

The values file is available here:
[values.yaml](https://github.com/scaleway/helm-charts/blob/master/charts/scaleway-secrets-store-csi/values.yaml).

### Using kubectl

You can also install using the deployment config in the `deployment` folder:
```bash
kubectl apply -n kube-system -f deployment/secrets-store-csi-driver-provider-scw.yaml
```

## Usage

### Authentication

To authenticate to Scaleway, create a Kubernetes Secret with your credentials:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: scaleway-credentials
  namespace: default
type: Opaque
stringData:
  accessKey: "my-access-key"
  secretKey: "my-secret-key"
```

> **Note**
>
> The provider does not support authentication with a Kubernetes Service Account yet.

### SecretProviderClass

Create a SecretProviderClass to define which secrets to mount:
```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: scaleway-provider
  namespace: default
spec:
  provider: scaleway
  parameters:
    apiURL: "https://api.scaleway.com"
    insecure: "false"
    defaultOrganizationID: "573e26e8-45b7-4859-aa81-547de6e9acff"
    defaultProjectID: "6a4027ce-58c3-4adb-9b69-7ff6e1f317d2"
    defaultRegion: "fr-par"
    objects: |
      - secretID: "e6a5d04a-c2c2-49a3-8180-09452bf64e26"
        revision: "latest_enabled"
        targetPath: "my-secret"
      - projectID: "dfdc0cab-c99b-479d-a8f0-1df78cd9f67e"
        secretPath: "/test"
        secretName: "my-other-secret"
        revision: "latest_enabled"
```

### Accessing Secrets

#### By Secret ID

To access a secret version by its ID, specify the `secretID` field:
```yaml
objects: |
  - secretID: "e6a5d04a-c2c2-49a3-8180-09452bf64e26"
    revision: "latest_enabled"
    targetPath: "my-secret"
```

**Fields:**
- `secretID`: The UUID of the secret
- `revision`: The revision to access e.g., "latest_enabled", "1", "2", etc. (optional, defaults to `latest_enabled`)
- `targetPath`: The relative path where the secret will be mounted

#### By Secret Path

To access a secret version by its path, specify the `secretPath` and `secretName` fields:
```yaml
objects: |
  - projectID: "dfdc0cab-c99b-479d-a8f0-1df78cd9f67e"
    secretPath: "/test"
    secretName: "my-other-secret"
    revision: "latest_enabled"
```

**Fields:**
- `projectID`: The project ID containing the secret (optional, default to `defaultProjectID` or `defaultOrganizationID`)
- `secretPath`: The absolute path to the secret folder
- `secretName`: The name of the secret
- `revision`: The revision to access e.g., "latest_enabled", "1", "2", etc. (optional, defaults to `latest_enabled`)
- `targetPath`: The relative path where the secret will be mounted (optional, defaults to `secretPath/secretName`)

### Mounting Secrets in a Pod

Mount the secrets in your Pod by referencing the SecretProviderClass:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-secret-pod
  namespace: default
spec:
  containers:
  - name: busybox
    image: busybox
    command: ["sleep", "infinity"]
    volumeMounts:
    - name: secrets-store
      mountPath: "/mnt/secrets"
      readOnly: true
  volumes:
  - name: secrets-store
    csi:
      driver: secrets-store.csi.k8s.io
      readOnly: true
      volumeAttributes:
        secretProviderClass: scaleway-provider
      nodePublishSecretRef:
        name: scaleway-credentials
```

## Troubleshooting

### Checking Logs

To troubleshoot issues with the Scaleway CSI provider, look at logs from the CSI provider pod
running on the same node as your application pod:
```bash
kubectl get pods -n kube-system -o wide
# Find the Scaleway CSI provider pod running on the same node as your application pod
kubectl logs -n kube-system scaleway-secrets-store-csi-xxxxx
```

### Enabling Debug Mode

To enable debug mode when installing with Helm, set the `provider.debug` value to `true`:
```yaml
provider:
  debug: true
```

### Local testing

You can build and deploy the provider on a local cluster. It requires
[kind](https://kind.sigs.k8s.io/), [helm](https://helm.sh/) and
[kubectl](https://kubernetes.io/docs/reference/kubectl/) to be installed. Simply run:
```bash
./deploy-local.sh
```

Export the generated kubeconfig and access the cluster with kubectl:
```bash
export KUBECONFIG="kubeconfig.yaml"
kubectl get pods -o wide
```

## Contribute

If you are looking for a way to contribute please read the [contributing guide](./CONTRIBUTING.md).

### Code of conduct

Participation in the Kubernetes community is governed by the
[CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Reach us

We love feedback. Feel free to reach us on [Scaleway Slack community](https://slack.scaleway.com),
we are waiting for you on #secret-manager.
