/*
 This file was autogenerated via
 -------------------------------------------
 ldetool generate --go-string pseudo-pre.lde
 -------------------------------------------
 do not touch it with bare hands!
*/

package semver

import (
	"fmt"
	"strconv"
	"strings"
)

var preMinus = "pre-"

// pseudo ...
type pseudo struct {
	Rest   string
	Base   string
	Moment uint64
	SHA    string
}

// Extract ...
func (p *pseudo) Extract(line string) (bool, error) {
	p.Rest = line
	var err error
	var pos int
	var rest1 string
	var tmp string
	var tmpUint uint64

	// Take until '-' as Base(string)
	pos = strings.IndexByte(p.Rest, '-')
	if pos >= 0 {
		p.Base = p.Rest[:pos]
		p.Rest = p.Rest[pos+1:]
	} else {
		return false, nil
	}
	rest1 = p.Rest

	// Checks if the rest starts with `"pre-"` and pass it
	if strings.HasPrefix(rest1, preMinus) {
		rest1 = rest1[len(preMinus):]
	} else {
		goto pseudopreAnonymousAreaLabel
	}
	p.Rest = rest1
pseudopreAnonymousAreaLabel:

	// Take until 15th character if it is'-' as Moment(uint64)
	if len(p.Rest) >= 14+1 && p.Rest[14] == '-' {
		pos = 14
	} else {
		pos = -1
	}
	if pos >= 0 {
		tmp = p.Rest[:pos]
		p.Rest = p.Rest[pos+1:]
	} else {
		return false, nil
	}
	if tmpUint, err = strconv.ParseUint(tmp, 10, 64); err != nil {
		return false, fmt.Errorf("cannot parse `%s` into field Moment(uint64): %s", tmp, err)
	}
	p.Moment = uint64(tmpUint)
	if len(p.Rest) <= 6 {
		return false, nil
	}

	// Take the rest as SHA(string)
	p.SHA = p.Rest
	p.Rest = p.Rest[len(p.Rest):]
	return true, nil
}
