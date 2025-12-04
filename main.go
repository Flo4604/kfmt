package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	fieldsFlag := flag.String("json-fields", "", "comma-separated list of JSON fields to convert")
	flag.Parse()

	var fields []string
	if *fieldsFlag != "" {
		fields = strings.Split(*fieldsFlag, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
	}

	// Check if we have arguments (direct number conversion)
	args := flag.Args()
	if len(args) > 0 {
		for _, arg := range args {
			result, err := formatValue(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error converting %s: %v\n", arg, err)
				os.Exit(1)
			}
			fmt.Println(result)
		}
		return
	}

	// Check if stdin has data
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  kfmt <number>                            Convert bytes to human readable")
		fmt.Fprintln(os.Stderr, "  <json> | kfmt --json-fields \"a,b\"        Convert specific JSON fields")
		os.Exit(1)
	}

	if len(fields) == 0 {
		fmt.Fprintln(os.Stderr, "error: --json-fields is required when processing JSON")
		os.Exit(1)
	}

	// Process JSON from stdin
	reader := bufio.NewReader(os.Stdin)
	input, err := io.ReadAll(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	output := processJSON(string(input), fields)
	fmt.Print(output)
}

func processJSON(input string, fields []string) string {
	result := input
	for _, field := range fields {
		// Match "fieldName": "value" where value is a quantity.
		// Quantities: integers, decimals, scientific notation, with optional suffix.
		// Valid suffixes: Ki, Mi, Gi, Ti, Pi, Ei (binary) or k, K, M, G, T, P, E (decimal)
		pattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]+)"`, regexp.QuoteMeta(field))
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			submatch := re.FindStringSubmatch(match)
			if len(submatch) < 2 {
				return match
			}
			formatted, err := formatValue(submatch[1])
			if err != nil {
				return match // not a valid quantity, leave unchanged
			}
			return fmt.Sprintf(`"%s": "%s"`, field, formatted)
		})

		// Match "fieldName": 12345 or "fieldName": 1.5e6 (unquoted numbers)
		pattern2 := fmt.Sprintf(`"%s"\s*:\s*(\d+\.?\d*(?:[eE][+-]?\d+)?)([,\s\n\r\}])`, regexp.QuoteMeta(field))
		re2 := regexp.MustCompile(pattern2)
		result = re2.ReplaceAllStringFunc(result, func(match string) string {
			submatch := re2.FindStringSubmatch(match)
			if len(submatch) < 3 {
				return match
			}
			formatted, err := formatValue(submatch[1])
			if err != nil {
				return match
			}
			return fmt.Sprintf(`"%s": "%s"%s`, field, formatted, submatch[2])
		})
	}
	return result
}

// parseQuantity parses a Kubernetes-style quantity string and returns bytes.
// Supports:
//   - Raw numbers: "1000", "1.5"
//   - Scientific notation: "12e6", "1.5e9"
//   - Binary suffixes (IEC): Ki, Mi, Gi, Ti, Pi, Ei
//   - Decimal suffixes (SI): k, K, M, G, T, P, E
func parseQuantity(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// Handle scientific notation first (e.g., "12e6", "1.5e9")
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		// Make sure this is scientific notation, not a suffix like "E" or "Ei"
		// Scientific notation has digits after e/E
		if i+1 < len(s) && (s[i+1] == '-' || s[i+1] == '+' || (s[i+1] >= '0' && s[i+1] <= '9')) {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return 0, err
			}
			if f < 0 {
				return 0, fmt.Errorf("negative value")
			}
			return uint64(f), nil
		}
	}

	// Try suffixes in order: binary first (longer), then decimal
	type suffixDef struct {
		suffix     string
		multiplier float64
	}
	suffixes := []suffixDef{
		// Binary (IEC)
		{"Ki", 1024},
		{"Mi", 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"Pi", 1024 * 1024 * 1024 * 1024 * 1024},
		{"Ei", 1024 * 1024 * 1024 * 1024 * 1024 * 1024},
		// Decimal (SI)
		{"k", 1000},
		{"K", 1000},
		{"M", 1000 * 1000},
		{"G", 1000 * 1000 * 1000},
		{"T", 1000 * 1000 * 1000 * 1000},
		{"P", 1000 * 1000 * 1000 * 1000 * 1000},
		{"E", 1000 * 1000 * 1000 * 1000 * 1000 * 1000},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf.suffix) {
			numStr := strings.TrimSuffix(s, sf.suffix)
			f, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, err
			}
			if f < 0 {
				return 0, fmt.Errorf("negative value")
			}
			return uint64(f * sf.multiplier), nil
		}
	}

	// No suffix, parse as raw bytes
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if f < 0 {
		return 0, fmt.Errorf("negative value")
	}
	return uint64(f), nil
}

func formatValue(s string) (string, error) {
	bytes, err := parseQuantity(s)
	if err != nil {
		return "", err
	}
	return humanizeIEC(bytes), nil
}

func humanizeIEC(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}

	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	exp := 0
	val := float64(bytes)

	for val >= unit && exp < len(units)-1 {
		val /= unit
		exp++
	}

	// Format with appropriate precision
	if val >= 100 {
		return fmt.Sprintf("%.0f%s", val, units[exp])
	} else if val >= 10 {
		return fmt.Sprintf("%.1f%s", val, units[exp])
	}
	return fmt.Sprintf("%.2f%s", val, units[exp])
}
