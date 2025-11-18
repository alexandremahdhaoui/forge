//go:build unit

package engineframework

import (
	"reflect"
	"testing"
)

func TestExtractString(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		wantVal string
		wantOk  bool
	}{
		{
			name:    "existing string value",
			spec:    map[string]any{"name": "my-app"},
			key:     "name",
			wantVal: "my-app",
			wantOk:  true,
		},
		{
			name:    "missing key",
			spec:    map[string]any{"name": "my-app"},
			key:     "missing",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "wrong type - int",
			spec:    map[string]any{"count": 42},
			key:     "count",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "wrong type - bool",
			spec:    map[string]any{"enabled": true},
			key:     "enabled",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "nil spec",
			spec:    nil,
			key:     "name",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "empty string value",
			spec:    map[string]any{"name": ""},
			key:     "name",
			wantVal: "",
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := ExtractString(tt.spec, tt.key)
			if gotVal != tt.wantVal || gotOk != tt.wantOk {
				t.Errorf("ExtractString() = (%q, %v), want (%q, %v)", gotVal, gotOk, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestExtractStringWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		spec         map[string]any
		key          string
		defaultValue string
		want         string
	}{
		{
			name:         "existing value",
			spec:         map[string]any{"name": "my-app"},
			key:          "name",
			defaultValue: "default",
			want:         "my-app",
		},
		{
			name:         "missing key uses default",
			spec:         map[string]any{"name": "my-app"},
			key:          "missing",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "wrong type uses default",
			spec:         map[string]any{"count": 42},
			key:          "count",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractStringWithDefault(tt.spec, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("ExtractStringWithDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractStringSlice(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		wantVal []string
		wantOk  bool
	}{
		{
			name:    "existing []string",
			spec:    map[string]any{"tags": []string{"a", "b", "c"}},
			key:     "tags",
			wantVal: []string{"a", "b", "c"},
			wantOk:  true,
		},
		{
			name:    "[]any with string elements (JSON unmarshal case)",
			spec:    map[string]any{"tags": []any{"a", "b", "c"}},
			key:     "tags",
			wantVal: []string{"a", "b", "c"},
			wantOk:  true,
		},
		{
			name:    "empty []string",
			spec:    map[string]any{"tags": []string{}},
			key:     "tags",
			wantVal: []string{},
			wantOk:  true,
		},
		{
			name:    "missing key",
			spec:    map[string]any{"tags": []string{"a"}},
			key:     "missing",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "wrong type - string",
			spec:    map[string]any{"name": "foo"},
			key:     "name",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "wrong type - []int",
			spec:    map[string]any{"numbers": []int{1, 2, 3}},
			key:     "numbers",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "[]any with mixed types",
			spec:    map[string]any{"mixed": []any{"a", 42, "c"}},
			key:     "mixed",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "nil spec",
			spec:    nil,
			key:     "tags",
			wantVal: nil,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := ExtractStringSlice(tt.spec, tt.key)
			if !reflect.DeepEqual(gotVal, tt.wantVal) || gotOk != tt.wantOk {
				t.Errorf("ExtractStringSlice() = (%v, %v), want (%v, %v)", gotVal, gotOk, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestExtractStringSliceWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		spec         map[string]any
		key          string
		defaultValue []string
		want         []string
	}{
		{
			name:         "existing value",
			spec:         map[string]any{"tags": []string{"a", "b"}},
			key:          "tags",
			defaultValue: []string{"default"},
			want:         []string{"a", "b"},
		},
		{
			name:         "missing key uses default",
			spec:         map[string]any{"tags": []string{"a"}},
			key:          "missing",
			defaultValue: []string{"default"},
			want:         []string{"default"},
		},
		{
			name:         "wrong type uses default",
			spec:         map[string]any{"numbers": []int{1, 2}},
			key:          "numbers",
			defaultValue: []string{"default"},
			want:         []string{"default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractStringSliceWithDefault(tt.spec, tt.key, tt.defaultValue)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractStringSliceWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractStringMap(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		wantVal map[string]string
		wantOk  bool
	}{
		{
			name:    "existing map[string]string",
			spec:    map[string]any{"labels": map[string]string{"app": "foo", "env": "dev"}},
			key:     "labels",
			wantVal: map[string]string{"app": "foo", "env": "dev"},
			wantOk:  true,
		},
		{
			name:    "map[string]any with string values (JSON unmarshal case)",
			spec:    map[string]any{"labels": map[string]any{"app": "foo", "env": "dev"}},
			key:     "labels",
			wantVal: map[string]string{"app": "foo", "env": "dev"},
			wantOk:  true,
		},
		{
			name:    "empty map",
			spec:    map[string]any{"labels": map[string]string{}},
			key:     "labels",
			wantVal: map[string]string{},
			wantOk:  true,
		},
		{
			name:    "missing key",
			spec:    map[string]any{"labels": map[string]string{"app": "foo"}},
			key:     "missing",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "wrong type - string",
			spec:    map[string]any{"name": "foo"},
			key:     "name",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "map[string]any with non-string values",
			spec:    map[string]any{"mixed": map[string]any{"app": "foo", "count": 42}},
			key:     "mixed",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "nil spec",
			spec:    nil,
			key:     "labels",
			wantVal: nil,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := ExtractStringMap(tt.spec, tt.key)
			if !reflect.DeepEqual(gotVal, tt.wantVal) || gotOk != tt.wantOk {
				t.Errorf("ExtractStringMap() = (%v, %v), want (%v, %v)", gotVal, gotOk, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestExtractStringMapWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		spec         map[string]any
		key          string
		defaultValue map[string]string
		want         map[string]string
	}{
		{
			name:         "existing value",
			spec:         map[string]any{"labels": map[string]string{"app": "foo"}},
			key:          "labels",
			defaultValue: map[string]string{"default": "value"},
			want:         map[string]string{"app": "foo"},
		},
		{
			name:         "missing key uses default",
			spec:         map[string]any{"labels": map[string]string{"app": "foo"}},
			key:          "missing",
			defaultValue: map[string]string{"default": "value"},
			want:         map[string]string{"default": "value"},
		},
		{
			name:         "wrong type uses default",
			spec:         map[string]any{"name": "foo"},
			key:          "name",
			defaultValue: map[string]string{"default": "value"},
			want:         map[string]string{"default": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractStringMapWithDefault(tt.spec, tt.key, tt.defaultValue)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractStringMapWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractBool(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		wantVal bool
		wantOk  bool
	}{
		{
			name:    "existing bool true",
			spec:    map[string]any{"enabled": true},
			key:     "enabled",
			wantVal: true,
			wantOk:  true,
		},
		{
			name:    "existing bool false",
			spec:    map[string]any{"enabled": false},
			key:     "enabled",
			wantVal: false,
			wantOk:  true,
		},
		{
			name:    "missing key",
			spec:    map[string]any{"enabled": true},
			key:     "missing",
			wantVal: false,
			wantOk:  false,
		},
		{
			name:    "wrong type - string",
			spec:    map[string]any{"name": "true"},
			key:     "name",
			wantVal: false,
			wantOk:  false,
		},
		{
			name:    "wrong type - int",
			spec:    map[string]any{"count": 1},
			key:     "count",
			wantVal: false,
			wantOk:  false,
		},
		{
			name:    "nil spec",
			spec:    nil,
			key:     "enabled",
			wantVal: false,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := ExtractBool(tt.spec, tt.key)
			if gotVal != tt.wantVal || gotOk != tt.wantOk {
				t.Errorf("ExtractBool() = (%v, %v), want (%v, %v)", gotVal, gotOk, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestExtractBoolWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		spec         map[string]any
		key          string
		defaultValue bool
		want         bool
	}{
		{
			name:         "existing true value",
			spec:         map[string]any{"enabled": true},
			key:          "enabled",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "existing false value",
			spec:         map[string]any{"enabled": false},
			key:          "enabled",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "missing key uses default",
			spec:         map[string]any{"enabled": true},
			key:          "missing",
			defaultValue: true,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractBoolWithDefault(tt.spec, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("ExtractBoolWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractInt(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		wantVal int
		wantOk  bool
	}{
		{
			name:    "existing int",
			spec:    map[string]any{"count": 42},
			key:     "count",
			wantVal: 42,
			wantOk:  true,
		},
		{
			name:    "existing int64",
			spec:    map[string]any{"count": int64(42)},
			key:     "count",
			wantVal: 42,
			wantOk:  true,
		},
		{
			name:    "existing float64 with integer value (JSON unmarshal case)",
			spec:    map[string]any{"count": float64(42)},
			key:     "count",
			wantVal: 42,
			wantOk:  true,
		},
		{
			name:    "float64 with decimal value - rejected",
			spec:    map[string]any{"rate": float64(42.5)},
			key:     "rate",
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "missing key",
			spec:    map[string]any{"count": 42},
			key:     "missing",
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "wrong type - string",
			spec:    map[string]any{"name": "42"},
			key:     "name",
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "nil spec",
			spec:    nil,
			key:     "count",
			wantVal: 0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := ExtractInt(tt.spec, tt.key)
			if gotVal != tt.wantVal || gotOk != tt.wantOk {
				t.Errorf("ExtractInt() = (%v, %v), want (%v, %v)", gotVal, gotOk, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestExtractIntWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		spec         map[string]any
		key          string
		defaultValue int
		want         int
	}{
		{
			name:         "existing value",
			spec:         map[string]any{"count": 42},
			key:          "count",
			defaultValue: 10,
			want:         42,
		},
		{
			name:         "missing key uses default",
			spec:         map[string]any{"count": 42},
			key:          "missing",
			defaultValue: 10,
			want:         10,
		},
		{
			name:         "wrong type uses default",
			spec:         map[string]any{"name": "42"},
			key:          "name",
			defaultValue: 10,
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractIntWithDefault(tt.spec, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("ExtractIntWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractMap(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		wantVal map[string]any
		wantOk  bool
	}{
		{
			name:    "existing map",
			spec:    map[string]any{"config": map[string]any{"timeout": 30, "enabled": true}},
			key:     "config",
			wantVal: map[string]any{"timeout": 30, "enabled": true},
			wantOk:  true,
		},
		{
			name:    "empty map",
			spec:    map[string]any{"config": map[string]any{}},
			key:     "config",
			wantVal: map[string]any{},
			wantOk:  true,
		},
		{
			name:    "missing key",
			spec:    map[string]any{"config": map[string]any{"timeout": 30}},
			key:     "missing",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "wrong type - string",
			spec:    map[string]any{"name": "foo"},
			key:     "name",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "nil spec",
			spec:    nil,
			key:     "config",
			wantVal: nil,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := ExtractMap(tt.spec, tt.key)
			if !reflect.DeepEqual(gotVal, tt.wantVal) || gotOk != tt.wantOk {
				t.Errorf("ExtractMap() = (%v, %v), want (%v, %v)", gotVal, gotOk, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestExtractMapWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		spec         map[string]any
		key          string
		defaultValue map[string]any
		want         map[string]any
	}{
		{
			name:         "existing value",
			spec:         map[string]any{"config": map[string]any{"timeout": 30}},
			key:          "config",
			defaultValue: map[string]any{"default": true},
			want:         map[string]any{"timeout": 30},
		},
		{
			name:         "missing key uses default",
			spec:         map[string]any{"config": map[string]any{"timeout": 30}},
			key:          "missing",
			defaultValue: map[string]any{"default": true},
			want:         map[string]any{"default": true},
		},
		{
			name:         "wrong type uses default",
			spec:         map[string]any{"name": "foo"},
			key:          "name",
			defaultValue: map[string]any{"default": true},
			want:         map[string]any{"default": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMapWithDefault(tt.spec, tt.key, tt.defaultValue)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractMapWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequireString(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		want    string
		wantErr bool
	}{
		{
			name:    "existing string",
			spec:    map[string]any{"name": "my-app"},
			key:     "name",
			want:    "my-app",
			wantErr: false,
		},
		{
			name:    "missing key returns error",
			spec:    map[string]any{"name": "my-app"},
			key:     "missing",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type returns error",
			spec:    map[string]any{"count": 42},
			key:     "count",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequireString(tt.spec, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RequireString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequireStringSlice(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		want    []string
		wantErr bool
	}{
		{
			name:    "existing slice",
			spec:    map[string]any{"tags": []string{"a", "b"}},
			key:     "tags",
			want:    []string{"a", "b"},
			wantErr: false,
		},
		{
			name:    "missing key returns error",
			spec:    map[string]any{"tags": []string{"a"}},
			key:     "missing",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "wrong type returns error",
			spec:    map[string]any{"numbers": []int{1, 2}},
			key:     "numbers",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequireStringSlice(tt.spec, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireStringSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RequireStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequireStringMap(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "existing map",
			spec:    map[string]any{"labels": map[string]string{"app": "foo"}},
			key:     "labels",
			want:    map[string]string{"app": "foo"},
			wantErr: false,
		},
		{
			name:    "missing key returns error",
			spec:    map[string]any{"labels": map[string]string{"app": "foo"}},
			key:     "missing",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "wrong type returns error",
			spec:    map[string]any{"name": "foo"},
			key:     "name",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequireStringMap(tt.spec, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireStringMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RequireStringMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequireBool(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		want    bool
		wantErr bool
	}{
		{
			name:    "existing bool",
			spec:    map[string]any{"enabled": true},
			key:     "enabled",
			want:    true,
			wantErr: false,
		},
		{
			name:    "missing key returns error",
			spec:    map[string]any{"enabled": true},
			key:     "missing",
			want:    false,
			wantErr: true,
		},
		{
			name:    "wrong type returns error",
			spec:    map[string]any{"name": "true"},
			key:     "name",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequireBool(tt.spec, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RequireBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequireInt(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		want    int
		wantErr bool
	}{
		{
			name:    "existing int",
			spec:    map[string]any{"count": 42},
			key:     "count",
			want:    42,
			wantErr: false,
		},
		{
			name:    "missing key returns error",
			spec:    map[string]any{"count": 42},
			key:     "missing",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong type returns error",
			spec:    map[string]any{"name": "42"},
			key:     "name",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequireInt(tt.spec, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RequireInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequireMap(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		key     string
		want    map[string]any
		wantErr bool
	}{
		{
			name:    "existing map",
			spec:    map[string]any{"config": map[string]any{"timeout": 30}},
			key:     "config",
			want:    map[string]any{"timeout": 30},
			wantErr: false,
		},
		{
			name:    "missing key returns error",
			spec:    map[string]any{"config": map[string]any{"timeout": 30}},
			key:     "missing",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "wrong type returns error",
			spec:    map[string]any{"name": "foo"},
			key:     "name",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequireMap(tt.spec, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RequireMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
