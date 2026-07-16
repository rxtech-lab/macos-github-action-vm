#!/bin/bash

set -e

export COPYFILE_DISABLE=1

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
TMP_DIR="$(/usr/bin/mktemp -d /private/tmp/rvmm-pkg-root.XXXXXX)"
PKG_SCRIPTS_DIR="$(/usr/bin/mktemp -d /private/tmp/rvmm-pkg-scripts.XXXXXX)"

cleanup() {
  /bin/rm -rf "${TMP_DIR}" "${PKG_SCRIPTS_DIR}"
}
trap cleanup EXIT

echo "Creating package structure"
mkdir -p "${TMP_DIR}/usr/local/bin"
mkdir -p "${TMP_DIR}/Library/LaunchDaemons"
mkdir -p "${TMP_DIR}/Library/Application Support/RVMM/Updater/requests"
mkdir -p "${PKG_SCRIPTS_DIR}"
chmod 0770 "${TMP_DIR}/Library/Application Support/RVMM/Updater/requests"
cp "assets/lab.rxtech.rvmm.updater.plist" "${TMP_DIR}/Library/LaunchDaemons/"
chmod 0644 "${TMP_DIR}/Library/LaunchDaemons/lab.rxtech.rvmm.updater.plist"
cp "scripts/pkg-scripts/postinstall" "${PKG_SCRIPTS_DIR}/postinstall"
chmod 0755 "${PKG_SCRIPTS_DIR}/postinstall"

for binary in "${BINARIES[@]}"; do
  BINARY_PATH="bin/${binary}"

  echo "Verifying binary signature: ${BINARY_PATH}"
  codesign --verify --verbose "${BINARY_PATH}" || {
    echo "Error: Binary ${binary} is not properly signed. Run sign.sh first."
    exit 1
  }

  cp "${BINARY_PATH}" "${TMP_DIR}/usr/local/bin/"
done

# Avoid packaging macOS metadata sidecars such as ._rvmm.
/usr/bin/xattr -cr "${TMP_DIR}"

echo "Building pkg installer (version ${PKG_VERSION})"
pkgbuild --root "${TMP_DIR}" \
  --ownership recommended \
  --identifier "${PKG_IDENTIFIER}" \
  --version "${PKG_VERSION}" \
  --sign "${INSTALLER_SIGNING_CERTIFICATE_NAME}" \
  --scripts "${PKG_SCRIPTS_DIR}" \
  --install-location "/" \
  "${PKG_FILE}"

echo "Submitting for notarization"
xcrun notarytool submit "${PKG_FILE}" --verbose --apple-id "${APPLE_ID}" --team-id "${APPLE_TEAM_ID}" --password "${APPLE_ID_PWD}" --wait

echo "Stapling notarization ticket"
xcrun stapler staple -v "${PKG_FILE}"

echo "Writing SHA-256 checksum"
/usr/bin/shasum -a 256 "${PKG_FILE}" > "${PKG_FILE}.sha256"

echo "Package created, signed, notarized and stapled successfully: ${PKG_FILE}"
