#!/bin/bash

# shellcheck disable=SC2005,SC2086
echo "$(go list -m -f '{{.Dir}}' $1)"

