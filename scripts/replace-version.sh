#!/usr/bin/env bash

set -euo pipefail

# Replace hello-world/template.yml with the latest version

# Query latest lambda
latestFullLayer=$(aws lambda list-layer-versions --layer-name pyroscope-extension-test --region us-east-1 --query 'max_by(LayerVersions, &Version).LayerVersionArn' --output=text)

latestLayer=$(echo "$latestFullLayer" | awk -F':' 'BEGIN { OFS = FS }; NF { NF -= 1 }; 1')
latestVersion=$(echo "$latestFullLayer" | awk -F: '{print $NF}')

# Replace the existing layer with the new one
# TODO(eh-am): fail if there's no match (eg the layer name has changed)
sed -i .bak -e "s@$latestLayer.*@$latestLayer:$latestVersion@g" hello-world/template.yml
