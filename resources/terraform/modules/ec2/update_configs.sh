#!/bin/bash
set -e

# We no longer read $1, $2, etc. We expect Environment Variables.

echo "--- Starting Secure Configuration Update ---"

# --- Update Input Cluster Config ---
if [ -f "$INPUT_CONFIG_PATH" ]; then
    echo "Found input config file."
    # Log generic messages, do not log the actual IDs
    echo "Updating Machine Config with VPC and Subnet details..."
    
    # We use the environment variables directly in yq
    yq e ".machineconfig.data.subnetId = \"$SUBNET_ID\" | .machineconfig.data.vpcId = \"$VPC_ID\"" \
       -i "$INPUT_CONFIG_PATH"
       
    echo "Successfully updated input cluster config."
else
    echo "Warning: Input config file not found at path. Skipping."
fi

# --- Update Cattle Config ---
if [ -f "$CATTLE_CONFIG_PATH" ]; then
    echo "Found cattle config file."
    echo "Updating Rancher Host IP..."
    
    yq e ".rancher.host = \"rancher.$RKE2_HOST_IP.sslip.io\"" \
       -i "$CATTLE_CONFIG_PATH"
       
    echo "Successfully updated cattle config."
else
    echo "Warning: Cattle config file not found. Skipping."
fi

echo "--- Configuration Update Complete ---"