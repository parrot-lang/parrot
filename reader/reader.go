package reader

import (
	"errors"
	// "fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sllt/parrot/types"
)

type Reader interface {
	next() *string
	peek() *string
}

type TokenReader struct {
	tokens   []string
	position int
}

func (t *TokenReader) next() *string {
	if t.position >= len(t.tokens) {
		return nil
	}
	token := t.tokens[t.position]
	t.position = t.position + 1
	return &token
}

func (t *TokenReader) peek() *string {
	if t.position >= len(t.tokens) {
		return nil
	}
	return &t.tokens[t.position]
}

func tokenize(str string) []string {
	results := make([]string, 0, 1)
	re := regexp.MustCompile(`[\s,]*(~@|[\[\]{}()'` + "`" +
		`~^@]|"(?:\\.|[^\\"])*"|;.*|[^\s\[\]{}('"` + "`" +
		`,;)]*)` + `|[-+]?([0-9]*\.[0-9]+|[0-9]+)`)
	for _, group := range re.FindAllStringSubmatch(str, -1) {
		if (group[1] == "") || (group[1][0] == ';') {
			continue
		}
		results = append(results, group[1])
	}
	return results
}

// func readFloat(rdr Reader) (types.ParrotType, error) {
// 	token := rdr.peek()
// 	if token == nil {
// 		return nil, errors.New("readFloat underflow")
// 	}
// 	if math, _ := regexp.MatchString(`[-+]?[0-9]*\.?[0-9]+`, *token); math {
// 		var f float64
// 		var e error
// 		if f, e = strconv.ParseFloat(*token, 64); e != nil {
// 			fmt.Println(e)
// 			return nil, nil
// 		}
// 		fmt.Println(f)
// 	}
// 	return nil, nil
// }

func readAtom(rdr Reader) (types.ParrotType, error) {
	token := rdr.next()
	if token == nil {
		return nil, errors.New("readAtom underflow")
	}
	if match, _ := regexp.MatchString(`^[-+]?[0-9]*\.?[0-9]+$`, *token); match {

		// parse int64 number
		i, e := strconv.Atoi(*token)
		if e == nil {
			return types.Int64{int64(i)}, nil
		} else {
			// parse float64 number

			f, e := strconv.ParseFloat(*token, 64)
			if e == nil {
				return types.Float64{f}, nil
			}
		}
		return nil, errors.New("number parse error")
	} else if (*token)[0] == '"' {
		str := (*token)[1 : len(*token)-1]
		val := strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(str, `\\`, "\u029e", -1),
					`\"`, `"`, -1),
				`\n`, "\n", -1),
			"\u029e", "\\", -1)

		return val, nil
	} else if (*token)[0] == ':' {
		return types.NewKeyword((*token)[1:len(*token)])
	} else if *token == "nil" {
		return nil, nil
	} else if *token == "true" {
		return true, nil
	} else if *token == "false" {
		return false, nil
	} else {
		return types.Symbol{*token}, nil
	}
	return token, nil
}

func readList(rdr Reader, start string, end string) (types.ParrotType, error) {
	token := rdr.next()
	if token == nil {
		return nil, errors.New("readList unferflow")
	}
	if *token != start {
		return nil, errors.New("expected '" + start + "'")
	}
	astList := []types.ParrotType{}
	token = rdr.peek()
	for ; true; token = rdr.peek() {
		if token == nil {
			return nil, errors.New("exepected '" + end + "', got EOF")
		}
		if *token == end {
			break
		}
		f, e := readForm(rdr)
		if e != nil {
			return nil, e
		}
		astList = append(astList, f)
	}
	rdr.next()
	return types.List{astList, nil}, nil
}

func readVector(rdr Reader) (types.ParrotType, error) {
	lst, e := readList(rdr, "[", "]")
	if e != nil {
		return nil, e
	}
	vec := types.Vector{lst.(types.List).Val, nil}
	return vec, nil
}

func readHashMap(rdr Reader) (types.ParrotType, error) {
	lst, e := readList(rdr, "{", "}")
	if e != nil {
		return nil, e
	}
	return types.NewHashMap(lst)
}

func readForm(rdr Reader) (types.ParrotType, error) {
	token := rdr.peek()
	if token == nil {
		return nil, errors.New("readForm underflow")
	}

	switch *token {
	case `'`:
		rdr.next()
		form, e := readForm(rdr)
		if e != nil {
			return nil, e
		}
		return types.List{[]types.ParrotType{types.Symbol{"quote"}, form}, nil}, nil
	case "`":
		rdr.next()
		form, e := readForm(rdr)
		if e != nil {
			return nil, e
		}
		return types.List{[]types.ParrotType{types.Symbol{"quasiquote"}, form}, nil}, nil
	case `~`:
		rdr.next()
		form, e := readForm(rdr)
		if e != nil {
			return nil, e
		}
		return types.List{[]types.ParrotType{types.Symbol{"unquote"}, form}, nil}, nil
	case `~@`:
		rdr.next()
		form, e := readForm(rdr)
		if e != nil {
			return nil, e
		}
		return types.List{[]types.ParrotType{types.Symbol{"splice-unquote"}, form}, nil}, nil
	case `^`:
		rdr.next()
		meta, e := readForm(rdr)
		if e != nil {
			return nil, e
		}
		form, e := readForm(rdr)
		if e != nil {
			return nil, e
		}
		return types.List{[]types.ParrotType{types.Symbol{"with-meta"}, form, meta}, nil}, nil
	case `@`:
		rdr.next()
		form, e := readForm(rdr)
		if e != nil {
			return nil, e
		}
		return types.List{[]types.ParrotType{types.Symbol{"deref"}, form}, nil}, nil
	case ")":
		return nil, errors.New("unexpected ')'")
	case "(":
		return readList(rdr, "(", ")")
	case "]":
		return nil, errors.New("unexpected ']'")
	case "[":
		return readVector(rdr)
	case "}":
		return nil, errors.New("unexpected '}'")
	case "{":
		return readHashMap(rdr)
	default:
		return readAtom(rdr)
	}
	return readAtom(rdr)
}

func ReadStr(str string) (types.ParrotType, error) {
	tokens := tokenize(str)
	if len(tokens) == 0 {
		return nil, errors.New("<empty line>")
	}

	return readForm(&TokenReader{tokens: tokens, position: 0})
}
