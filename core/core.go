package core

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	// "reflect"
	"strings"
	"time"

	"github.com/sllt/parrot/printer"
	"github.com/sllt/parrot/reader"
	"github.com/sllt/parrot/readline"
	"github.com/sllt/parrot/types"
)

type NumericOp int

const (
	Add NumericOp = iota
	Sub
	Mult
	Div
	Pow
)

func FloatNumericDo(op NumericOp, a, b types.Float64) types.ParrotType {
	switch op {
	case Add:
		return types.Float64{a.Val + b.Val}
	case Sub:
		return types.Float64{a.Val - b.Val}
	case Mult:
		return types.Float64{a.Val * b.Val}
	case Div:
		return types.Float64{a.Val / b.Val}
	case Pow:
		return types.Float64{math.Pow(a.Val, b.Val)}
	}
	return nil
}

func IntegerNumericDo(op NumericOp, a, b types.Int64) types.ParrotType {
	switch op {
	case Add:
		return types.Int64{a.Val + b.Val}
	case Sub:
		return types.Int64{a.Val - b.Val}
	case Mult:
		return types.Int64{a.Val * b.Val}
	case Div:
		return types.Int64{a.Val / b.Val}
	case Pow:
		return types.Int64{int64(math.Pow(float64(a.Val), float64(b.Val)))}
	}
	return nil
}

func NumericMatchInteger(op NumericOp, a types.Int64, b types.ParrotType) (types.ParrotType, error) {
	switch tb := b.(type) {
	case types.Float64:
		return FloatNumericDo(op, types.Float64{float64(a.Val)}, tb), nil
	case types.Int64:
		return IntegerNumericDo(op, a, tb), nil
	}
	return nil, nil
}

func NumericMatchFloat(op NumericOp, a types.Float64, b types.ParrotType) (types.ParrotType, error) {
	var fb types.Float64
	switch tb := b.(type) {
	case types.Float64:
		fb = tb
	case types.Int64:
		fb = types.Float64{float64(b.(types.Int64).Val)}
	}
	return FloatNumericDo(op, a, fb), nil
}

func NumericDo(op NumericOp, a, b types.ParrotType) (types.ParrotType, error) {
	switch ta := a.(type) {
	case types.Float64:
		return NumericMatchFloat(op, ta, b)
	case types.Int64:
		return NumericMatchInteger(op, ta, b)
	}
	return nil, nil
}

func NumericFunction(op NumericOp, args []types.ParrotType) (types.ParrotType, error) {
	accum := args[0]
	var err error
	for _, v := range args[1:] {
		accum, err = NumericDo(op, accum, v)
		if err != nil {
			// fmt.Println(err)
			return nil, nil
		}
	}
	// switch ra := accum.(type) {
	// case types.Int64:
	// 	return ra.Val, nil
	// case types.Float64:
	// 	return ra.Val, nil
	// default:
	// 	return nil, nil
	// }
	return accum, nil
}
func signumFloat(f float64) int {
	if f > 0 {
		return 1
	}
	if f < 0 {
		return -1
	}
	return 0
}
func signumInt(f int64) int {
	if f > 0 {
		return 1
	}
	if f < 0 {
		return -1
	}
	return 0
}

func compareInt(a types.Int64, b types.ParrotType) (int, error) {
	switch bt := b.(type) {
	case types.Int64:
		return signumInt(a.Val - bt.Val), nil
	case types.Float64:
		return signumFloat(float64(a.Val) - bt.Val), nil
	}
	msg := fmt.Sprintf("cannot compare %T to %T", a, b)
	return 0, errors.New(msg)
}

func compareFloat(a types.Float64, b types.ParrotType) (int, error) {
	switch bt := b.(type) {
	case types.Int64:
		return signumFloat(a.Val - float64(bt.Val)), nil
	case types.Float64:
		nanCount := 0
		if math.IsNaN(a.Val) {
			nanCount++
		}
		if math.IsNaN(bt.Val) {
			nanCount++
		}
		if nanCount > 0 {
			return 1 + nanCount, nil
		}
		return signumFloat(a.Val - bt.Val), nil
	}
	msg := fmt.Sprintf("cannot compare %T to %T", a, b)
	return 0, errors.New(msg)

}

func compareBool(a types.Bool, b types.ParrotType) (int, error) {
	msg := fmt.Sprintf("cannot compare %T to %T", a, b)
	return 0, errors.New(msg)
}

func Compare(a types.ParrotType, b types.ParrotType) (int, error) {
	switch at := a.(type) {
	case types.Int64:
		return compareInt(at, b)
	case types.Float64:
		return compareFloat(at, b)
	case types.Bool:
		return compareBool(at, b)
	}

	msg := fmt.Sprintf("cannot compare %T to %T", a, b)
	return 0, errors.New(msg)
}

func CompareFunction(name string, args []types.ParrotType) (types.ParrotType, error) {
	if len(args) != 2 {
		return nil, errors.New("requires 2 args")
	}

	res, err := Compare(args[0], args[1])
	if err != nil {
		return nil, err
	}

	if res > 1 {
		if name == "!=" {
			return types.Bool{true}, nil
		}
		return types.Bool{false}, nil
	}

	cond := false
	switch name {
	case "<":
		cond = res < 0
	case ">":
		cond = res > 0
	case "<=":
		cond = res <= 0
	case ">=":
		cond = res >= 0
	case "=":
		cond = res == 0
	case "!=":
		cond = res != 0
	}

	return cond, nil
}

func GoroutineFunction(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) < 2 {
		return nil, errors.New("apply requires at least 2 args")
	}
	f := a[0]
	args := []types.ParrotType{}
	for _, b := range a[1 : len(a)-1] {
		args = append(args, b)
	}
	last, e := types.GetSlice(a[len(a)-1])
	if e != nil {
		return nil, e
	}
	args = append(args, last...)
	return types.Apply(f, args, true)
}

func MakeChanFunction(args []types.ParrotType) (types.ParrotType, error) {
	if len(args) > 1 {
		return nil, errors.New("required 1 args")
	}
	size := 0
	if len(args) == 1 {
		switch t := args[0].(type) {
		case types.Int64:
			size = int(t.Val)
		default:
			return nil, errors.New("argment must be int")
		}
	}
	return types.Channel{make(chan types.ParrotType, size)}, nil
}

func ChanFunction(name string, args []types.ParrotType) (types.ParrotType, error) {
	if len(args) < 1 {
		return nil, errors.New("argment error")
	}
	var channel chan types.ParrotType
	switch t := args[0].(type) {
	case types.Channel:
		channel = t.Val
	default:
		return nil, errors.New(fmt.Sprintf("argument 0 of %s must be channel", args[0]))
	}
	if name == "send" {
		if len(args) != 2 {
			return nil, errors.New("argment error (2)")
		}
		channel <- args[1]
		return nil, nil
	}
	return <-channel, nil
}

func CloseChanFunction(args []types.ParrotType) (types.ParrotType, error) {
	if len(args) < 1 {
		return nil, errors.New("argment error")
	}
	switch t := args[0].(type) {
	case types.Channel:
		close(t.Val)
		return nil, nil
	}
	return nil, errors.New(fmt.Sprintf("argument 0 of %s must be channel", args[0]))
}

// Exceptions
func throw(a []types.ParrotType) (types.ParrotType, error) {
	return nil, types.ParrotError{a[0]}
}

func fn_q(a []types.ParrotType) (types.ParrotType, error) {
	switch f := a[0].(type) {
	case types.ParrotFunc:
		return !f.GetMacro(), nil
	case types.Func:
		return true, nil
	case func([]types.ParrotType) (types.ParrotType, error):
		return true, nil
	default:
		return false, nil
	}
}

// tuple

func cons(a []types.ParrotType) (types.ParrotType, error) {
	val := a[0]
	lst, e := types.GetSlice(a[1])
	if e != nil {
		return nil, e
	}
	return types.List{append([]types.ParrotType{val}, lst...), nil}, nil
}

func concat(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) == 0 {
		return types.List{}, nil
	}
	slc1, e := types.GetSlice(a[0])
	if e != nil {
		return nil, e
	}
	for i := 1; i < len(a); i++ {
		slc2, e := types.GetSlice(a[i])
		if e != nil {
			return nil, e
		}
		slc1 = append(slc1, slc2...)
	}
	return types.List{slc1, nil}, nil
}

func empty_Q(a []types.ParrotType) (types.ParrotType, error) {
	switch obj := a[0].(type) {
	case types.List:
		return len(obj.Val) == 0, nil
	case types.Vector:
		return len(obj.Val) == 0, nil
	case nil:
		return true, nil
	default:
		return nil, errors.New("count called on non-sequence")
	}
}

func count(a []types.ParrotType) (types.ParrotType, error) {
	switch obj := a[0].(type) {
	case types.List:
		return len(obj.Val), nil
	case types.Vector:
		return len(obj.Val), nil
	case map[string]types.ParrotType:
		return len(obj), nil
	case nil:
		return 0, nil
	default:
		return nil, errors.New("count called on non-sequence")
	}
}

func nth(a []types.ParrotType) (types.ParrotType, error) {
	slc, e := types.GetSlice(a[0])
	if e != nil {
		return nil, e
	}
	idx := a[1].(int)
	if idx < len(slc) {
		return slc[idx], nil
	} else {
		return nil, errors.New("nth: index out of range")
	}
}

func first(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) == 0 {
		return nil, nil
	}
	if a[0] == nil {
		return nil, nil
	}
	slc, e := types.GetSlice(a[0])
	if e != nil {
		return nil, e
	}
	if len(slc) == 0 {
		return nil, nil
	}
	return slc[0], nil
}

func rest(a []types.ParrotType) (types.ParrotType, error) {
	if a[0] == nil {
		return types.List{}, nil
	}
	slc, e := types.GetSlice(a[0])
	if e != nil {
		return nil, e
	}
	if len(slc) == 0 {
		return types.List{}, nil
	}
	return types.List{slc[1:], nil}, nil
}

func apply(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) < 2 {
		return nil, errors.New("apply requires at least 2 args")
	}
	f := a[0]
	args := []types.ParrotType{}
	for _, b := range a[1 : len(a)-1] {
		args = append(args, b)
	}
	last, e := types.GetSlice(a[len(a)-1])
	if e != nil {
		return nil, e
	}
	args = append(args, last...)
	return types.Apply(f, args, false)
}

func do_map(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) != 2 {
		return nil, errors.New("map requires 2 args")
	}
	f := a[0]
	results := []types.ParrotType{}
	args, e := types.GetSlice(a[1])
	if e != nil {
		return nil, e
	}
	for _, arg := range args {
		res, e := types.Apply(f, []types.ParrotType{arg}, false)
		results = append(results, res)
		if e != nil {
			return nil, e
		}
	}
	return types.List{results, nil}, nil
}

func conj(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) < 2 {
		return nil, errors.New("conj requires at least 2 arguments")
	}
	switch seq := a[0].(type) {
	case types.List:
		new_slc := []types.ParrotType{}
		for i := len(a) - 1; i > 0; i -= 1 {
			new_slc = append(new_slc, a[i])
		}
		return types.List{append(new_slc, seq.Val...), nil}, nil
	case types.Vector:
		new_slc := seq.Val
		for _, x := range a[1:] {
			new_slc = append(new_slc, x)
		}
		return types.Vector{new_slc, nil}, nil
	}

	if !types.HashMap_Q(a[0]) {
		return nil, errors.New("dissoc called on non-hash map")
	}
	new_hm := copyHashMap(a[0].(types.HashMap))
	for i := 1; i < len(a); i += 1 {
		key := a[i]
		if !types.String_Q(key) {
			return nil, errors.New("dissoc called with non-string key")
		}
		delete(new_hm.Val, key.(string))
	}
	return new_hm, nil
}

func seq(a []types.ParrotType) (types.ParrotType, error) {
	if a[0] == nil {
		return nil, nil
	}
	switch arg := a[0].(type) {
	case types.List:
		if len(arg.Val) == 0 {
			return nil, nil
		}
		return arg, nil
	case types.Vector:
		if len(arg.Val) == 0 {
			return nil, nil
		}
		return types.List{arg.Val, nil}, nil
	case string:
		if len(arg) == 0 {
			return nil, nil
		}
		new_slc := []types.ParrotType{}
		for _, ch := range strings.Split(arg, "") {
			new_slc = append(new_slc, ch)
		}
		return types.List{new_slc, nil}, nil
	}
	return nil, errors.New("seq requires string or list or vector or nil")
}

// String
func pr_str(a []types.ParrotType) (types.ParrotType, error) {
	return printer.PrintList(a, true, "", "", " "), nil
}

func prn(a []types.ParrotType) (types.ParrotType, error) {
	fmt.Println(printer.PrintList(a, true, "", "", " "))
	return nil, nil
}

func str(a []types.ParrotType) (types.ParrotType, error) {
	return printer.PrintList(a, false, "", "", ""), nil
}

func println(a []types.ParrotType) (types.ParrotType, error) {
	fmt.Println(printer.PrintList(a, false, "", "", ""))
	return nil, nil
}

func slurp(a []types.ParrotType) (types.ParrotType, error) {
	b, e := ioutil.ReadFile(a[0].(string))
	if e != nil {
		return nil, e
	}
	return string(b), nil
}

// time
func time_ms(a []types.ParrotType) (types.ParrotType, error) {
	return int(time.Now().UnixNano()), nil
}

// hashmap
func copyHashMap(hm types.HashMap) types.HashMap {
	new_hm := types.HashMap{map[string]types.ParrotType{}, nil}
	for k, v := range hm.Val {
		new_hm.Val[k] = v
	}
	return new_hm
}

func assoc(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) < 3 {
		return nil, errors.New("assoc requires at least 3 arguments")
	}
	if len(a)%2 != 1 {
		return nil, errors.New("assoc requires odd number of arguments")
	}
	if !types.HashMap_Q(a[0]) {
		return nil, errors.New("assoc called on non-hash map")
	}
	new_hm := copyHashMap(a[0].(types.HashMap))
	for i := 1; i < len(a); i += 2 {
		key := a[i]
		if !types.String_Q(key) {
			return nil, errors.New("assoc called with non-string key")
		}
		new_hm.Val[key.(string)] = a[i+1]
	}
	return new_hm, nil
}

func dissoc(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) < 2 {
		return nil, errors.New("dissoc requires at least 3 arguments")
	}
	if !types.HashMap_Q(a[0]) {
		return nil, errors.New("dissoc called on non-hash map")
	}
	new_hm := copyHashMap(a[0].(types.HashMap))
	for i := 1; i < len(a); i += 1 {
		key := a[i]
		if !types.String_Q(key) {
			return nil, errors.New("dissoc called with non-string key")
		}
		delete(new_hm.Val, key.(string))
	}
	return new_hm, nil
}

func get(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) != 2 {
		return nil, errors.New("get requires 2 arguments")
	}
	if types.Nil_Q(a[0]) {
		return nil, nil
	}
	if !types.HashMap_Q(a[0]) {
		return nil, errors.New("get called on non-hash map")
	}
	if !types.String_Q(a[1]) {
		return nil, errors.New("get called with non-string key")
	}
	return a[0].(types.HashMap).Val[a[1].(string)], nil
}

func update(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) != 3 {
		return nil, errors.New("get requires 3 arguments")
	}
	if types.Nil_Q(a[0]) {
		return nil, nil
	}
	if !types.HashMap_Q(a[0]) {
		return nil, errors.New("get called on non-hash map")
	}
	if !types.String_Q(a[1]) {
		return nil, errors.New("get called with non-string key")
	}
	a[0].(types.HashMap).Val[a[1].(string)] = a[2].(types.ParrotType)
	return a[0].(types.HashMap), nil
}

func contains_Q(hm types.ParrotType, key types.ParrotType) (types.ParrotType, error) {
	if types.Nil_Q(hm) {
		return false, nil
	}
	if !types.HashMap_Q(hm) {
		return nil, errors.New("get called on non-hash map")
	}
	if !types.String_Q(key) {
		return nil, errors.New("get called with non-string key")
	}
	_, ok := hm.(types.HashMap).Val[key.(string)]
	return ok, nil
}

func keys(a []types.ParrotType) (types.ParrotType, error) {
	if !types.HashMap_Q(a[0]) {
		return nil, errors.New("keys called on non-hash map")
	}
	slc := []types.ParrotType{}
	for k, _ := range a[0].(types.HashMap).Val {
		slc = append(slc, k)
	}
	return types.List{slc, nil}, nil
}
func vals(a []types.ParrotType) (types.ParrotType, error) {
	if !types.HashMap_Q(a[0]) {
		return nil, errors.New("keys called on non-hash map")
	}
	slc := []types.ParrotType{}
	for _, v := range a[0].(types.HashMap).Val {
		slc = append(slc, v)
	}
	return types.List{slc, nil}, nil
}

// meta
func with_meta(a []types.ParrotType) (types.ParrotType, error) {
	if len(a) != 2 {
		return nil, errors.New("with-meta requires 2 args")
	}
	obj := a[0]
	m := a[1]
	switch tobj := obj.(type) {
	case types.List:
		return types.List{tobj.Val, m}, nil
	case types.Vector:
		return types.Vector{tobj.Val, m}, nil
	case types.HashMap:
		return types.HashMap{tobj.Val, m}, nil
	case types.Func:
		return types.Func{tobj.Fn, m, false}, nil
	case types.ParrotFunc:
		fn := tobj
		fn.Meta = m
		return fn, nil
	default:
		return nil, errors.New("with-meta not supported on type")
	}
}

func meta(a []types.ParrotType) (types.ParrotType, error) {
	obj := a[0]
	switch tobj := obj.(type) {
	case types.List:
		return tobj.Meta, nil
	case types.Vector:
		return tobj.Meta, nil
	case types.HashMap:
		return tobj.Meta, nil
	case types.Func:
		return tobj.Meta, nil
	case types.ParrotFunc:
		return tobj.Meta, nil
	default:
		return nil, errors.New("meta not supported on type")
	}
}

// atom
func deref(a []types.ParrotType) (types.ParrotType, error) {
	if !types.Atom_Q(a[0]) {
		return nil, errors.New("deref called with non-atom")
	}
	return a[0].(*types.Atom).Val, nil
}

func reset_BANG(a []types.ParrotType) (types.ParrotType, error) {
	if !types.Atom_Q(a[0]) {
		return nil, errors.New("reset! called with non-atom")
	}
	a[0].(*types.Atom).Set(a[1])
	return a[1], nil
}

func swap_BANG(a []types.ParrotType) (types.ParrotType, error) {
	if !types.Atom_Q(a[0]) {
		return nil, errors.New("swap! called with non-atom")
	}
	if len(a) < 2 {
		return nil, errors.New("swap! requires at least 2 args")
	}
	atm := a[0].(*types.Atom)
	args := []types.ParrotType{atm.Val}
	f := a[1]
	args = append(args, a[2:]...)
	res, e := types.Apply(f, args, false)
	if e != nil {
		return nil, e
	}
	atm.Set(res)
	return res, nil
}

var NS = map[string]types.ParrotType{
	// exception
	"throw": throw,
	// check
	"nil?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Nil_Q(a[0]), nil
	},
	"true?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.True_Q(a[0]), nil
	},
	"symbol?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Symbol_Q(a[0]), nil
	},
	"string?": func(a []types.ParrotType) (types.ParrotType, error) {
		return (types.String_Q(a[0]) && !types.Keyword_Q(a[0])), nil
	},
	"keyword?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Keyword_Q(a[0]), nil
	},
	"number?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Number_Q(a[0]), nil
	},
	"fn?": fn_q,
	"macro?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.ParrotFunc_Q(a[0]) && a[0].(types.ParrotFunc).GetMacro(), nil
	},
	// type
	"symbol": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Symbol{a[0].(string)}, nil
	},
	"keyword": func(a []types.ParrotType) (types.ParrotType, error) {
		if types.Keyword_Q(a[0]) {
			return a[0], nil
		} else {
			return types.NewKeyword(a[0].(string))
		}
	},

	//string
	"str": func(a []types.ParrotType) (types.ParrotType, error) {
		return str(a)
	},
	"pr-str": func(a []types.ParrotType) (types.ParrotType, error) {
		return pr_str(a)
	},
	"println": func(a []types.ParrotType) (types.ParrotType, error) {
		return println(a)
	},
	"prn": func(a []types.ParrotType) (types.ParrotType, error) {
		return prn(a)
	},
	"read-string": func(a []types.ParrotType) (types.ParrotType, error) {
		return reader.ReadStr(a[0].(string))
	},
	"readline": func(a []types.ParrotType) (types.ParrotType, error) {
		return readline.Readline(a[0].(string))
	},
	"slurp": slurp,
	"string-split": func(a []types.ParrotType) (types.ParrotType, error) {
		arr := strings.Split(a[0].(string), a[1].(string))
		new_arr := []types.ParrotType{}
		for _, v := range arr {
			new_arr = append(new_arr, v)
		}
		return types.List{new_arr, nil}, nil
	},
	// op
	"=": func(a []types.ParrotType) (types.ParrotType, error) {
		return CompareFunction("=", a)
	},

	"<": func(a []types.ParrotType) (types.ParrotType, error) {
		return CompareFunction("<", a)
	},
	">": func(a []types.ParrotType) (types.ParrotType, error) {
		return CompareFunction(">", a)
	},
	">=": func(a []types.ParrotType) (types.ParrotType, error) {
		return CompareFunction(">=", a)
	},
	"<=": func(a []types.ParrotType) (types.ParrotType, error) {
		return CompareFunction("<=", a)
	},
	"+": func(a []types.ParrotType) (types.ParrotType, error) {
		return NumericFunction(Add, a)
	},
	"-": func(a []types.ParrotType) (types.ParrotType, error) {
		return NumericFunction(Sub, a)
	},
	"*": func(a []types.ParrotType) (types.ParrotType, error) {
		return NumericFunction(Mult, a)
	},
	"/": func(a []types.ParrotType) (types.ParrotType, error) {
		return NumericFunction(Div, a)
	},
	"time-ms": time_ms,

	// tuple
	"count": func(a []types.ParrotType) (types.ParrotType, error) {
		return count(a)
	},
	"empty?": func(a []types.ParrotType) (types.ParrotType, error) {
		return empty_Q(a)
	},
	"list": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.List{a, nil}, nil
	},
	"list?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.List_Q(a[0]), nil
	},
	"vector": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Vector{a, nil}, nil
	},
	"vector?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Vector_Q(a[0]), nil
	},
	"hash-map": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.NewHashMap(types.List{a, nil})
	},
	"map?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.HashMap_Q(a[0]), nil
	},
	"assoc":  assoc,
	"dissoc": dissoc,
	"get":    get,
	"update": update,
	"contains?": func(a []types.ParrotType) (types.ParrotType, error) {
		return contains_Q(a[0], a[1])
	},
	"keys": keys,
	"vals": vals,

	"sequential?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Sequential_Q(a[0]), nil
	},
	"cons":   cons,
	"concat": concat,
	"nth":    nth,
	"first":  first,
	"rest":   rest,
	"apply":  apply,
	"map":    do_map,
	"conj":   conj,
	"seq":    seq,

	"with-meta": with_meta,
	"meta":      meta,
	"atom": func(a []types.ParrotType) (types.ParrotType, error) {
		return &types.Atom{a[0], nil}, nil
	},
	"atom?": func(a []types.ParrotType) (types.ParrotType, error) {
		return types.Atom_Q(a[0]), nil
	},
	"deref":  deref,
	"reset!": reset_BANG,
	"swap!":  swap_BANG,

	"sleep": func(a []types.ParrotType) (types.ParrotType, error) {
		time.Sleep(time.Duration(a[0].(types.Int64).Val) * time.Millisecond)
		return nil, nil
	},
	"go": func(a []types.ParrotType) (types.ParrotType, error) {
		return GoroutineFunction(a)
	},
	"makeChan": func(a []types.ParrotType) (types.ParrotType, error) {
		return MakeChanFunction(a)
	},
	"closeChan": func(a []types.ParrotType) (types.ParrotType, error) {
		return CloseChanFunction(a)
	},
	"send": func(a []types.ParrotType) (types.ParrotType, error) {
		return ChanFunction("send", a)
	},
	"receive": func(a []types.ParrotType) (types.ParrotType, error) {
		return ChanFunction("receive", a)
	},
	"system": func(a []types.ParrotType) (types.ParrotType, error) {
		return SystemFunction(a)
	},
	"exit": func(a []types.ParrotType) (types.ParrotType, error) {
		fmt.Println("Bye !")
		os.Exit(0)
		return nil, nil
	},
}
