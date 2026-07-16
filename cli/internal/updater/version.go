package updater

import (
	"fmt"
	"strconv"
	"strings"
)

type semanticVersion struct {
	major      uint64
	minor      uint64
	patch      uint64
	prerelease []string
}

func parseVersion(value string) (semanticVersion, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	if buildIndex := strings.IndexByte(value, '+'); buildIndex >= 0 {
		value = value[:buildIndex]
	}

	core := value
	var prerelease []string
	if prereleaseIndex := strings.IndexByte(value, '-'); prereleaseIndex >= 0 {
		core = value[:prereleaseIndex]
		pre := value[prereleaseIndex+1:]
		if pre == "" {
			return semanticVersion{}, fmt.Errorf("invalid semantic version %q", value)
		}
		prerelease = strings.Split(pre, ".")
	}

	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semanticVersion{}, fmt.Errorf("invalid semantic version %q", value)
	}
	values := make([]uint64, 3)
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return semanticVersion{}, fmt.Errorf("invalid semantic version %q", value)
		}
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return semanticVersion{}, fmt.Errorf("invalid semantic version %q: %w", value, err)
		}
		values[i] = parsed
	}

	for _, identifier := range prerelease {
		if identifier == "" {
			return semanticVersion{}, fmt.Errorf("invalid semantic version %q", value)
		}
		for _, r := range identifier {
			if !(r >= '0' && r <= '9') && !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && r != '-' {
				return semanticVersion{}, fmt.Errorf("invalid semantic version %q", value)
			}
		}
		if isNumeric(identifier) && len(identifier) > 1 && identifier[0] == '0' {
			return semanticVersion{}, fmt.Errorf("invalid semantic version %q", value)
		}
	}

	return semanticVersion{major: values[0], minor: values[1], patch: values[2], prerelease: prerelease}, nil
}

func compareVersions(left, right string) (int, error) {
	l, err := parseVersion(left)
	if err != nil {
		return 0, err
	}
	r, err := parseVersion(right)
	if err != nil {
		return 0, err
	}

	if result := compareUint64(l.major, r.major); result != 0 {
		return result, nil
	}
	if result := compareUint64(l.minor, r.minor); result != 0 {
		return result, nil
	}
	if result := compareUint64(l.patch, r.patch); result != 0 {
		return result, nil
	}
	return comparePrerelease(l.prerelease, r.prerelease), nil
}

func compareUint64(left, right uint64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func comparePrerelease(left, right []string) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}

	for i := 0; i < len(left) && i < len(right); i++ {
		if left[i] == right[i] {
			continue
		}
		leftNumeric := isNumeric(left[i])
		rightNumeric := isNumeric(right[i])
		switch {
		case leftNumeric && rightNumeric:
			if len(left[i]) < len(right[i]) {
				return -1
			}
			if len(left[i]) > len(right[i]) {
				return 1
			}
			if left[i] < right[i] {
				return -1
			}
			return 1
		case leftNumeric:
			return -1
		case rightNumeric:
			return 1
		case left[i] < right[i]:
			return -1
		default:
			return 1
		}
	}

	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
