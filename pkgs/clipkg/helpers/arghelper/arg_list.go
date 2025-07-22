package arghelper

import (
	"fmt"
	"strconv"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////

type TwitterListIdsArg []uint64

func (l *TwitterListIdsArg) Set(str string) error {
	vals := strings.Split(str, ",")

	for _, v := range vals {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid list ID: %s", v)
		}
		*l = append(*l, id)
	}

	return nil
}

func (l *TwitterListIdsArg) String() string {
	return fmt.Sprintf("%v", *l)
}
