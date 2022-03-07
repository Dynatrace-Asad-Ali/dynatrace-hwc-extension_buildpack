# Copyright 2020 Dynatrace LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

    # http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/bash
set -euo pipefail

GO_VERSION="1.10"
export GoInstallDir="/tmp/go$GO_VERSION"
mkdir -p $GoInstallDir

if [ ! -f $GoInstallDir/go/bin/go ]; then
GO_SHA256="244200952f414e9ae6269d32569722a7cd88435f5c52d488cd9599b8bfa1498b"
URL=https://buildpacks.cloudfoundry.org/dependencies/go/go${GO_VERSION}.linux-amd64-${GO_SHA256:0:8}.tar.gz

echo "-----> Download go ${GO_VERSION}"
curl -s -L --retry 15 --retry-delay 2 $URL -o /tmp/go.tar.gz

DOWNLOAD_SHA256=$(shasum -a256 /tmp/go.tar.gz | cut -d ' ' -f 1)
if [[ $DOWNLOAD_SHA256 != $GO_SHA256 ]]; then
echo "       **ERROR** SHA256 mismatch: got $DOWNLOAD_SHA256 expected $GO_SHA256"
exit 1
fi

tar xzf /tmp/go.tar.gz -C $GoInstallDir
rm /tmp/go.tar.gz
fi
if [ ! -f $GoInstallDir/go/bin/go ]; then
echo "       **ERROR** Could not download go"
exit 1
fi
