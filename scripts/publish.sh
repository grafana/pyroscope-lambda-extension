#!/usr/bin/env bash

set -euo pipefail

DRY_RUN=""
# Enable this when debugging
#DRY_RUN="enabled"
export PS4='+(${BASH_SOURCE}:${LINENO}): ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'

name="${1:-pyroscope-extension-test}"
amdName="$name-x86_64"
armName="$name-arm64"

# To get the list of regions run
# aws ec2 describe-regions | jq '.Regions[].RegionName'
REGIONS=("us-east-1")


dryrun() {
    # Is empty?
    if [[ -z "$DRY_RUN" ]]; then
     # Execute as normally
     "$@"
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


for region in "${REGIONS[@]}"; do
  echo "Publishing name='$amdName' region=$region "
  publishAmdCmd="dryrun aws lambda publish-layer-version --layer-name=$amdName --region=$region --zip-file fileb://extension.zip"
  if [[ -z "$DRY_RUN" ]]; then
    amdVersion=$($publishAmdCmd | jq '.Version')
  else
    dryrun "$publishAmdCmd"
  fi
done
echo "AMD Version: '$amdVersion'"
popd


# Publish arm version
echo "Publishing arm64"
pushd bin/x86_64
armVersion="DEV"

for region in "${REGIONS[@]}"; do
  echo "Publishing name='$armName' region=$region"
  publishArmCmd="dryrun aws lambda publish-layer-version --layer-name=$armName --region=$region --zip-file fileb://extension.zip"
  if [[ -z "$DRY_RUN" ]]; then
    armVersion=$($publishArmCmd | jq '.Version')
  else
    dryrun "$publishArmCmd"
  fi
done

popd
echo "ARM Version: '$armVersion'"


echo "Making extension public..."

# Make them public
echo "Making x86_64 public"
for region in "${REGIONS[@]}"; do
  echo "Making public name='$amdName' version='$amdVersion' region=$region"
  dryrun aws lambda add-layer-version-permission \
    --region="$region" \
    --layer-name="$amdName" \
    --statement-id="$name-$amdVersion-$region" \
    --version-number="$amdVersion" \
    --principal="*" \
    --action lambda:GetLayerVersion
done

echo "Making arm64 public"
for region in "${REGIONS[@]}"; do
  echo "Making public name='$armName' version='$armVersion' region=$region"
  dryrun aws lambda add-layer-version-permission \
    --region="$region" \
    --layer-name="$armName" \
    --statement-id="$name-$armVersion-$region" \
    --version-number="$armVersion" \
    --principal="*" \
    --action lambda:GetLayerVersion
done
