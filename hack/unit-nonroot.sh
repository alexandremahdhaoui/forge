#!/bin/sh
# Run the unit stage as a user who cannot write to /.
#
# Twice now a unit test passed "/cache", "/clone" or "/register" straight into
# code that creates a directory or a lock file. Both were green here and both
# failed on a colleague's machine. The difference was not the code. It was that
# this container runs as root, and root can write to /.
#
# A grep cannot tell those two lines apart from the hundred correct ones. A
# literal path handed to a mocked runner touches nothing and is fine; the same
# literal handed to real code creates a directory at the root of the machine.
# Running the tests as somebody who cannot write to / tells them apart exactly,
# with no allowlist and no heuristic.
set -eu

: "${NONROOT_USER:=forgetest}"

if [ "$(id -u)" -ne 0 ]; then
    echo "already unprivileged as $(id -un); running the unit stage directly"
    exec env GOWORK=off go test ./...
fi

if ! id "$NONROOT_USER" >/dev/null 2>&1; then
    useradd -m "$NONROOT_USER" >/dev/null 2>&1 ||
        adduser -D "$NONROOT_USER" >/dev/null 2>&1 || {
        echo "cannot create $NONROOT_USER, and this must not pass by default" >&2
        echo "set NONROOT_USER to an existing unprivileged account" >&2
        exit 1
    }
fi

home=$(getent passwd "$NONROOT_USER" | cut -d: -f6)
work="$home/forge"

# The tests read the tree and write only under TMPDIR, so a copy the user owns
# is enough and leaves the real checkout untouched.
rm -rf "$work"
mkdir -p "$work"
tar -cf - --exclude=.git --exclude=build . | (cd "$work" && tar -xf -)
mkdir -p "$home/go" "$home/.cache"
chown -R "$NONROOT_USER" "$work" "$home/go" "$home/.cache"

# Prove the premise before trusting the result. If / is writable by this user
# the stage measures nothing, and a green that measures nothing is the bug.
if su "$NONROOT_USER" -c 'mkdir /forge-nonroot-probe' 2>/dev/null; then
    rmdir /forge-nonroot-probe
    echo "$NONROOT_USER can write to /, so this stage would prove nothing" >&2
    exit 1
fi

su "$NONROOT_USER" -c "cd '$work' && \
    HOME='$home' GOPATH='$home/go' GOCACHE='$home/.cache/go-build' \
    GOWORK=off GOFLAGS=-mod=mod GOMODCACHE='${GOMODCACHE:-$home/go/pkg/mod}' \
    $(command -v go) test ./..."
