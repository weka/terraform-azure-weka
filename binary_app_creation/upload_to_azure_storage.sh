#!/bin/bash

# Usage:
# upload_to_azure_storage.sh <file_path> <resource_group> <storage_account_name> <container_name> [<blob_name>]

set -e

file_path="$1"
resource_group="$2"
storage_account_name="$3"
container_name="$4"
blob_name="${5:-$(basename "$file_path")}"

echo "Uploading $file_path to $storage_account_name/$container_name as $blob_name..."

# Get the storage account key using the Azure CLI
storage_account_key=$(az storage account keys list --resource-group "$resource_group" --account-name "$storage_account_name" --query '[0].value' --output tsv)

# Upload the zip file to the specified container
az storage blob upload \
    --account-name "$storage_account_name" \
    --account-key "$storage_account_key" \
    --container-name "$container_name" \
    --type block \
    --name "$blob_name" \
    --file "$file_path" \
    --overwrite

echo "Upload complete."
