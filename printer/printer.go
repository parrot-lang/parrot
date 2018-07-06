package printer

import (
	"fmt"
	"github.com/sllt/parrot/types"
	"strconv"
	"strings"
)

func PrintList(lst []types.ParrotType, pr bool,
	start string, end string, join string) string {
	strList := make([]string, 0, len(lst))
	for _, e := range lst {
		strList = append(strList, PrintStr(e, pr))
	}
	return start + strings.Join(strList, join) + end
}

func PrintStr(obj types.ParrotType, print_readably bool) string {
	switch tobj := obj.(type) {
	case types.List:
		return PrintList(tobj.Val, print_readably, "(", ")", " ")
	case types.Vector:
		return PrintList(tobj.Val, print_readably, "[", "]", " ")
	case types.HashMap:
		strList := make([]string, 0, len(tobj.Val)*2)
		for k, v := range tobj.Val {
			strList = append(strList, PrintStr(k, print_readably))
			strList = append(strList, PrintStr(v, print_readably))
		}
		return "{" + strings.Join(strList, " ") + "}"
	case string:
		if strings.HasPrefix(tobj, "\u029e") {
			return ":" + tobj[2:len(tobj)]
		} else if print_readably {
			return `"` + strings.Replace(
				strings.Replace(
					strings.Replace(tobj, `\`, `\\`, -1),
					`"`, `\"`, -1),
				"\n", `\n`, -1) + `"`
		} else {
			return tobj
		}
	case types.Symbol:
		return tobj.Val
	case types.Float64:
		return fmt.Sprintf("%v", tobj.Val)
	case types.Int64:
		return strconv.FormatInt(tobj.Val, 10)
	case types.Bool:
		return strconv.FormatBool(tobj.Val)
	case nil:
		return "nil"
	case types.ParrotFunc:
		return "(fn " +
			PrintStr(tobj.Params, true) + " " +
			PrintStr(tobj.Exp, true) + ")"
	case func([]types.ParrotType) (types.ParrotType, error):
		return fmt.Sprintf("<function %v>", obj)
	case *types.Atom:
		return "(atom " +
			PrintStr(tobj.Val, true) + ")"
	default:
		return fmt.Sprintf("%v", obj)
	}
}
