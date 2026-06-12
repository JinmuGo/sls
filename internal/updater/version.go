package updater

import (
	"strconv"
	"strings"
)

// compareVersions compares two semantic version strings and returns -1 if a < b,
// 0 if a == b, and 1 if a > b. A leading "v" is ignored. Missing components are
// treated as zero (so "1.1" == "1.1.0"). A pre-release suffix (e.g. "-rc1") sorts
// lower than the corresponding release; two pre-releases are compared lexically.
//
// It is intentionally dependency-free — sls uses simple vX.Y.Z tags and the
// project values having zero external dependencies for core logic.
func compareVersions(a, b string) int {
	aCore, aPre := splitVersion(a)
	bCore, bPre := splitVersion(b)

	if c := compareCore(aCore, bCore); c != 0 {
		return c
	}

	// Cores are equal — a release outranks a pre-release.
	switch {
	case aPre == "" && bPre == "":
		return 0
	case aPre == "":
		return 1
	case bPre == "":
		return -1
	}
	return comparePrerelease(aPre, bPre)
}

// comparePrerelease compares two non-empty pre-release suffixes. Identifiers are
// split on "." and compared in order; a suffix with fewer identifiers ranks lower
// when it is a prefix of the other. Each identifier is compared with natural
// ordering so that, in addition to strict semver dotted numerics ("alpha.2" <
// "alpha.10"), the common un-dotted style also sorts intuitively ("rc9" < "rc10").
func comparePrerelease(a, b string) int {
	aIDs := strings.Split(a, ".")
	bIDs := strings.Split(b, ".")
	n := max(len(aIDs), len(bIDs))

	for i := range n {
		// A shorter pre-release (a prefix of the other) has lower precedence.
		if i >= len(aIDs) {
			return -1
		}
		if i >= len(bIDs) {
			return 1
		}
		if c := compareIdentifier(aIDs[i], bIDs[i]); c != 0 {
			return c
		}
	}
	return 0
}

// compareIdentifier compares two pre-release identifiers using natural ordering:
// maximal digit and non-digit runs are compared pairwise, digit runs numerically
// and non-digit runs lexically. A digit run ranks lower than a non-digit run, and
// the shorter identifier ranks lower when it is a prefix of the other.
func compareIdentifier(a, b string) int {
	ai, bi := 0, 0
	for ai < len(a) && bi < len(b) {
		aRun, aDigit, aNext := nextRun(a, ai)
		bRun, bDigit, bNext := nextRun(b, bi)

		switch {
		case aDigit && bDigit:
			if c := compareNumericRun(aRun, bRun); c != 0 {
				return c
			}
		case aDigit != bDigit:
			if aDigit { // numeric runs rank lower than alphabetic
				return -1
			}
			return 1
		default:
			if c := strings.Compare(aRun, bRun); c != 0 {
				return c
			}
		}
		ai, bi = aNext, bNext
	}
	switch {
	case ai < len(a):
		return 1
	case bi < len(b):
		return -1
	}
	return 0
}

// nextRun returns the maximal run of same-class (digit / non-digit) characters
// starting at i, whether that run is digits, and the index past it.
func nextRun(s string, i int) (run string, isDigit bool, next int) {
	isDigit = s[i] >= '0' && s[i] <= '9'
	j := i
	for j < len(s) && (s[j] >= '0' && s[j] <= '9') == isDigit {
		j++
	}
	return s[i:j], isDigit, j
}

// compareNumericRun compares two digit runs by numeric value, tolerating leading
// zeros and arbitrarily large numbers (length-then-lexical avoids overflow).
func compareNumericRun(a, b string) int {
	a = strings.TrimLeft(a, "0")
	b = strings.TrimLeft(b, "0")
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

// splitVersion strips a leading "v" and splits the numeric core from an optional
// pre-release suffix introduced by "-".
func splitVersion(v string) (core, pre string) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	core, pre, _ = strings.Cut(v, "-")
	return core, pre
}

// compareCore compares two dotted numeric version cores component by component.
func compareCore(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	n := max(len(aParts), len(bParts))

	for i := range n {
		av := numAt(aParts, i)
		bv := numAt(bParts, i)
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// numAt returns the integer at index i, or 0 if out of range / unparseable.
func numAt(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n, err := strconv.Atoi(parts[i])
	if err != nil {
		return 0
	}
	return n
}
