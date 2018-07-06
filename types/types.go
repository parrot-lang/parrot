package types

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type ParrotError struct {
	Obj ParrotType
}

// base types
type ParrotType interface{}

type Channel struct {
	Val chan ParrotType
}

type Bool struct {
	Val bool
}

type Float64 struct {
	Val float64
}

type Int64 struct {
	Val int64
}

type String struct {
	Val string
}

type Symbol struct {
	Val string
}

type List struct {
	Val  []ParrotType
	Meta ParrotType
}

func (p ParrotError) Error() string {
	return fmt.Sprintf("%#v", p.Obj)
}

type EnvType interface {
	Find(key Symbol) EnvType
	Set(key Symbol, value ParrotType) ParrotType
	Get(key Symbol) (ParrotType, error)
	All() (ParrotType, error)
}

func Nil_Q(obj ParrotType) bool {
	return obj == nil
}

func True_Q(obj ParrotType) bool {
	b, ok := obj.(bool)
	return ok && b == true
}

func False_Q(obj ParrotType) bool {
	b, ok := obj.(bool)
	return ok && b == false
}

func Number_Q(obj ParrotType) bool {
	_, ok := obj.(int)
	return ok
}

func Symbol_Q(obj ParrotType) bool {
	_, ok := obj.(Symbol)
	return ok
}

// keywords
func NewKeyword(s string) (ParrotType, error) {
	return "\u029e" + s, nil
}

func Keyword_Q(obj ParrotType) bool {
	s, ok := obj.(string)
	return ok && strings.HasPrefix(s, "\u029e")
}

// strings
func String_Q(obj ParrotType) bool {
	_, ok := obj.(string)
	return ok
}

// Functions
type Func struct {
	Fn          func([]ParrotType) (ParrotType, error)
	Meta        ParrotType
	IsGoroutine bool
}

func Func_Q(obj ParrotType) bool {
	_, ok := obj.(Func)
	return ok
}

type ParrotFunc struct {
	Eval        func(ParrotType, EnvType) (ParrotType, error)
	Exp         ParrotType
	Env         EnvType
	Params      ParrotType
	IsMacro     bool
	GenEnv      func(EnvType, ParrotType, ParrotType) (EnvType, error)
	Meta        ParrotType
	IsGoroutine bool
}

type Goroutine struct {
	ParrotFunc
	IsGoroutine bool
}

func ParrotFunc_Q(obj ParrotType) bool {
	_, ok := obj.(ParrotFunc)
	return ok
}

func (f ParrotFunc) SetMacro() ParrotType {
	f.IsMacro = true
	return f
}

func (f ParrotFunc) GetMacro() bool {
	return f.IsMacro
}

// Take either a MalFunc or regular function and apply it to the
// arguments
func Apply(f_mt ParrotType, a []ParrotType, isGoroutine bool) (ParrotType, error) {
	switch f := f_mt.(type) {
	case ParrotFunc:
		env, e := f.GenEnv(f.Env, f.Params, List{a, nil})
		if e != nil {
			return nil, e
		}
		if isGoroutine {
			go f.Eval(f.Exp, env)
			return nil, nil
		}
		return f.Eval(f.Exp, env)
	case Func:
		if isGoroutine {
			go f.Fn(a)
			return nil, nil
		}
		return f.Fn(a)
	case func([]ParrotType) (ParrotType, error):
		return f(a)
	default:
		return nil, errors.New("Invalid function to Apply")
	}
}

func NewList(a ...ParrotType) ParrotType {
	return List{a, nil}
}

func List_Q(obj ParrotType) bool {
	_, ok := obj.(List)
	return ok
}

type Vector struct {
	Val  []ParrotType
	Meta ParrotType
}

func Vector_Q(obj ParrotType) bool {
	_, ok := obj.(Vector)
	return ok
}

func GetSlice(seq ParrotType) ([]ParrotType, error) {
	switch obj := seq.(type) {
	case List:
		return obj.Val, nil
	case Vector:
		return obj.Val, nil
	default:
		return nil, errors.New("GetSlice called on non-sequence")
	}
}

type HashMap struct {
	Val  map[string]ParrotType
	Meta ParrotType
}

func NewHashMap(seq ParrotType) (ParrotType, error) {
	lst, e := GetSlice(seq)
	if e != nil {
		return nil, e
	}

	if len(lst)%2 == 1 {
		return nil, errors.New("Odd number of arguments to NewHashMap")
	}
	m := map[string]ParrotType{}
	for i := 0; i < len(lst); i += 2 {
		str, ok := lst[i].(string)
		if !ok {
			return nil, errors.New("expected hash-map key string")
		}
		m[str] = lst[i+1]
	}
	return HashMap{m, nil}, nil
}

func HashMap_Q(obj ParrotType) bool {
	_, ok := obj.(HashMap)
	return ok
}

type Atom struct {
	Val  ParrotType
	Meta ParrotType
}

func (a *Atom) Set(val ParrotType) ParrotType {
	a.Val = val
	return a
}

func Atom_Q(obj ParrotType) bool {
	_, ok := obj.(*Atom)
	return ok
}

func _obj_type(obj ParrotType) string {
	if obj == nil {
		return "nil"
	}
	return reflect.TypeOf(obj).Name()
}

func Sequential_Q(seq ParrotType) bool {
	if seq == nil {
		return false
	}
	return (reflect.TypeOf(seq).Name() == "List") ||
		(reflect.TypeOf(seq).Name() == "Vector")
}
func Equal_Q(a ParrotType, b ParrotType) bool {
	ota := reflect.TypeOf(a)
	otb := reflect.TypeOf(b)

	if !((ota == otb) || (Sequential_Q(a) && Sequential_Q(b))) {
		return false
	}

	switch a.(type) {
	case Symbol:
		return a.(Symbol).Val == b.(Symbol).Val
	case List:
		as, _ := GetSlice(a)
		bs, _ := GetSlice(b)
		if len(as) != len(bs) {
			return false
		}
		for i := 0; i < len(as); i++ {
			if !Equal_Q(as[i], bs[i]) {
				return false
			}
		}
		return true
	case Vector:
		as, _ := GetSlice(a)
		bs, _ := GetSlice(b)
		if len(as) != len(bs) {
			return false
		}
		for i := 0; i < len(as); i++ {
			if !Equal_Q(as[i], bs[i]) {
				return false
			}
		}
		return true
	case HashMap:
		am := a.(HashMap).Val
		bm := a.(HashMap).Val
		if len(am) != len(bm) {
			return false
		}
		for k, v := range am {
			if !Equal_Q(v, bm[k]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
