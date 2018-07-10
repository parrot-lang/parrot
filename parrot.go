package parrot

import (
	"errors"
	"io/ioutil"
	// "fmt"
	// "os"
	// "strings"
)

import (
	"github.com/sllt/parrot/core"
	. "github.com/sllt/parrot/env"
	"github.com/sllt/parrot/printer"
	"github.com/sllt/parrot/reader"
	// "github.com/sllt/parrot/readline"
	. "github.com/sllt/parrot/types"
)

// read
func Read(str string) (ParrotType, error) {
	return reader.ReadStr(str)
}

func isPair(x ParrotType) bool {
	slc, e := GetSlice(x)
	if e != nil {
		return false
	}
	return len(slc) > 0
}

func quasiquote(ast ParrotType) ParrotType {
	if !isPair(ast) {
		return List{[]ParrotType{Symbol{"quote"}, ast}, nil}
	} else {
		slc, _ := GetSlice(ast)
		a0 := slc[0]
		if Symbol_Q(a0) && (a0.(Symbol).Val == "unquote") {
			return slc[1]
		} else if isPair(a0) {
			slc0, _ := GetSlice(a0)
			a00 := slc0[0]
			if Symbol_Q(a00) && (a00.(Symbol).Val == "splice-unquote") {
				return List{[]ParrotType{Symbol{"concat"},
					slc0[1],
					quasiquote(List{slc[1:], nil})}, nil}
			}
		}
		return List{[]ParrotType{Symbol{"cons"},
			quasiquote(a0),
			quasiquote(List{slc[1:], nil})}, nil}
	}
}

func isMacroCall(ast ParrotType, env EnvType) bool {
	if List_Q(ast) {
		slc, _ := GetSlice(ast)
		if len(slc) == 0 {
			return false
		}
		a0 := slc[0]
		if Symbol_Q(a0) && env.Find(a0.(Symbol)) != nil {
			mac, e := env.Get(a0.(Symbol))
			if e != nil {
				return false
			}
			if ParrotFunc_Q(mac) {
				return mac.(ParrotFunc).GetMacro()
			}
		}
	}
	return false
}

func macroexpand(ast ParrotType, env EnvType) (ParrotType, error) {
	var mac ParrotType
	var e error

	for isMacroCall(ast, env) {
		slc, _ := GetSlice(ast)
		a0 := slc[0]
		mac, e = env.Get(a0.(Symbol))
		if e != nil {
			return nil, e
		}
		fn := mac.(ParrotFunc)
		ast, e = Apply(fn, slc[1:], false)
		if e != nil {
			return nil, e
		}
	}
	return ast, nil
}

// func DoGoroutine(ast ParrotType, env EnvType) (ParrotType, error) {
// 	slc, _ := GetSlice(ast)
// 	a0 := slc[0]
// 	fn_env, e := env.Get(a0.(Symbol))
// 	if e != nil {
// 		return nil, e
// 	}
// 	switch ft := fn_env.(type) {
// 	case ParrotFunc:
// 	case Func:
// 		ft.IsGoroutine = true
// 		ast, e = Apply(ft, slc[1:], false)
// 		if e != nil {
// 			return nil, e
// 		}
// 		return ast, nil
// 	}
// 	// fn := fn_env.(ParrotFunc)
// 	// fn.IsGoroutine = true
// 	// ast, e = Apply(fn, slc[1:])
// 	// if e != nil {
// 	// 	return nil, e
// 	// }
// 	return nil, nil
// }

func evalAst(ast ParrotType, env EnvType) (ParrotType, error) {
	if Symbol_Q(ast) {
		return env.Get(ast.(Symbol))
	} else if List_Q(ast) {
		lst := []ParrotType{}
		for _, a := range ast.(List).Val {
			exp, e := Eval(a, env)
			if e != nil {
				return nil, e
			}
			lst = append(lst, exp)
		}
		return List{lst, nil}, nil
	} else if Vector_Q(ast) {
		lst := []ParrotType{}
		for _, a := range ast.(Vector).Val {
			exp, e := Eval(a, env)
			if e != nil {
				return nil, e
			}
			lst = append(lst, exp)
		}
		return Vector{lst, nil}, nil
	} else if HashMap_Q(ast) {
		m := ast.(HashMap)
		new_hm := HashMap{map[string]ParrotType{}, nil}
		for k, v := range m.Val {
			ke, e1 := Eval(k, env)
			if e1 != nil {
				return nil, e1
			}
			if _, ok := ke.(string); !ok {
				return nil, errors.New("non string hash-map key")
			}
			kv, e2 := Eval(v, env)
			if e2 != nil {
				return nil, e2
			}
			new_hm.Val[ke.(string)] = kv
		}
		return new_hm, nil
	} else {
		return ast, nil
	}
}

func Eval(ast ParrotType, env EnvType) (ParrotType, error) {
	var e error
	for {

		switch ast.(type) {
		case List: // continue
		default:
			return evalAst(ast, env)
		}

		// apply list
		ast, e = macroexpand(ast, env)
		if e != nil {
			return nil, e
		}
		if !List_Q(ast) {
			return evalAst(ast, env)
		}
		if len(ast.(List).Val) == 0 {
			return ast, nil
		}

		a0 := ast.(List).Val[0]
		var a1 ParrotType = nil
		var a2 ParrotType = nil
		switch len(ast.(List).Val) {
		case 1:
			a1 = nil
			a2 = nil
		case 2:
			a1 = ast.(List).Val[1]
			a2 = nil
		default:
			a1 = ast.(List).Val[1]
			a2 = ast.(List).Val[2]
		}
		a0sym := "__<*fn*>__"
		if Symbol_Q(a0) {
			a0sym = a0.(Symbol).Val
		}
		switch a0sym {
		case "def":
			res, e := Eval(a2, env)
			if e != nil {
				return nil, e
			}
			return env.Set(a1.(Symbol), res), nil
		case "let":
			let_env, e := NewEnv(env, nil, nil)
			if e != nil {
				return nil, e
			}
			arr1, e := GetSlice(a1)
			if e != nil {
				return nil, e
			}
			for i := 0; i < len(arr1); i += 2 {
				if !Symbol_Q(arr1[i]) {
					return nil, errors.New("non-symbol bind value")
				}
				exp, e := Eval(arr1[i+1], let_env)
				if e != nil {
					return nil, e
				}
				let_env.Set(arr1[i].(Symbol), exp)
			}
			ast = a2
			env = let_env
		case "quote":
			return a1, nil
		case "quasiquote":
			ast = quasiquote(a1)
		case "defmacro":
			fn, e := Eval(a2, env)
			fn = fn.(ParrotFunc).SetMacro()
			if e != nil {
				return nil, e
			}
			return env.Set(a1.(Symbol), fn), nil
		case "macroexpand":
			return macroexpand(a1, env)
		case "try":
			var exc ParrotType
			exp, e := Eval(a1, env)
			if e == nil {
				return exp, nil
			} else {
				if a2 != nil && List_Q(a2) {
					a2s, _ := GetSlice(a2)
					if Symbol_Q(a2s[0]) && (a2s[0].(Symbol).Val == "catch") {
						switch e.(type) {
						case ParrotError:
							exc = e.(ParrotError).Obj
						default:
							exc = e.Error()
						}
						binds := NewList(a2s[1])
						new_env, e := NewEnv(env, binds, NewList(exc))
						if e != nil {
							return nil, e
						}
						exp, e = Eval(a2s[2], new_env)
						if e == nil {
							return exp, nil
						}
					}
				}
				return nil, e
			}
		case "do":
			lst := ast.(List).Val
			_, e := evalAst(List{lst[1 : len(lst)-1], nil}, env)
			if e != nil {
				return nil, e
			}
			if len(lst) == 1 {
				return nil, nil
			}
			ast = lst[len(lst)-1]
		case "if":
			cond, e := Eval(a1, env)
			if e != nil {
				return nil, e
			}
			if cond == nil || cond == false {
				if len(ast.(List).Val) >= 4 {
					ast = ast.(List).Val[3]
				} else {
					return nil, nil
				}
			} else {
				ast = a2
			}
		case "fn":
			fn := ParrotFunc{Eval, a2, env, a1, false, NewEnv, nil, false}
			return fn, nil
		default:
			el, e := evalAst(ast, env)
			if e != nil {
				return nil, e
			}
			f := el.(List).Val[0]
			if ParrotFunc_Q(f) {
				fn := f.(ParrotFunc)
				ast = fn.Exp
				env, e = NewEnv(fn.Env, fn.Params, List{el.(List).Val[1:], nil})
				if e != nil {
					return nil, e
				}
			} else {
				fn, ok := f.(Func)
				if !ok {
					return nil, errors.New("attempt to call non-function")
				}
				return fn.Fn(el.(List).Val[1:])
			}
		}

	} // TCO loop
}

// print
func Print(exp ParrotType) (string, error) {
	return printer.PrintStr(exp, true), nil
}

func Run(str string) (ParrotType, error) {
	b, e := ioutil.ReadFile(str)
	if e != nil {
		return nil, e
	}
	return Rep(string(b))
}

// repl
func Rep(str string) (ParrotType, error) {
	var exp ParrotType
	var res string
	var e error
	if exp, e = Read(str); e != nil {
		return nil, e
	}
	if exp, e = Eval(exp, Repl_env); e != nil {
		return nil, e
	}
	if res, e = Print(exp); e != nil {
		return nil, e
	}
	return res, nil
}

var Repl_env, _ = NewEnv(nil, nil, nil)

func init() {
	for k, v := range core.NS {
		Repl_env.Set(Symbol{k}, Func{v.(func([]ParrotType) (ParrotType, error)), nil, false})
	}
	Repl_env.Set(Symbol{"eval"}, Func{func(a []ParrotType) (ParrotType, error) {
		return Eval(a[0], Repl_env)
	}, nil, false})
	Repl_env.Set(Symbol{"*ARGV*"}, List{})

	Rep("(def *host-language* \"go\")")
	Rep("(def not (fn (a) (if a false true)))")
	Rep("(def load-file (fn (f) (eval (read-string (str \"(do \" (slurp f) \")\")))))")
	Rep("(defmacro cond (fn (& xs) (if (> (count xs) 0) (list 'if (first xs) (if (> (count xs) 1) (nth xs 1) (throw \"odd number of forms to cond\")) (cons 'cond (rest (rest xs)))))))")
	Rep("(def *gensym-counter* (atom 0))")
	Rep("(def gensym (fn [] (symbol (str \"G__\" (swap! *gensym-counter* (fn* [x] (+ 1 x)))))))")
	Rep("(defmacro or (fn (& xs) (if (empty? xs) nil (if (= 1 (count xs)) (first xs) (let (condvar (gensym)) `(let (~condvar ~(first xs)) (if ~condvar ~condvar (or ~@(rest xs)))))))))")
	Rep("(defmacro defn (fn [name args body] `(def ~name (fn ~args ~body))))")
	Rep("(defn curry [func args] (fn [arg] (apply func (cons args (list arg)))))")
}
