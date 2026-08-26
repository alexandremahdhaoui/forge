#!/bin/sh
# Cross-build every command for the distribution: CGO off, trimmed, static,
# named by the travel convention <name>_<os>_<arch> under build/dist, so
# the binaries run on any linux of their architecture and the release side
# picks them up by shape alone.
#
# The revision the pipeline proved arrives as FORGE_CI_REVISION and is
# stamped into the binaries, so a released binary knows the distribution it
# shipped with. A -X against a symbol a command does not carry is ignored
# by the linker, so one stamp line serves every command.
set -eu

PLATFORMS="${DIST_PLATFORMS:-linux/amd64 linux/arm64}"
REVISION="${FORGE_CI_REVISION:-}"
LABEL=$(git describe --tags --always 2>/dev/null || echo dev)

STAMP="-s -w -X main.Version=$LABEL -X main.version=$LABEL"
if [ -n "$REVISION" ]; then
    STAMP="$STAMP -X github.com/alexandremahdhaoui/forge/pkg/toolresolver.CompanionRevision=$REVISION"
fi

rm -rf build/dist
mkdir -p build/dist

for dir in ./cmd/*/; do
    name=$(basename "$dir")

    for platform in $PLATFORMS; do
        os=${platform%/*}
        arch=${platform#*/}

        GOWORK=off GOFLAGS= CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
            go build -trimpath -ldflags "$STAMP" \
            -o "build/dist/${name}_${os}_${arch}" "$dir"
    done
done

echo "dist: $(ls build/dist | wc -l) binaries for $PLATFORMS"
