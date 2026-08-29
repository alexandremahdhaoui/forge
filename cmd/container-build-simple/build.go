/*
Copyright 2024 Alexandre Mahdhaoui

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/v1/layout"

	"github.com/alexandremahdhaoui/forge/internal/imageassembly"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// ArtifactType is what the release side filters on. A container is not a file
// somebody uploads, so it never travels through the binary path.
const ArtifactType = "container"

var (
	// ErrNoMatch means a glob matched nothing. An image that silently ships
	// empty fails on the runner that tries to use it, days later and far
	// from the cause, so the build fails here instead.
	ErrNoMatch = errors.New("a from glob matched nothing")
	// ErrEmptyPlatform means a declared platform got no files.
	ErrEmptyPlatform = errors.New("a declared platform got no files")
)

// travelSuffix reads the name_os_arch convention a cross-built file travels
// under. It is how a file is matched to the platform it belongs to.
func travelSuffix(path string) (imageassembly.Platform, bool) {
	base := filepath.Base(path)

	parts := strings.Split(base, "_")
	if len(parts) < 3 {
		return imageassembly.Platform{}, false
	}

	return imageassembly.Platform{
		OS:   parts[len(parts)-2],
		Arch: parts[len(parts)-1],
	}, true
}

// expand resolves the globs against root and answers the matches, sorted so
// two runs over the same tree assemble the same layer.
func expand(root string, globs []string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}

	for _, glob := range globs {
		pattern := glob
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(root, glob)
		}

		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("reading the glob %q: %w", glob, err)
		}

		found := 0

		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || info.IsDir() {
				continue
			}

			found++

			if !seen[m] {
				seen[m] = true

				out = append(out, m)
			}
		}

		// Per glob, not overall: one glob quietly matching nothing while the
		// others matched is exactly how an image ships missing a tool.
		if found == 0 {
			return nil, fmt.Errorf("%w: %q", ErrNoMatch, glob)
		}
	}

	sort.Strings(out)

	return out, nil
}

// group sorts the files onto the platforms they belong to. A file carrying a
// name_os_arch suffix goes to that platform alone; a file carrying none goes
// to every platform, because a script or a certificate is the same on all of
// them.
func group(files []string, platforms []imageassembly.Platform) (map[imageassembly.Platform][]string, error) {
	out := map[imageassembly.Platform][]string{}
	for _, p := range platforms {
		out[p] = []string{}
	}

	declared := map[imageassembly.Platform]bool{}
	for _, p := range platforms {
		declared[p] = true
	}

	for _, f := range files {
		platform, suffixed := travelSuffix(f)

		switch {
		case suffixed && declared[platform]:
			out[platform] = append(out[platform], f)

		case suffixed:
			// A file built for a platform nobody declared is not an error:
			// build/dist holds every platform, and this image carries the
			// subset it was asked for.
			continue

		default:
			for _, p := range platforms {
				out[p] = append(out[p], f)
			}
		}
	}

	for _, p := range platforms {
		if len(out[p]) == 0 {
			return nil, fmt.Errorf("%w: %s", ErrEmptyPlatform, p)
		}
	}

	return out, nil
}

// Build assembles the image and writes it to disk as an OCI image layout. It
// names no registry and pushes nothing: a build writes a file and a release
// publishes it, exactly as a binary does, so this engine holds no credential.
func Build(_ context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
	root := input.Context
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("reading the working directory: %w", err)
		}

		root = cwd
	}

	platforms, err := platformsOf(spec)
	if err != nil {
		return nil, err
	}

	files, err := expand(root, spec.From)
	if err != nil {
		return nil, err
	}

	byPlatform, err := group(files, platforms)
	if err != nil {
		return nil, err
	}

	binDir := spec.BinDir
	if binDir == "" {
		binDir = "/usr/local/bin"
	}

	base := spec.Base
	if base == "" {
		base = "debian:stable-slim"
	}

	log.Printf("assembling %s: base %s, %d files, %d platform(s)",
		input.Name, base, len(files), len(platforms))

	images, err := imageassembly.New(&imageassembly.Remote{Token: os.Getenv("REGISTRY_TOKEN")}).
		Assemble(imageassembly.Request{
			Base:   base,
			BinDir: binDir,
			Files:  byPlatform,
			Env:    spec.Env,
			Labels: spec.Labels,
		})
	if err != nil {
		return nil, err
	}

	index, err := imageassembly.Index(images)
	if err != nil {
		return nil, err
	}

	dest := input.Dest
	if dest == "" {
		dest = filepath.Join(root, "build", "images")
	} else if !filepath.IsAbs(dest) {
		dest = filepath.Join(root, dest)
	}

	out := filepath.Join(dest, input.Name+".oci")

	// A stale layout would be merged into rather than replaced, so the image
	// would carry manifests from a build nobody asked to keep.
	if err := os.RemoveAll(out); err != nil {
		return nil, fmt.Errorf("clearing %s: %w", out, err)
	}

	if err := os.MkdirAll(out, 0o750); err != nil {
		return nil, fmt.Errorf("creating %s: %w", out, err)
	}

	if _, err := layout.Write(out, index); err != nil {
		return nil, fmt.Errorf("writing the image layout to %s: %w", out, err)
	}

	digest, err := index.Digest()
	if err != nil {
		return nil, fmt.Errorf("reading the image digest: %w", err)
	}

	log.Printf("wrote %s (%s)", out, digest)

	return &forge.Artifact{
		Name:      input.Name,
		Type:      ArtifactType,
		Location:  "file://" + out,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   digest.String(),
	}, nil
}

func platformsOf(spec *Spec) ([]imageassembly.Platform, error) {
	raw := spec.Platforms
	if len(raw) == 0 {
		raw = []string{"linux/amd64"}
	}

	out := make([]imageassembly.Platform, 0, len(raw))

	for _, s := range raw {
		p, err := imageassembly.ParsePlatform(s)
		if err != nil {
			return nil, err
		}

		out = append(out, p)
	}

	return out, nil
}
