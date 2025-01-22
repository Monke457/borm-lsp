package analysis

import (
	"borm-lsp/lsp"
	"strings"
)

type Token struct {
	pos lsp.Position
	value string
}

func Tokenize(text string) []Token {
	tokens := []Token{}
	sb := strings.Builder{}

	for i, line := range strings.Split(text, "\n") {
		for j := 0; j < len(line); j++ {
			if line[j] != ' ' {
				sb.WriteByte(line[j])
				continue
			}
			if sb.Len() == 0 {
				continue
			}
			pos := lsp.Position{Line:i, Character: j+1-sb.Len()}
			tokens = append(tokens, Token{pos: pos, value: sb.String()})
			sb.Reset()
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

func Stringify(tokens []Token) string {
	value := strings.Builder{}
	for i, token := range tokens {
		if i > 0 {
			value.WriteByte(' ')
		}
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
