package analysis

import (
	"borm-lsp/lsp"
	"fmt"
	"iter"
	"slices"
	"strings"
)

type Token struct {
	pos lsp.Position
	value string
}

var splitters = []byte{'(', ')', '{', '}', '<', '>', '.', ',', '"', '\'', '`', ';', ' ', '/', ':'}

func Tokenize(text string) []Token {
	tokens := []Token{}
	sb := strings.Builder{}

	// void func test(param string) {} //comment
	for i, line := range strings.Split(text, "\n") {
		for j := 0; j < len(line); j++ {
			if !slices.Contains(splitters, line[j]) { 
				sb.WriteByte(line[j])
				continue
			} 

			if sb.Len() > 0 {
				pos := lsp.Position{Line:i, Character: j-sb.Len()}
				tokens = append(tokens, Token{pos: pos, value: sb.String()})
				sb.Reset()
			}

			if line[j] != ' ' {
				pos := lsp.Position{Line:i, Character: j}
				tokens = append(tokens, Token{pos: pos, value: string(line[j])}) 
			}
		}
		if sb.Len() == 0 {
			continue
		}
		pos := lsp.Position{Line:i, Character: len(line)-sb.Len()}
		tokens = append(tokens, Token{pos: pos, value: sb.String()})
		sb.Reset()
	}
	return tokens
}

func GetTokensToNewLine(tokens []Token) []Token {
	idx := 0
	for idx < len(tokens) && tokens[idx].pos.Line == tokens[0].pos.Line {
		idx++
	}
	return tokens[:idx] 
}

func IterTokensToNewLine(tokens []Token) iter.Seq2[int, Token] {
	line := tokens[0].pos.Line
	return func(yield func(int, Token) bool) {
		for i, v := range tokens {
			if !yield(i, v) {
				return
			}
			if v.pos.Line != line {
				return 
			}
		}
	}
}

func IterTokensToValue(tokens []Token, val string) iter.Seq2[int, Token] {
	return func(yield func(int, Token) bool) {
		for i, v := range tokens {
			if !yield(i, v) {
				return
			}
			if v.value == val {
				return 
			}
		}
	}
}

func IterTokensToAnyValue(tokens []Token, vals... string) iter.Seq2[int, Token] {
	return func(yield func(int, Token) bool) {
		for i, v := range tokens {
			if !yield(i, v) {
				return
			}
			if slices.Contains(vals, v.value) {
				return 
			}
		}
	}
}

func GetFunctionTokens(tokens []Token, declaration string, l int) ([]Token, error) {
	name, params, found := strings.Cut(declaration, "(")
	if !found {
		//is bad - declaration always needs name and brackets
		return tokens, fmt.Errorf("Malformed function.")
	}

	nameParts := strings.Split(name, " ")
	if len(nameParts) < 3 {
		//is bad - should be: [1]type [2]function [3]name(params)
		return tokens, fmt.Errorf("Malformed function.")
	}

	params, rest, found := strings.Cut(params, ")")
	if !found {
		// same bad as first bad
		return tokens, fmt.Errorf("Malformed function.")
	}

	rest = strings.TrimSpace(rest)
	if rest[0] == ';' {
		// it's only a delcaration
		return tokens[:l], nil
	}
	
	if l >= len(tokens)-1 {
		// bad no body and no semi colon
		return tokens, fmt.Errorf("Function declarations must be closed with a semi colon.")
	}
		
	return tokens, nil
}

func GetFunctionBodyTokens(tokens []Token) ([]Token, bool) {
	depth := 0
	inCommentLine := -1
	inBody := false
	inQuotes := false
	inSingles := false
	inAccent := false

	for i := 0; i < len(tokens); i++ {
		if inCommentLine == tokens[i].pos.Line {
			continue
		}
		val := tokens[i].value
		if cIdx := strings.Index(val, "//"); cIdx >= 0 {
			// it's a comment disregard until new line
			inCommentLine = tokens[i].pos.Line
			//process whatever is before the slashes
			val = val[:cIdx]
		}
		for _, char := range val {
			if char == '"' && !inAccent && !inSingles {
				inQuotes = !inQuotes
			}
			if char == '\'' && !inQuotes && !inAccent {
				inSingles = !inSingles
			}
			if char == '`' && !inQuotes && !inSingles {
				inAccent = !inAccent
			}
			if inQuotes || inSingles || inAccent {
				continue
			}
			if char == '{' {
				inBody = true
				depth++
			}
			if char == '}' {
				depth--
			}
			if inBody && depth == 0 {
				return tokens[:i], true
			}
		}
	}
	return tokens, false
}

func Stringify(tokens []Token) string {
	value := strings.Builder{}
	for _, token := range tokens {
		value.WriteString(token.value)
	}
	return value.String()
}

func GetStartPos(tokens... Token) lsp.Position {
	return lsp.Position{
		Line:tokens[0].pos.Line, 
		Character: tokens[0].pos.Character,
	}
}

func GetFinalPos(tokens... Token) lsp.Position {
	idx := len(tokens)-1
	finalPos := lsp.Position {
		Line: tokens[idx].pos.Line,
		Character:tokens[idx].pos.Character + len(tokens[idx].value),
	}
	return finalPos
}
