package env

import (
	"errors"
	"fmt"
	"github.com/sllt/parrot/types"
)

type Env struct {
	Data      map[string]types.ParrotType
	Outer     types.EnvType
	curFunc   types.ParrotType // current func
	pc        int
	addrStack types.ParrotType // stack trace
}

func NewEnv(outer types.EnvType, binds_mt types.ParrotType,
	exprs_mt types.ParrotType) (types.EnvType, error) {
	env := Env{map[string]types.ParrotType{}, outer, nil, 0, nil}

	if binds_mt != nil && exprs_mt != nil {
		binds, e := types.GetSlice(binds_mt)
		if e != nil {
			return nil, e
		}
		exprs, e := types.GetSlice(exprs_mt)
		if e != nil {
			return nil, e
		}

		for i := 0; i < len(binds); i++ {
			if types.Symbol_Q(binds[i]) && binds[i].(types.Symbol).Val == "&" {
				env.Data[binds[i+1].(types.Symbol).Val] = types.List{exprs[i:], nil}
				break
			} else {
				env.Data[binds[i].(types.Symbol).Val] = exprs[i]
			}
		}
	}

	return env, nil
}

func (e Env) Find(key types.Symbol) types.EnvType {
	if _, ok := e.Data[key.Val]; ok {
		return e
	} else if e.Outer != nil {
		return e.Outer.Find(key)
	} else {
		return nil
	}
}

func (e Env) Set(key types.Symbol, value types.ParrotType) types.ParrotType {
	e.Data[key.Val] = value
	return value
}

func (e Env) Get(key types.Symbol) (types.ParrotType, error) {
	env := e.Find(key)
	if env == nil {
		return nil, errors.New("'" + key.Val + "' not found")
	}
	return env.(Env).Data[key.Val], nil
}

func (e Env) All() (types.ParrotType, error) {
	for k, _ := range e.Data {
		fmt.Print(k + " ")
	}
	return nil, nil
}

func (e *Env) GetStackTrace() {

}
