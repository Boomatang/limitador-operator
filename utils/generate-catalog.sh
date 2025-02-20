#!/usr/bin/env bash

# Generate OLM catalog file

set -euo pipefail

### CONSTANTS
# Used as well in the subscription object
CHANNEL_NAME=preview
###

OPM="${1?:Error \$OPM not set. Bye}"
YQ="${2?:Error \$YQ not set. Bye}"
BUNDLE_IMG="${3?:Error \$BUNDLE_IMG not set. Bye}"
CATALOG_FILE="${4?:Error \$CATALOG_FILE not set. Bye}"

CATALOG_FILE_BASEDIR="$( cd "$( dirname "$(realpath ${CATALOG_FILE})" )" && pwd )"
CATALOG_BASEDIR="$( cd "$( dirname "$(realpath ${CATALOG_FILE_BASEDIR})" )" && pwd )"

TMP_DIR=$(mktemp -d)

${OPM} render ${BUNDLE_IMG} --output=yaml >> ${TMP_DIR}/limitador-operator-bundle.yaml

mkdir -p ${CATALOG_FILE_BASEDIR}
touch ${CATALOG_FILE}

###
# Limitador Operator
###
# Add the package
${OPM} init limitador-operator --default-channel=${CHANNEL_NAME} --output yaml >> ${CATALOG_FILE}
# Add a bundles to the Catalog
cat ${TMP_DIR}/limitador-operator-bundle.yaml >> ${CATALOG_FILE}
# Add a channel entry for the bundle
V=`${YQ} eval '.name' ${TMP_DIR}/limitador-operator-bundle.yaml` \
    ${YQ} eval '(.entries[0].name = strenv(V))' ${CATALOG_BASEDIR}/limitador-operator-channel-entry.yaml >> ${CATALOG_FILE}

rm -rf $TMP_DIR
