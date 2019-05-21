package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirkon/goproxy/internal/semver"
)

// IsValid exposes outside semver.IsValid from internal package
func IsValid(sample string) bool {
	return semver.IsValid(sample)
}

// Major returns major component of semver string. Returns -1 if major component turned to be not a number
func Major(sample string) int {
	lit := semver.Major(sample)
	if len(lit) == 0 {
		return -1
	}
	res, err := strconv.Atoi(lit[1:])
	if err != nil {
		return -1
	}
	return int(res)
}

// Compare exposes outside semver.Compare from internal package
func Compare(x, y string) int {
	return semver.Compare(x, y)
}

var extractor = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+).*`)

// MajorMinorPatch returns three main components of given semver
func MajorMinorPatch(version string) (int, int, int) {
	if !extractor.MatchString(version) {
		return 0, 0, 0
	}

	res := extractor.FindStringSubmatch(version)
	if len(res) != 4 {
		return 0, 0, 0
	}

	nums := make([]int, 3)
	for i, lit := range res[1:4] {
		value, err := strconv.Atoi(lit)
		if err != nil {
			return 0, 0, 0
		}
		nums[i] = value
	}

	return nums[0], nums[1], nums[2]
}

// Base base returns base version part of given version string
func Base(v string) string {
	major, minor, patch := MajorMinorPatch(v)
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

// Pseudo determines if given version v looks like a pseudo-version. Returns SHA if it does.
func Pseudo(v string) string {
	var p pseudo
	ok, _ := p.Extract(v)
	if !ok {
		return ""
	}
	return p.SHA
}

// PseudoParts returns parts of pseudo version. All parts are empty if it is not pseudo-version
func PseudoParts(v string) (base string, moment string, sha string) {
	var p pseudo
	ok, _ := p.Extract(v)
	if !ok {
		return "", "", ""
	}
	return p.Base, fmt.Sprintf("%d", p.Moment), p.SHA
}

// Max return maximal semver between two
func Max(v, w string) string {
	return semver.Max(v, w)
}

// Canonical returns canonical semver from some version-like input string
func Canonical(v string) string {
	return semver.Canonical(v)
}

// IsPrerelease if this version is pre-released in our custom sense: it should have form vX.Y.Z-pre-....
func IsPrerelease(v string) bool {
	build := v[len(Base(v)):]
	return strings.HasPrefix(build, "-pre-")
}
