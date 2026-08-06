#!/bin/bash

set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-scw}"
BUILD_VERSION="${BUILD_VERSION:-dev}"
KUBECONFIG="${KUBECONFIG:-kubeconfig.yaml}"
DOCKER_IMAGE="scaleway/scaleway-secrets-store-csi"

export KUBECONFIG

echo "=== Deleting existing cluster (if any) ==="
kind delete cluster --name "$CLUSTER_NAME" || true

echo "=== Creating kind cluster '$CLUSTER_NAME' ==="
kind create cluster --name "$CLUSTER_NAME" --wait 60s

echo "=== Ensuring kube-system namespace ==="
kubectl create namespace kube-system --dry-run=client -o yaml | kubectl apply -f -

echo "=== Building docker image ==="
BUILD_VERSION="$BUILD_VERSION" make docker-build

echo "=== Loading image into kind cluster ==="
kind load docker-image "${DOCKER_IMAGE}:${BUILD_VERSION}" --name "$CLUSTER_NAME"

echo "=== Installing secrets-store-csi-driver helm chart ==="
helm repo add secrets-store-csi-driver https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts
helm install csi-secrets-store secrets-store-csi-driver/secrets-store-csi-driver --namespace kube-system

echo "=== Installing provider via Helm ==="
helm upgrade --install scaleway-secrets-store-csi --namespace kube-system scaleway/scaleway-secrets-store-csi \
  --set pod.image.repository="$DOCKER_IMAGE" \
  --set pod.image.tag="$BUILD_VERSION" \
  --set pod.image.pullPolicy=Never \
  --set provider.debug=true

echo "=== Provider pods ==="
kubectl get pods -n kube-system -l app.kubernetes.io/name=secrets-store-csi-driver-provider-scw

echo "=== Deployment complete ==="
