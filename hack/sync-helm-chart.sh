#!/bin/sh
# Run after make manifests to distribute the generated APIs and permissions.
set -eu

cd "$(dirname "$0")/.."

chart=charts/octelium-operator

mkdir -p "$chart/crds" "$chart/files"

rm -f "$chart"/crds/*.yaml

cp config/crd/bases/*.yaml "$chart/crds/"
cp config/rbac/role.yaml "$chart/files/manager-role.yaml"
cp LICENSE-AGPL-3.0 "$chart/LICENSE"
cp LICENSE-APACHE "$chart/LICENSE-APACHE"
