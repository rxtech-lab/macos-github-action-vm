#!/bin/bash

set -e

if [ -z "${SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Error: SIGNING_CERTIFICATE_NAME is not set"
  exit 1
fi

source "$(dirname "$0")/binaries.sh"

for binary in "${BINARIES[@]}"; do
  BINARY_PATH="bin/${binary}"

  echo "Signing binary: ${BINARY_PATH}"
  codesign --force --options runtime --timestamp --sign "${SIGNING_CERTIFICATE_NAME}" "${BINARY_PATH}"
  codesign --verify --verbose "${BINARY_PATH}"
done

echo "Binaries signed successfully"
