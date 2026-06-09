package fzf

import (
	"testing"
)

func TestParseRange(t *testing.T) {
	{
		i := ".."
		r, _ := ParseRange(&i)
		if r.begin != rangeEllipsis || r.end != rangeEllipsis {
			t.Errorf("%v", r)
		}
	}
	{
		i := "3.."
		r, _ := ParseRange(&i)
		if r.begin != 3 || r.end != rangeEllipsis {
			t.Errorf("%v", r)
		}
	}
	{
		i := "3..5"
		r, _ := ParseRange(&i)
		if r.begin != 3 || r.end != 5 {
			t.Errorf("%v", r)
		}
	}
	{
		i := "-3..-5"
		r, _ := ParseRange(&i)
		if r.begin != -3 || r.end != -5 {
			t.Errorf("%v", r)
		}
	}
	{
		i := "3"
		r, _ := ParseRange(&i)
		if r.begin != 3 || r.end != 3 {
			t.Errorf("%v", r)
		}
	}
	{
		i := "1..3..5"
		if r, ok := ParseRange(&i); ok {
			t.Errorf("%v", r)
		}
	}
	{
		i := "-3..3"
		if r, ok := ParseRange(&i); ok {
			t.Errorf("%v", r)
		}
	}
}

func TestTokenize(t *testing.T) {
	// AWK-style
	input := "  abc: \n\t def:  ghi  "
	tokens := Tokenize(input, Delimiter{})
	if tokens[0].text.ToString() != "abc: \n\t " || tokens[0].prefixLength != 2 {
		t.Errorf("%s", tokens)
	}

	// With delimiter
	tokens = Tokenize(input, delimiterRegexp(":"))
	if tokens[0].text.ToString() != "  abc:" || tokens[0].prefixLength != 0 {
		t.Error(tokens[0].text.ToString(), tokens[0].prefixLength)
	}

	// With delimiter regex
	tokens = Tokenize(input, delimiterRegexp("\\s+"))
	if tokens[0].text.ToString() != "  " || tokens[0].prefixLength != 0 ||
		tokens[1].text.ToString() != "abc: \n\t " || tokens[1].prefixLength != 2 ||
		tokens[2].text.ToString() != "def:  " || tokens[2].prefixLength != 10 ||
		tokens[3].text.ToString() != "ghi  " || tokens[3].prefixLength != 16 {
		t.Errorf("%s", tokens)
	}
}

func TestTransform(t *testing.T) {
	input := "  abc:  def:  ghi:  jkl"
	{
		tokens := Tokenize(input, Delimiter{})
		{
			ranges, _ := splitNth("1,2,3")
			tx := Transform(tokens, ranges)
			if JoinTokens(tx) != "abc:  def:  ghi:  " {
				t.Errorf("%s", tx)
			}
		}
		{
			ranges, _ := splitNth("1..2,3,2..,1")
			tx := Transform(tokens, ranges)
			if string(JoinTokens(tx)) != "abc:  def:  ghi:  def:  ghi:  jklabc:  " ||
				len(tx) != 4 ||
				tx[0].text.ToString() != "abc:  def:  " || tx[0].prefixLength != 2 ||
				tx[1].text.ToString() != "ghi:  " || tx[1].prefixLength != 14 ||
				tx[2].text.ToString() != "def:  ghi:  jkl" || tx[2].prefixLength != 8 ||
				tx[3].text.ToString() != "abc:  " || tx[3].prefixLength != 2 {
				t.Errorf("%s", tx)
			}
		}
	}
	{
		tokens := Tokenize(input, delimiterRegexp(":"))
		{
			ranges, _ := splitNth("1..2,3,2..,1")
			tx := Transform(tokens, ranges)
			if JoinTokens(tx) != "  abc:  def:  ghi:  def:  ghi:  jkl  abc:" ||
				len(tx) != 4 ||
				tx[0].text.ToString() != "  abc:  def:" || tx[0].prefixLength != 0 ||
				tx[1].text.ToString() != "  ghi:" || tx[1].prefixLength != 12 ||
				tx[2].text.ToString() != "  def:  ghi:  jkl" || tx[2].prefixLength != 6 ||
				tx[3].text.ToString() != "  abc:" || tx[3].prefixLength != 0 {
				t.Errorf("%s", tx)
			}
		}
	}
}

func TestTransformIndexOutOfBounds(t *testing.T) {
	s, _ := splitNth("1")
	Transform([]Token{}, s)
}

// ============================================================
// ParseRange edge cases
// ============================================================

func TestParseRangeEdgeCases(t *testing.T) {
	// Empty string
	s := ""
	if _, ok := ParseRange(&s); ok {
		t.Error("empty string should be invalid")
	}

	// Zero is invalid
	s = "0"
	if _, ok := ParseRange(&s); ok {
		t.Error("0 should be invalid")
	}

	// Zero in ranges
	for _, inv := range []string{"0..5", "5..0", "0..0", "..0", "0.."} {
		s = inv
		if _, ok := ParseRange(&s); ok {
			t.Errorf("%q should be invalid", inv)
		}
	}

	// Non-numeric strings
	for _, inv := range []string{"abc", "1..abc", "abc..1", "1.2"} {
		s = inv
		if _, ok := ParseRange(&s); ok {
			t.Errorf("%q should be invalid", inv)
		}
	}

	// Negative-to-positive cross is invalid
	s = "-3..3"
	if _, ok := ParseRange(&s); ok {
		t.Error("-3..3 should be invalid")
	}
}

func TestParseRangeDotDotN(t *testing.T) {
	// "..N" form: begin is rangeEllipsis
	s := "..5"
	r, ok := ParseRange(&s)
	if !ok || r.begin != rangeEllipsis || r.end != 5 {
		t.Errorf("..5: expected {%d,5}, got %v ok=%v", rangeEllipsis, r, ok)
	}

	// "..-3" form
	s = "..-3"
	r, ok = ParseRange(&s)
	if !ok || r.begin != rangeEllipsis || r.end != -3 {
		t.Errorf("..-3: expected {%d,-3}, got %v ok=%v", rangeEllipsis, r, ok)
	}
}

func TestParseRangeNegativeSingle(t *testing.T) {
	// "-1" → newRange(-1, -1) → end=-1 is normalized to rangeEllipsis
	s := "-1"
	r, ok := ParseRange(&s)
	if !ok || r.begin != -1 || r.end != rangeEllipsis {
		t.Errorf("-1: expected {-1,%d}, got %v ok=%v", rangeEllipsis, r, ok)
	}

	s = "-5"
	r, ok = ParseRange(&s)
	if !ok || r.begin != -5 || r.end != -5 {
		t.Errorf("-5: expected {-5,-5}, got %v ok=%v", r, ok)
	}
}

func TestParseRangeNormalization(t *testing.T) {
	// begin=1, end!=1 → begin becomes rangeEllipsis
	s := "1..5"
	r, ok := ParseRange(&s)
	if !ok || r.begin != rangeEllipsis || r.end != 5 {
		t.Errorf("1..5: expected {%d,5}, got %v", rangeEllipsis, r)
	}

	// begin=1, end=1 → no normalization
	s = "1"
	r, ok = ParseRange(&s)
	if !ok || r.begin != 1 || r.end != 1 {
		t.Errorf("1: expected {1,1}, got %v", r)
	}

	// end=-1 → end becomes rangeEllipsis
	s = "-3..-1"
	r, ok = ParseRange(&s)
	if !ok || r.begin != -3 || r.end != rangeEllipsis {
		t.Errorf("-3..-1: expected {-3,%d}, got %v", rangeEllipsis, r)
	}

	// 1.. → begin=rangeEllipsis (begin=1 normalization), end=rangeEllipsis (end=-1 normalization)
	s = "1.."
	r, ok = ParseRange(&s)
	if !ok || r.begin != rangeEllipsis || r.end != rangeEllipsis {
		t.Errorf("1..: expected {%d,%d}, got %v", rangeEllipsis, rangeEllipsis, r)
	}
}

func TestParseRangePositiveToNegative(t *testing.T) {
	// Positive begin, negative end is valid (e.g., field 3 to last-3)
	s := "3..-3"
	r, ok := ParseRange(&s)
	if !ok || r.begin != 3 || r.end != -3 {
		t.Errorf("3..-3: expected {3,-3}, got %v ok=%v", r, ok)
	}
}

func TestParseRangeDescending(t *testing.T) {
	// Descending positive range is syntactically valid
	s := "5..3"
	r, ok := ParseRange(&s)
	if !ok || r.begin != 5 || r.end != 3 {
		t.Errorf("5..3: expected {5,3}, got %v ok=%v", r, ok)
	}
}

// ============================================================
// Tokenize empty/whitespace inputs
// ============================================================

func TestTokenizeEmptyInput(t *testing.T) {
	// AWK mode: empty → no tokens
	tokens := Tokenize("", Delimiter{})
	if len(tokens) != 0 {
		t.Errorf("AWK empty: expected 0 tokens, got %d", len(tokens))
	}

	// String delimiter: empty → one empty token (from strings.SplitAfter)
	tokens = Tokenize("", delimiterRegexp(":"))
	if len(tokens) != 1 || tokens[0].text.ToString() != "" {
		t.Errorf("str delim empty: expected 1 empty token, got %d", len(tokens))
	}

	// Regex delimiter: empty → no tokens
	tokens = Tokenize("", delimiterRegexp("\\s+"))
	if len(tokens) != 0 {
		t.Errorf("regex delim empty: expected 0 tokens, got %d", len(tokens))
	}
}

func TestTokenizeWhitespaceOnlyAwk(t *testing.T) {
	// Pure whitespace: AWK should produce no tokens
	for _, input := range []string{" ", "  ", "\t", "\n", " \t \n "} {
		tokens := Tokenize(input, Delimiter{})
		if len(tokens) != 0 {
			t.Errorf("AWK whitespace-only %q: expected 0 tokens, got %d", input, len(tokens))
		}
	}
}

// ============================================================
// AWK tokenizer edge cases
// ============================================================

func TestTokenizeAwkSingleToken(t *testing.T) {
	tokens := Tokenize("hello", Delimiter{})
	if len(tokens) != 1 || tokens[0].text.ToString() != "hello" || tokens[0].prefixLength != 0 {
		t.Errorf("single token: %v", tokens)
	}
}

func TestTokenizeAwkLeadingWhitespace(t *testing.T) {
	tokens := Tokenize("\t  hello", Delimiter{})
	if len(tokens) != 1 || tokens[0].text.ToString() != "hello" || tokens[0].prefixLength != 3 {
		t.Errorf("leading whitespace: text=%q prefix=%d", tokens[0].text.ToString(), tokens[0].prefixLength)
	}
}

func TestTokenizeAwkTrailingWhitespace(t *testing.T) {
	// Trailing whitespace is attached to the last token
	tokens := Tokenize("hello   ", Delimiter{})
	if len(tokens) != 1 || tokens[0].text.ToString() != "hello   " || tokens[0].prefixLength != 0 {
		t.Errorf("trailing whitespace: text=%q prefix=%d", tokens[0].text.ToString(), tokens[0].prefixLength)
	}
}

func TestTokenizeAwkMixedWhitespace(t *testing.T) {
	// Tab, space, newline are all treated as whitespace separators
	tokens := Tokenize("a\tb\nc", Delimiter{})
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != "a\t" {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[1].text.ToString() != "b\n" {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
	if tokens[2].text.ToString() != "c" {
		t.Errorf("token[2]: %q", tokens[2].text.ToString())
	}
}

func TestTokenizeAwkMultipleConsecutiveWhitespace(t *testing.T) {
	// Multiple consecutive whitespace between tokens
	tokens := Tokenize("a   b", Delimiter{})
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
		return
	}
	// AWK attaches trailing whitespace to the preceding token
	if tokens[0].text.ToString() != "a   " {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[1].text.ToString() != "b" {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
}

// ============================================================
// Unicode content with AWK delimiter
// ============================================================

func TestTokenizeUnicodeContentAwk(t *testing.T) {
	tokens := Tokenize("  hello  世界  ", Delimiter{})
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != "hello  " {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[0].prefixLength != 2 {
		t.Errorf("token[0] prefixLength: expected 2, got %d", tokens[0].prefixLength)
	}
	if tokens[1].text.ToString() != "世界  " {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
}

func TestTokenizeUnicodeOnlyAwk(t *testing.T) {
	tokens := Tokenize("你好 世界", Delimiter{})
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != "你好 " {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[1].text.ToString() != "世界" {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
}

// ============================================================
// Unicode multi-byte delimiters
// ============================================================

func TestTokenizeUnicodeStringDelimiter(t *testing.T) {
	// Single-rune multi-byte delimiter (→ is 3 bytes)
	tokens := Tokenize("a→b→c", delimiterRegexp("→"))
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != "a→" {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[1].text.ToString() != "b→" {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
	if tokens[2].text.ToString() != "c" {
		t.Errorf("token[2]: %q", tokens[2].text.ToString())
	}
}

func TestTokenizeUnicodeMultiCharDelimiter(t *testing.T) {
	// Multi-character multi-byte delimiter ("::" is simple but "：：" is multi-byte)
	tokens := Tokenize("abc：：def：：ghi", delimiterRegexp("：："))
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != "abc：：" {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[1].text.ToString() != "def：：" {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
	if tokens[2].text.ToString() != "ghi" {
		t.Errorf("token[2]: %q", tokens[2].text.ToString())
	}
}

func TestTokenizeUnicodeRegexDelimiter(t *testing.T) {
	// Regex that matches Unicode characters
	tokens := Tokenize("abc→def→→ghi", delimiterRegexp("→+"))
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != "abc→" {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[1].text.ToString() != "def→→" {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
	if tokens[2].text.ToString() != "ghi" {
		t.Errorf("token[2]: %q", tokens[2].text.ToString())
	}
}

func TestTokenizeUnicodeContentWithUnicodeDelimiter(t *testing.T) {
	tokens := Tokenize("你好→世界→再见", delimiterRegexp("→"))
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != "你好→" {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[1].text.ToString() != "世界→" {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
	if tokens[2].text.ToString() != "再见" {
		t.Errorf("token[2]: %q", tokens[2].text.ToString())
	}
}

// ============================================================
// Consecutive and edge-case delimiters
// ============================================================

func TestTokenizeConsecutiveStringDelimiters(t *testing.T) {
	// Consecutive delimiters produce empty tokens between them
	tokens := Tokenize("a::b", delimiterRegexp(":"))
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != "a:" {
		t.Errorf("token[0]: %q", tokens[0].text.ToString())
	}
	if tokens[1].text.ToString() != ":" {
		t.Errorf("token[1]: %q", tokens[1].text.ToString())
	}
	if tokens[2].text.ToString() != "b" {
		t.Errorf("token[2]: %q", tokens[2].text.ToString())
	}
}

func TestTokenizeNoDelimiterMatch(t *testing.T) {
	// String delimiter that doesn't appear in input
	tokens := Tokenize("hello world", delimiterRegexp(":"))
	if len(tokens) != 1 || tokens[0].text.ToString() != "hello world" {
		t.Errorf("no match str: got %d tokens", len(tokens))
	}

	// Regex delimiter that doesn't match: whole input becomes one token
	tokens = Tokenize("abc", delimiterRegexp("[0-9]+"))
	if len(tokens) != 1 || tokens[0].text.ToString() != "abc" {
		t.Errorf("no match regex: expected 1 token 'abc', got %d tokens", len(tokens))
	}
}

func TestTokenizeInputIsDelimiter(t *testing.T) {
	// Input is exactly the delimiter
	tokens := Tokenize(":", delimiterRegexp(":"))
	if len(tokens) != 2 {
		// strings.SplitAfter(":", ":") returns [":", ""]
		t.Errorf("expected 2 tokens, got %d", len(tokens))
		return
	}
	if tokens[0].text.ToString() != ":" || tokens[1].text.ToString() != "" {
		t.Errorf("tokens: %q, %q", tokens[0].text.ToString(), tokens[1].text.ToString())
	}
}

// ============================================================
// StripLastDelimiter edge cases
// ============================================================

func TestStripLastDelimiterEmpty(t *testing.T) {
	// AWK mode: strips trailing whitespace
	if result := StripLastDelimiter("", Delimiter{}); result != "" {
		t.Errorf("AWK empty: expected empty, got %q", result)
	}

	// String delimiter
	if result := StripLastDelimiter("", delimiterRegexp(":")); result != "" {
		t.Errorf("str empty: expected empty, got %q", result)
	}

	// Regex delimiter
	if result := StripLastDelimiter("", delimiterRegexp("\\s+")); result != "" {
		t.Errorf("regex empty: expected empty, got %q", result)
	}
}

func TestStripLastDelimiterAwk(t *testing.T) {
	// AWK mode strips trailing whitespace
	if result := StripLastDelimiter("hello  ", Delimiter{}); result != "hello" {
		t.Errorf("expected %q, got %q", "hello", result)
	}
	if result := StripLastDelimiter("hello", Delimiter{}); result != "hello" {
		t.Errorf("expected %q, got %q", "hello", result)
	}
	if result := StripLastDelimiter("  \t\n", Delimiter{}); result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestStripLastDelimiterString(t *testing.T) {
	delim := delimiterRegexp(":")
	if result := StripLastDelimiter("abc:", delim); result != "abc" {
		t.Errorf("expected %q, got %q", "abc", result)
	}
	// No trailing delimiter: unchanged
	if result := StripLastDelimiter("abc", delim); result != "abc" {
		t.Errorf("expected %q, got %q", "abc", result)
	}
	// Multiple delimiters: only last stripped
	if result := StripLastDelimiter("a:b:", delim); result != "a:b" {
		t.Errorf("expected %q, got %q", "a:b", result)
	}
}

func TestStripLastDelimiterRegex(t *testing.T) {
	delim := delimiterRegexp("\\s+")
	if result := StripLastDelimiter("abc   ", delim); result != "abc" {
		t.Errorf("expected %q, got %q", "abc", result)
	}
	// No trailing match
	if result := StripLastDelimiter("abc", delim); result != "abc" {
		t.Errorf("expected %q, got %q", "abc", result)
	}
}

func TestStripLastDelimiterUnicode(t *testing.T) {
	delim := delimiterRegexp("→")
	if result := StripLastDelimiter("你好→", delim); result != "你好" {
		t.Errorf("expected %q, got %q", "你好", result)
	}
}

// ============================================================
// GetLastDelimiter edge cases
// ============================================================

func TestGetLastDelimiterEmpty(t *testing.T) {
	if result := GetLastDelimiter("", delimiterRegexp(":")); result != "" {
		t.Errorf("expected empty, got %q", result)
	}
	if result := GetLastDelimiter("", delimiterRegexp("\\s+")); result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestGetLastDelimiterFound(t *testing.T) {
	// String delimiter at end
	if result := GetLastDelimiter("abc:", delimiterRegexp(":")); result != ":" {
		t.Errorf("expected %q, got %q", ":", result)
	}
	// Not at end
	if result := GetLastDelimiter("a:b", delimiterRegexp(":")); result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestGetLastDelimiterRegex(t *testing.T) {
	delim := delimiterRegexp("\\s+")
	if result := GetLastDelimiter("abc   ", delim); result != "   " {
		t.Errorf("expected %q, got %q", "   ", result)
	}
	if result := GetLastDelimiter("abc", delim); result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestGetLastDelimiterUnicode(t *testing.T) {
	if result := GetLastDelimiter("你好→", delimiterRegexp("→")); result != "→" {
		t.Errorf("expected %q, got %q", "→", result)
	}
}

// ============================================================
// JoinTokens edge cases
// ============================================================

func TestJoinTokensEmpty(t *testing.T) {
	if result := JoinTokens([]Token{}); result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestJoinTokensSingle(t *testing.T) {
	tokens := Tokenize("hello", Delimiter{})
	if result := JoinTokens(tokens); result != "hello" {
		t.Errorf("expected %q, got %q", "hello", result)
	}
}

// ============================================================
// Transform edge cases
// ============================================================

func TestTransformFullRange(t *testing.T) {
	tokens := Tokenize("a:b:c:d", delimiterRegexp(":"))
	ranges, _ := splitNth("..")
	tx := Transform(tokens, ranges)
	if len(tx) != 1 || JoinTokens(tx) != "a:b:c:d" {
		t.Errorf("full range: %q", JoinTokens(tx))
	}
}

func TestTransformNegativeIndex(t *testing.T) {
	// -1 means the last field
	tokens := Tokenize("a:b:c:d", delimiterRegexp(":"))
	ranges, _ := splitNth("-1")
	tx := Transform(tokens, ranges)
	if JoinTokens(tx) != "d" {
		t.Errorf("-1: expected %q, got %q", "d", JoinTokens(tx))
	}

	// -2 means second to last
	ranges, _ = splitNth("-2")
	tx = Transform(tokens, ranges)
	if JoinTokens(tx) != "c:" {
		t.Errorf("-2: expected %q, got %q", "c:", JoinTokens(tx))
	}
}

func TestTransformNegativeRange(t *testing.T) {
	tokens := Tokenize("a:b:c:d", delimiterRegexp(":"))
	// -2.. means from second-to-last to end
	ranges, _ := splitNth("-2..")
	tx := Transform(tokens, ranges)
	if JoinTokens(tx) != "c:d" {
		t.Errorf("-2..: expected %q, got %q", "c:d", JoinTokens(tx))
	}
}

func TestTransformOutOfBoundsIndex(t *testing.T) {
	tokens := Tokenize("a:b", delimiterRegexp(":"))
	// Field 10 doesn't exist → empty
	ranges, _ := splitNth("10")
	tx := Transform(tokens, ranges)
	if JoinTokens(tx) != "" {
		t.Errorf("out-of-bounds: expected empty, got %q", JoinTokens(tx))
	}
}

func TestTransformMultipleFields(t *testing.T) {
	tokens := Tokenize("a:b:c:d", delimiterRegexp(":"))
	// Reverse order
	ranges, _ := splitNth("-1,-2,-3,-4")
	tx := Transform(tokens, ranges)
	if JoinTokens(tx) != "dc:b:a:" {
		t.Errorf("reverse: expected %q, got %q", "dc:b:a:", JoinTokens(tx))
	}
}

func TestTransformEmptyTokens(t *testing.T) {
	// Transform with empty input returns tokens with empty text
	ranges, _ := splitNth("1,2,3")
	tx := Transform([]Token{}, ranges)
	if len(tx) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tx))
	}
	if JoinTokens(tx) != "" {
		t.Errorf("expected empty, got %q", JoinTokens(tx))
	}
}

// ============================================================
// RangesToString
// ============================================================

func TestRangesToString(t *testing.T) {
	tests := []struct {
		ranges []Range
		expect string
	}{
		{[]Range{{rangeEllipsis, rangeEllipsis}}, ".."},
		{[]Range{{3, 5}}, "3..5"},
		{[]Range{{3, 3}}, "3"},
		{[]Range{{3, rangeEllipsis}}, "3.."},
		{[]Range{{rangeEllipsis, 5}}, "..5"},
		{[]Range{{-3, -5}}, "-3..-5"},
		{[]Range{{1, 1}, {3, 3}}, "1,3"},
		{[]Range{}, ""},
	}
	for _, tt := range tests {
		result := RangesToString(tt.ranges)
		if result != tt.expect {
			t.Errorf("RangesToString(%v): expected %q, got %q", tt.ranges, tt.expect, result)
		}
	}
}

// ============================================================
// Range.IsFull
// ============================================================

func TestRangeIsFull(t *testing.T) {
	r := Range{rangeEllipsis, rangeEllipsis}
	if !r.IsFull() {
		t.Error(".. should be full")
	}
	r = Range{1, 1}
	if r.IsFull() {
		t.Error("{1,1} should not be full")
	}
	r = Range{rangeEllipsis, 5}
	if r.IsFull() {
		t.Error("{0,5} should not be full")
	}
}

// ============================================================
// Delimiter.IsAwk
// ============================================================

func TestDelimiterIsAwk(t *testing.T) {
	awk := Delimiter{}
	if !awk.IsAwk() {
		t.Error("empty Delimiter should be AWK")
	}
	d := delimiterRegexp(":")
	if d.IsAwk() {
		t.Error("string delimiter should not be AWK")
	}
	d = delimiterRegexp("\\s+")
	if d.IsAwk() {
		t.Error("regex delimiter should not be AWK")
	}
}
