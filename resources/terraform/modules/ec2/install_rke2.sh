#!/bin/bash

# Define default values
DEFAULT_RKE2_VERSION="v1.32.2+rke2r1"
DEFAULT_CERT_MANAGER_VERSION="v1.15.3"
DEFAULT_HELM_REPO_NAME="rancher"
DEFAULT_HELM_REPO_URL="https://releases.rancher.com/server-charts/latest"

# Get inputs or use defaults
RKE2_VERSION="${1:-$DEFAULT_RKE2_VERSION}"
CERT_MANAGER_VERSION="${2:-$DEFAULT_CERT_MANAGER_VERSION}"
HELM_REPO_URL="${3:-$DEFAULT_HELM_REPO_URL}"

echo "🚀 Installing RKE2 version: $RKE2_VERSION"
echo "🔐 Installing Cert Manager version: $CERT_MANAGER_VERSION"
echo "📦 Using Helm repo URL: $HELM_REPO_URL"

# Install RKE2
curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=$RKE2_VERSION sh -
systemctl enable --now rke2-server.service
systemctl restart rke2-server

# Configure kubectl
mkdir -p ~/.kube
ln -sf /etc/rancher/rke2/rke2.yaml ~/.kube/config
ln -sf /var/lib/rancher/rke2/bin/kubectl /usr/local/bin/

# Install Helm
echo "📦 Installing Helm..."
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh
rm -f get_helm.sh

# Add Rancher Helm repo (with default name 'rancher')
echo "📌 Adding Helm repo '$DEFAULT_HELM_REPO_NAME' -> $HELM_REPO_URL"
helm repo add "$DEFAULT_HELM_REPO_NAME" "$HELM_REPO_URL"
helm repo update

# Install Cert Manager
echo "🔧 Installing Cert Manager version: $CERT_MANAGER_VERSION"
kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/$CERT_MANAGER_VERSION/cert-manager.yaml"

# Create Rancher namespace
kubectl create namespace cattle-system

echo "✅ Installation complete! RKE2 and Rancher Helm repo is set up."
