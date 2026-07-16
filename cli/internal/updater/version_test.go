package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{"v1.2.3", "1.2.3", 0},
		{"v1.2.4", "v1.2.3", 1},
		{"v2.0.0", "v1.99.99", 1},
		{"v1.2.3-beta.2", "v1.2.3-beta.11", -1},
		{"v1.2.3", "v1.2.3-rc.1", 1},
		{"v1.2.3+build.2", "v1.2.3+build.1", 0},
	}
	for _, test := range tests {
		test := test
		t.Run(test.left+"_"+test.right, func(t *testing.T) {
			t.Parallel()
			got, err := compareVersions(test.left, test.right)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestParseVersionRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"dev", "1.2", "1.02.3", "1.2.3-", "1.2.3-01"} {
		if _, err := parseVersion(value); err == nil {
			t.Errorf("parseVersion(%q) succeeded, want error", value)
		}
	}
}
