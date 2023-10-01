#!/usr/bin/env bash -ex

go run sigs.k8s.io/controller-tools/cmd/controller-gen@latest \
    object:headerFile="boilerplate.go.txt" \
    paths="./..."

go run sigs.k8s.io/controller-tools/cmd/controller-gen@latest \
    crd:ignoreUnexportedFields=true \
    paths="./..." \
    output:crd:artifacts:config=../../../deploy/crds

# Download JSON encoded CRDs to this package

go run sigs.k8s.io/controller-tools/cmd/controller-gen@latest \
    crd:ignoreUnexportedFields=true paths="./..." \
    output:crd:dir=./crds

# Convert each yaml file to JSON

for f in ./crds/*.yaml; do
    cat "$f" | yq > "${f%.*}.json"
    rm "$f"
done
