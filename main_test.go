package main

import (
	"testing"
)

func TestHumanizeIEC(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{0, "0B"},
		{1, "1B"},
		{512, "512B"},
		{1023, "1023B"},
		{1024, "1.00KiB"},
		{1536, "1.50KiB"},
		{10240, "10.0KiB"},
		{102400, "100KiB"},
		{1048576, "1.00MiB"},
		{178255984, "170MiB"},
		{293007, "286KiB"},
		{128849018, "123MiB"},
		{1073741824, "1.00GiB"},
		{10737418240, "10.0GiB"},
		{1099511627776, "1.00TiB"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := humanizeIEC(tc.bytes)
			if got != tc.want {
				t.Errorf("humanizeIEC(%d) = %q, want %q", tc.bytes, got, tc.want)
			}
		})
	}
}

func TestParseQuantity(t *testing.T) {
	tests := []struct {
		input   string
		want    uint64
		wantErr bool
	}{
		// Raw bytes
		{"178255984", 178255984, false},
		{"0", 0, false},
		{"1024", 1024, false},

		// Binary suffixes (IEC)
		{"1Ki", 1024, false},
		{"1Mi", 1048576, false},
		{"1Gi", 1073741824, false},
		{"1Ti", 1099511627776, false},
		{"12075408Ki", 12365217792, false},
		{"100Mi", 104857600, false},

		// Decimal suffixes (SI)
		{"1K", 1000, false},
		{"1k", 1000, false},
		{"1M", 1000000, false},
		{"1G", 1000000000, false},
		{"500M", 500000000, false},

		// Errors
		{"", 0, true},
		{"invalid", 0, true},
		{"Ki", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseQuantity(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseQuantity(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("parseQuantity(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		// Raw bytes
		{"178255984", "170MiB", false},
		{"293007", "286KiB", false},
		{"1073741824", "1.00GiB", false},

		// Binary suffixes
		{"12075408Ki", "11.5GiB", false},
		{"100Mi", "100MiB", false},
		{"1Gi", "1.00GiB", false},

		// Decimal suffixes (converted to binary display)
		{"1G", "954MiB", false},
		{"500M", "477MiB", false},

		// Errors
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := formatValue(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("formatValue(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("formatValue(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestProcessJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		fields []string
		want   string
	}{
		{
			name:   "single field quoted",
			input:  `{"usedBytes": "178255984"}`,
			fields: []string{"usedBytes"},
			want:   `{"usedBytes": "170MiB"}`,
		},
		{
			name:   "multiple fields",
			input:  `{"usedBytes": "178255984", "growthRate": "293007"}`,
			fields: []string{"usedBytes", "growthRate"},
			want:   `{"usedBytes": "170MiB", "growthRate": "286KiB"}`,
		},
		{
			name:   "nested object",
			input:  `{"data": {"usedBytes": "178255984"}}`,
			fields: []string{"usedBytes"},
			want:   `{"data": {"usedBytes": "170MiB"}}`,
		},
		{
			name:   "unquoted number",
			input:  `{"usedBytes": 178255984}`,
			fields: []string{"usedBytes"},
			want:   `{"usedBytes": "170MiB"}`,
		},
		{
			name:   "field not in list",
			input:  `{"otherField": "178255984"}`,
			fields: []string{"usedBytes"},
			want:   `{"otherField": "178255984"}`,
		},
		{
			name:   "preserves other fields",
			input:  `{"usedBytes": "178255984", "name": "test", "count": 42}`,
			fields: []string{"usedBytes"},
			want:   `{"usedBytes": "170MiB", "name": "test", "count": 42}`,
		},
		{
			name:   "empty fields list",
			input:  `{"usedBytes": "178255984"}`,
			fields: []string{},
			want:   `{"usedBytes": "178255984"}`,
		},
		{
			name:   "kubernetes quantity Ki suffix",
			input:  `{"spaceAvailable": "12075408Ki"}`,
			fields: []string{"spaceAvailable"},
			want:   `{"spaceAvailable": "11.5GiB"}`,
		},
		{
			name:   "kubernetes quantity Mi suffix",
			input:  `{"capacity": "100Mi"}`,
			fields: []string{"capacity"},
			want:   `{"capacity": "100MiB"}`,
		},
		{
			name:   "kubernetes quantity Gi suffix",
			input:  `{"size": "12Gi"}`,
			fields: []string{"size"},
			want:   `{"size": "12.0GiB"}`,
		},
		{
			name:   "mixed raw and suffixed",
			input:  `{"usedBytes": "178255984", "spaceAvailable": "12075408Ki"}`,
			fields: []string{"usedBytes", "spaceAvailable"},
			want:   `{"usedBytes": "170MiB", "spaceAvailable": "11.5GiB"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := processJSON(tc.input, tc.fields)
			if got != tc.want {
				t.Errorf("processJSON() = %q, want %q", got, tc.want)
			}
		})
	}
}
