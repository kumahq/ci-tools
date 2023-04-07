package version

import (
	"fmt"
	"strconv"
	"strings"
)

func SplitSemVer(in string) (int, int, int, error) {
	sp := strings.Split(in, ".")
	if len(sp) < 3 {
		return 0, 0, 0, fmt.Errorf("%s is not a valid semver", in)
	}
	major, err := strconv.Atoi(sp[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("major part of semver is not a number")
	}
	minor, err := strconv.Atoi(sp[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("minor part of semver is not a number")
	}
	patch, err := strconv.Atoi(strings.Split(sp[2], "-")[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("patch part of semver is not a number")
	}
	return major, minor, patch, nil
}

func MustSplitSemVer(in string) (int, int, int) {
	major, minor, patch, err := SplitSemVer(in)
	if err != nil {
		panic(err)
	}
	return major, minor, patch
}
