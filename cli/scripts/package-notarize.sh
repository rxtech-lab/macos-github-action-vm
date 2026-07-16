#!/bin/bash

set -e

if [ -z "${INSTALLER_SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Error: INSTALLER_SIGNING_CERTIFICATE_NAME is not set"
  exit 1
fi

if [ -z "${APPLE_ID}" ] || [ -z "${APPLE_ID_PWD}" ] || [ -z "${APPLE_TEAM_ID}" ]; then
  echo "Error: Apple ID credentials not set (APPLE_ID, APPLE_ID_PWD, APPLE_TEAM_ID)"
  exit 1
fi

source "$(dirname "$0")/binaries.sh"

PKG_FILE="${PKG_FILE:-rvmm_macOS_arm64.pkg}"
PKG_VERSION="${PKG_VERSION:-1.0}"
PKG_IDENTIFIER="${PKG_IDENTIFIER:-lab.rxtech.rvmm}"
TMP_DIR="tmp_pkg_build"

echo "Creating package structure"
mkdir -p "${TMP_DIR}/usr/local/bin"

for binary in "${BINARIES[@]}"; do
  BINARY_PATH="bin/${binary}"

  echo "Verifying binary signature: ${BINARY_PATH}"
  codesign --verify --verbose "${BINARY_PATH}" || {
    echo "Error: Binary ${binary} is not properly signed. Run sign.sh first."
    exit 1
  }

  cp "${BINARY_PATH}" "${TMP_DIR}/usr/local/bin/"
done

echo "Building pkg installer (version ${PKG_VERSION})"
pkgbuild --root "${TMP_DIR}" \
  --identifier "${PKG_IDENTIFIER}" \
  --version "${PKG_VERSION}" \
  --sign "${INSTALLER_SIGNING_CERTIFICATE_NAME}" \
  --install-location "/" \
  "${PKG_FILE}"

rm -rf "${TMP_DIR}"

echo "Submitting for notarization"
xcrun notarytool submit "${PKG_FILE}" --verbose --apple-id "${APPLE_ID}" --team-id "${APPLE_TEAM_ID}" --password "${APPLE_ID_PWD}" --wait

echo "Stapling notarization ticket"
xcrun stapler staple -v "${PKG_FILE}"

echo "Package created, signed, notarized and stapled successfully: ${PKG_FILE}"
