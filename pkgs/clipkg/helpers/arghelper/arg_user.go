package arghelper

import (
	"fmt"
	"strconv"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////

type UserTwitterIdsArg []uint64

func (u *UserTwitterIdsArg) Set(str string) error {
	vals := strings.Split(str, ",")

	for _, v := range vals {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %s", v)
		}
		*u = append(*u, id)
	}

	return nil
}

func (u *UserTwitterIdsArg) String() string {
	return fmt.Sprintf("%v", *u)
}

////////////////////////////////////////////////////////////////////////////////

type UserTwitterScreenNamesArg []string

func (u *UserTwitterScreenNamesArg) Set(str string) error {
	vals := strings.Split(str, ",")

	for _, v := range vals {
		str, _ = strings.CutPrefix(v, "@")
		if str == "" {
			return fmt.Errorf("invalid user screen name: %s", v)
		}
		*u = append(*u, str)
	}

	return nil
}

func (u *UserTwitterScreenNamesArg) String() string {
	return fmt.Sprintf("%v", *u)
}
