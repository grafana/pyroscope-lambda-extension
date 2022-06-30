#!/usr/bin/env bash

set -euo pipefail

DRY_RUN=""
# Enable this when debugging
#DRY_RUN="enabled"

name="${1:-pyroscope-extension-test}"
region="us-east-1"

export PS4='+(${BASH_SOURCE}:${LINENO}): ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'



dryrun() {
    # Is empty?
    if [[ -z "$DRY_RUN" ]]; then
     # Execute as normally
     $@
    else
      printf -v cmd_str '%q ' "$@"; echo "[DRY-RUN] '$cmd_str'" >&2
    fi
}

make clean

# Build both versions
echo "Building..."
make build-amd
make build-arm

make package-amd
make package-arm

echo "Publishing..."

echo "Publishing x86_64"

# When running in dry-run mode, use a dummy version
# This is so that we don't need to call aws
pushd bin/x86_64
amdVersion="DEV"
publishAmdCmd="dryrun aws lambda publish-layer-version --layer-name=$name-x86_64 --region=us-east-1 --zip-file fileb://extension.zip"
if [[ -z "$DRY_RUN" ]]; then
  amdVersion=$($publishAmdCmd | jq '.Version')
else
  dryrun "$publishAmdCmd"
fi
popd
echo "AMD Version: '$amdVersion'"


# Publish arm version
echo "Publishing arm64"
pushd bin/x86_64
armVersion="DEV"
publishArmCmd="dryrun aws lambda publish-layer-version --layer-name=$name-arm64 --region=us-east-1 --zip-file fileb://extension.zip"
if [[ -z "$DRY_RUN" ]]; then
  armVersion=$($publishArmCmd | jq '.Version')
else
  dryrun "$publishArmCmd"
fi
popd
echo "ARM Version: '$armVersion'"


echo "Making extension public..."

# Make them public
echo "Making x86_64 public"
dryrun aws lambda add-layer-version-permission \
  --region="$region" \
  --layer-name="$name-x86_64" \
  --statement-id="$name-$amdVersion-$region" \
  --version-number="$amdVersion" \
  --principal="*" \
  --action lambda:GetLayerVersion

echo "Making arm64 public"
dryrun aws lambda add-layer-version-permission \
  --region="$region" \
  --layer-name="$name-arm64" \
  --statement-id="$name-$amdVersion-$region" \
  --version-number="$armVersion" \
  --principal="*" \
  --action lambda:GetLayerVersion
