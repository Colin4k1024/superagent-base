#!/usr/bin/env bash
set -euo pipefail
#
# Copyright 2025 coze-dev Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#


SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/../../" && pwd)"
BACKEND_DIR="$BASE_DIR/backend"
BIN_DIR="$BASE_DIR/bin"
CONFIG_DIR="$BIN_DIR/resources/conf"
RESOURCES_DIR="$BIN_DIR/resources/"
DOCKER_DIR="$BASE_DIR/docker"
# source "$DOCKER_DIR/.env"
ENV_FILE="$DOCKER_DIR/.env"

if [[ "$APP_ENV" == "debug" ]]; then
    ENV_FILE="$DOCKER_DIR/.env.debug"
elif [[ "$APP_ENV" == "oceanbase" ]]; then
    ENV_FILE="$DOCKER_DIR/.env"
fi

source "$ENV_FILE"

if [[ "$CODE_RUNNER_TYPE" == "sandbox" ]] && ! command -v deno &> /dev/null; then
    echo "deno is not installed, installing now..."
    curl -fsSL https://deno.land/install.sh | sh
    export PATH="$HOME/.deno/bin:$PATH"
fi

echo "🧹 Checking for sandbox availability..."

echo "🧹 Checking for goimports availability..."

if command -v goimports >/dev/null 2>&1; then
    echo "🧹 Formatting Go files with goimports..."
    find "$BACKEND_DIR" \
        -path "$BACKEND_DIR/api/model" -prune -o \
        -path "$BACKEND_DIR/api/router" -prune -o \
        -path "*/dal/query*" -prune -o \
        -path "*/mock/*" -prune -o \
        -path "*_mock.go" -prune -o \
        -path "*/dal/model*" -prune -o \
        -name "*.go" -exec goimports -w -local "github.com/coze-dev/coze-studio" {} \;
else
    echo "⚠️ goimports not found, skipping Go file formatting."
fi

echo "🛠  Building Go project..."
rm -rf "$BIN_DIR/opencoze"
cd $BACKEND_DIR &&
    go build -ldflags="-s -w" -o "$BIN_DIR/opencoze" main.go

# 添加构建失败检查
if [ $? -ne 0 ]; then
    echo "❌ Go build failed - aborting startup"
    exit 1
fi

echo "✅ Build completed successfully!"

echo "📑 Copying environment file..."
if [ -f "$ENV_FILE" ]; then
    cp "$ENV_FILE" "$BIN_DIR"
else
    echo "❌ .env file not found in $DOCKER_DIR"
    exit 1
fi


echo "📑 Cleaning configuration files..."
rm -rf "$CONFIG_DIR"
mkdir -p "$CONFIG_DIR"

echo "📑 Copying plugin configuration files..."

cp -r "$BACKEND_DIR/conf" "$RESOURCES_DIR"
cp -r "$BACKEND_DIR/static" "$RESOURCES_DIR"

for arg in "$@"; do
    if [[ "$arg" == "-start" ]]; then
        echo "🚀 Starting Go service..."
        cd $BIN_DIR && ./opencoze "$@"
        exit 0
    fi
done
