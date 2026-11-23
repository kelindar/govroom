#!/usr/bin/env bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
sdk=$(realpath -- "${1:?usage: bash wasm/build.sh /path/to/wasi-sdk-34.0}")
command -v wasm-opt >/dev/null
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

git clone --quiet --depth 1 --branch v1.15.0 https://github.com/VROOM-Project/vroom.git "$work/vroom"
cd "$work/vroom"
test "$(git rev-parse HEAD)" = 43dd7d0b8b560431eb555bf335cf4797eb7343c4
git submodule update --init --depth 1 include/cxxopts include/rapidjson
git apply "$root/wasm/serial.patch"

make -C src -j "${JOBS:-4}" ../bin/vroom \
  USE_ROUTING=false GLPK_HEADER= \
  CXX="$sdk/bin/clang++" \
  CXXFLAGS="-MMD -MP -I. -std=c++20 -O3 -DUSE_ROUTING=false -DNDEBUG -fwasm-exceptions" \
  LDLIBS="-lunwind -Wl,-z,stack-size=1048576,--strip-all"

# The WASI SDK emits legacy exception instructions. wazero uses exnref.
wasm-opt --enable-exception-handling --enable-reference-types \
  --enable-bulk-memory --enable-mutable-globals --enable-multivalue \
  --enable-nontrapping-float-to-int --enable-sign-ext \
  --translate-to-exnref bin/vroom -o "$work/vroom.wasm"
cp "$work/vroom.wasm" "$root/wasm/vroom.wasm"
