// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

func deleteEnvArgs(envID string, force bool) []string {
	if force {
		return []string{envID, "--force"}
	}

	return []string{envID}
}

func reportsOfEnvironment(store *forge.ArtifactStore, envID string) []*forge.TestReport {
	if envID == "" {
		return nil
	}

	var reports []*forge.TestReport
	for _, report := range forge.ListTestReports(store, "") {
		if report.TestenvID == envID {
			reports = append(reports, report)
		}
	}

	sort.Slice(reports, func(i, j int) bool { return reports[i].ID < reports[j].ID })

	return reports
}

func refusalToDropReports(envID string, reports []*forge.TestReport) error {
	lines := make([]string, 0, len(reports)+2)
	lines = append(lines, fmt.Sprintf("environment %s still holds %d test report(s):", envID, len(reports)))
	for _, report := range reports {
		lines = append(lines, fmt.Sprintf("  %s  stage=%s  status=%s", report.ID, report.Stage, report.Status))
	}
	lines = append(lines, fmt.Sprintf("rerun with --force to remove the environment and its %d reports", len(reports)))

	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}

func guardEnvironmentReports(artifactStorePath, envID string, force bool) ([]*forge.TestReport, error) {
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		return nil, fmt.Errorf("reading artifact store %s: %w", artifactStorePath, err)
	}

	reports := reportsOfEnvironment(&store, envID)
	if len(reports) > 0 && !force {
		return nil, refusalToDropReports(envID, reports)
	}

	return reports, nil
}

func removeEnvironmentReports(artifactStorePath string, reports []*forge.TestReport) error {
	for _, report := range reports {
		if err := forge.AtomicDeleteTestReport(artifactStorePath, report.ID); err != nil {
			return fmt.Errorf("deleting test report %s: %w", report.ID, err)
		}

		fmt.Printf("Deleted test report: %s (stage=%s, status=%s)\n", report.ID, report.Stage, report.Status)
	}

	return nil
}
