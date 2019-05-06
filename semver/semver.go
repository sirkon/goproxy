package semver

import (
	"strconv"

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
		return 0
	}
		res, err := strconv.Atoi(lit)
	if err != nil {
		return -1
	}
	return int(res)
}

// Compare exposes outside semver.Compare from internal package
func Compare(x, y string) int {
	return semver.Compare(x, y)
}