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

		// Scientific notation (e-notation)
		{"1e3", 1000, false},
		{"12e6", 12000000, false},
		{"1.5e9", 1500000000, false},
		{"1.0e6", 1000000, false},
		{"2.5e3", 2500, false},

		// Decimal values with binary suffixes
		{"1.5Ki", 1536, false},
		{"1.5Mi", 1572864, false},
		{"1.5Gi", 1610612736, false},
		{"2.5Gi", 2684354560, false},
		{"0.5Mi", 524288, false},

		// Decimal values with decimal suffixes
		{"1.5k", 1500, false},
		{"1.5M", 1500000, false},
		{"2.5G", 2500000000, false},

		// Higher unit suffixes (Ti, Pi, Ei)
		{"1Pi", 1125899906842624, false},
		{"1.5Ti", 1649267441664, false},
		{"1T", 1000000000000, false},
		{"1P", 1000000000000000, false},

		// Scientific notation edge cases
		{"1e+3", 1000, false},
		{"1.0e+6", 1000000, false},

		// Raw decimal bytes (no suffix)
		{"1.5", 1, false},  // truncates to 1 byte
		{"100.9", 100, false},

		// Whitespace handling
		{" 1Ki ", 1024, false},
		{"  1024  ", 1024, false},

		// Very small values (truncate to 0)
		{"1e-3", 0, false},    // 0.001 truncates to 0 bytes
		{"0.001", 0, false},

		// Errors
		{"", 0, true},
		{"invalid", 0, true},
		{"Ki", 0, true},
		{"-1", 0, true},       // negative value
		{"-100Mi", 0, true},   // negative with suffix
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

		// Scientific notation (e-notation)
		{"1.5e9", "1.40GiB", false},
		{"12e6", "11.4MiB", false},
		{"1e3", "1000B", false},

		// Decimal values with binary suffixes
		{"1.5Gi", "1.50GiB", false},
		{"1.5Mi", "1.50MiB", false},
		{"1.5Ki", "1.50KiB", false},
		{"2.5Gi", "2.50GiB", false},

		// Decimal values with decimal suffixes
		{"1.5M", "1.43MiB", false},
		{"2.5G", "2.33GiB", false},

		// Higher unit suffixes
		{"1Ti", "1.00TiB", false},
		{"1.5Ti", "1.50TiB", false},

		// Raw decimal bytes
		{"1.5", "1B", false},
		{"100.9", "100B", false},

		// Errors
		{"invalid", "", true},
		{"", "", true},
		{"-1", "", true},
		{"-100Mi", "", true},
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
		{
			name:   "scientific notation quoted",
			input:  `{"size": "1.5e9"}`,
			fields: []string{"size"},
			want:   `{"size": "1.40GiB"}`,
		},
		{
			name:   "scientific notation unquoted",
			input:  `{"size": 1.5e9}`,
			fields: []string{"size"},
			want:   `{"size": "1.40GiB"}`,
		},
		{
			name:   "decimal with binary suffix",
			input:  `{"capacity": "1.5Gi"}`,
			fields: []string{"capacity"},
			want:   `{"capacity": "1.50GiB"}`,
		},
		{
			name:   "decimal with decimal suffix",
			input:  `{"rate": "1.5M"}`,
			fields: []string{"rate"},
			want:   `{"rate": "1.43MiB"}`,
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
