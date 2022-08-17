#!/bin/bash

set -eo pipefail
#set -x

endpoint=${1}
bucket_name=${2}
file_path=${3}
secret_name=${4}

access_key=$(kubectl -n crossplane-system get secret ${secret_name} -o jsonpath='{.data.EXOSCALE_API_KEY}' | base64 -d)
secret_key=$(kubectl -n crossplane-system get secret ${secret_name} -o jsonpath='{.data.EXOSCALE_API_SECRET}' | base64 -d)
export MC_HOST_exoscale=https://${access_key}:${secret_key}@${endpoint}

${GOBIN}/mc cp --quiet "${file_path}" "exoscale/${bucket_name}"
