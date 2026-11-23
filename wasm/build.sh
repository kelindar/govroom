#!/usr/bin/env bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
sdk=$(realpath -- "${1:?usage: bash wasm/build.sh /path/to/wasi-sdk-34.0}")
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

git clone --quiet --depth 1 --branch v1.15.0 https://github.com/VROOM-Project/vroom.git "$work/vroom"
cd "$work/vroom"
test "$(git rev-parse HEAD)" = 43dd7d0b8b560431eb555bf335cf4797eb7343c4
git submodule update --init --depth 1 include/rapidjson
git apply "$root/wasm/serial.patch"
cp "$root/wasm/main.cpp" src/main.cpp

# VROOM's explicit errors can exit the disposable WASI instance directly.
sed -E -i 's/\bthrow (InputException|RoutingException|InternalException)\(/::vroom::fail<\1>(/g' \
  src/algorithms/heuristics/heuristics.cpp src/routing/wrapper.h \
  src/structures/vroom/{cost_wrapper,time_window,tw_route,vehicle}.cpp \
  src/structures/vroom/input/input.cpp \
  src/utils/{helpers.cpp,helpers.h,input_parser.cpp}

make -C src -j "${JOBS:-4}" ../bin/vroom \
  USE_ROUTING=false GLPK_HEADER= \
  CXX="$sdk/bin/clang++" \
  CXXFLAGS="-MMD -MP -I. -std=c++20 -O3 -DUSE_ROUTING=false -DNDEBUG -fno-exceptions" \
  LDLIBS="-Wl,-z,stack-size=1048576,--strip-all"

cp bin/vroom "$root/wasm/vroom.wasm"
