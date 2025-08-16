#!/bin/bash
# This script generates a Software Bill of Materials (SBOM) in CycloneDX format.

# Ensure cyclonedx-gomod is installed
if ! command -v cyclonedx-gomod &> /dev/null
then
    echo "cyclonedx-gomod could not be found, installing..."
    go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
fi

echo "Generating SBOM..."
cyclonedx-gomod mod -output sbom.xml
