package filter

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// newScanner allocates a scanner from an io.Reader
func newScanner(r io.Reader) *scanner {
	return &scanner{r: bufio.NewReader(r)}
}

// scanner tokenizes an input stream
// TODO(b5): set position properly for errors
type scanner struct {
	r *bufio.Reader

	// scanning state
	tok               token
	text              strings.Builder
	line, col, offset int
	err               error
}

// Scan reads one token from the input stream
func (s *scanner) Scan() token {
	s.text.Reset()

	for {
		ch := s.read()

		switch ch {
		case eof:
			return s.newTok(tEOF)
		// ignore whitespace
		case '\r', ' ':
			continue

		case '|':
			return s.newTok(tPipe)
		case '[':
			return s.newTok(tLeftBracket)
		case ']':
			return s.newTok(tRightBracket)
		case '(':
			return s.newTok(tLeftParen)
		case ')':
			return s.newTok(tRightParen)
		case '{':
			return s.newTok(tLeftBrace)
		case '}':
			return s.newTok(tRightBrace)
		case ':':
			return s.newTok(tColon)
		case '.':
			if p, err := s.r.Peek(1); err == nil {
				if isNumericByte(p[0]) {
					return s.scanNumber()
				}
			}
			return s.newTok(tDot)
		case ',':
			return s.newTok(tComma)

		case '+':
			return s.newTok(tPlus)
		case '-':
			if p, err := s.r.Peek(1); err == nil {
				if isNumericByte(p[0]) {
					return s.scanNumber()
				}
			}
			return s.newTok(tMinus)
		case '*':
			return s.newTok(tStar)
		case '/':
			return s.newTok(tForwardSlash)

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			s.unread()
			return s.scanNumber()
		case '"':
			return s.scanQuotedText()
		default:
			s.text.WriteRune(ch)
			return s.scanLiteral()
		}
	}
}

// read reads the next rune from the buffered reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (s *scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

func (s *scanner) unread() error {
	return s.r.UnreadRune()
}

// newTok creates a new token from current scanner state
func (s *scanner) newTok(t tokenType) token {
	return token{
		Type: t,
		Text: strings.TrimSpace(s.text.String()),
		Pos:  position{Line: s.line, Col: s.col, Offset: s.offset},
	}
}

func (s *scanner) newTextTok() token {
	return token{
		Type: tText,
		Text: strings.TrimSpace(s.text.String()),
		Pos:  position{Line: s.line, Col: s.col, Offset: s.offset},
	}
}

var literalMatch = regexp.MustCompile(`[\w\n_\-]`)

func (s *scanner) scanLiteral() token {
	for {
		ch := s.read()
		if literalMatch.MatchString(string(ch)) {
			s.text.WriteRune(ch)
		} else {
			s.unread()
			return s.newTextTok()
		}
	}
}

func (s *scanner) scanQuotedText() token {
	for {
		ch := s.read()
		switch ch {
		default:
			s.text.WriteRune(ch)
		case '"', eof:
			return s.newTextTok()
		}
	}
}

func (s *scanner) scanNumber() token {
	for {
		ch := s.read()
		if isNumericByte(byte(ch)) {
			s.text.WriteRune(ch)
		} else {
			s.unread()
			return token{
				Type: tNumber,
				Text: strings.TrimSpace(s.text.String()),
				Pos:  position{Line: s.line, Col: s.col, Offset: s.offset},
			}
		}
	}
}

func isNumericByte(b byte) bool {
	switch b {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		return true
	default:
		return false
	}
}

// eof represents a marker rune for the end of the reader.
var eof = rune(0)
