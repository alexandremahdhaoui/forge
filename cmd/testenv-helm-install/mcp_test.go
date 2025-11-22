//go:build unit

package main

import "testing"

// Test ChartSpec validation and structure
func TestChartSpec(t *testing.T) {
	spec := ChartSpec{
		Name:            "my-podinfo",
		SourceType:      "helm-repo",
		URL:             "https://stefanprodan.github.io/podinfo",
		ChartName:       "podinfo",
		Version:         "6.0.0",
		Namespace:       "test-ns",
		ReleaseName:     "custom-release",
		CreateNamespace: true,
		DisableWait:     false,
		Timeout:         "10m",
	}

	// Test that the spec fields are set correctly
	if spec.SourceType != "helm-repo" {
		t.Errorf("SourceType = %v, want helm-repo", spec.SourceType)
	}
	if spec.ChartName != "podinfo" {
		t.Errorf("ChartName = %v, want podinfo", spec.ChartName)
	}
	if spec.URL != "https://stefanprodan.github.io/podinfo" {
		t.Errorf("URL = %v, want https://stefanprodan.github.io/podinfo", spec.URL)
	}
	if spec.Version != "6.0.0" {
		t.Errorf("Version = %v, want 6.0.0", spec.Version)
	}
	if spec.Namespace != "test-ns" {
		t.Errorf("Namespace = %v, want test-ns", spec.Namespace)
	}
	if spec.ReleaseName != "custom-release" {
		t.Errorf("ReleaseName = %v, want custom-release", spec.ReleaseName)
	}
	if !spec.CreateNamespace {
		t.Errorf("CreateNamespace = %v, want true", spec.CreateNamespace)
	}
	if spec.DisableWait {
		t.Errorf("DisableWait = %v, want false", spec.DisableWait)
	}
	if spec.Timeout != "10m" {
		t.Errorf("Timeout = %v, want 10m", spec.Timeout)
	}
}

func TestExtractRepoNameFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "podinfo repo",
			url:  "https://stefanprodan.github.io/podinfo",
			want: "podinfo",
		},
		{
			name: "with trailing slash",
			url:  "https://charts.bitnami.com/bitnami/",
			want: "bitnami",
		},
		{
			name: "simple url",
			url:  "https://example.com",
			want: "example.com",
		},
		{
			name: "nested path",
			url:  "https://charts.example.com/stable/releases",
			want: "releases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepoNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("extractRepoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
