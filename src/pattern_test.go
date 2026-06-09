package fzf

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

var slab *util.Slab

func init() {
	slab = util.MakeSlab(slab16Size, slab32Size)
}

func TestParseTermsExtended(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$ | ^iii$ ^xxx | 'yyy | zzz$ | !ZZZ |")
	if len(terms) != 9 ||
		terms[0][0].typ != termFuzzy || terms[0][0].inv ||
		terms[1][0].typ != termExact || terms[1][0].inv ||
		terms[2][0].typ != termPrefix || terms[2][0].inv ||
		terms[3][0].typ != termSuffix || terms[3][0].inv ||
		terms[4][0].typ != termExact || !terms[4][0].inv ||
		terms[5][0].typ != termFuzzy || !terms[5][0].inv ||
		terms[6][0].typ != termPrefix || !terms[6][0].inv ||
		terms[7][0].typ != termSuffix || !terms[7][0].inv ||
		terms[7][1].typ != termEqual || terms[7][1].inv ||
		terms[8][0].typ != termPrefix || terms[8][0].inv ||
		terms[8][1].typ != termExact || terms[8][1].inv ||
		terms[8][2].typ != termSuffix || terms[8][2].inv ||
		terms[8][3].typ != termExact || !terms[8][3].inv {
		t.Errorf("%v", terms)
	}
	for _, termSet := range terms[:8] {
		term := termSet[0]
		if len(term.text) != 3 {
			t.Errorf("%v", term)
		}
	}
}

func TestParseTermsExtendedExact(t *testing.T) {
	terms := parseTerms(false, CaseSmart, false,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$")
	if len(terms) != 8 ||
		terms[0][0].typ != termExact || terms[0][0].inv || len(terms[0][0].text) != 3 ||
		terms[1][0].typ != termFuzzy || terms[1][0].inv || len(terms[1][0].text) != 3 ||
		terms[2][0].typ != termPrefix || terms[2][0].inv || len(terms[2][0].text) != 3 ||
		terms[3][0].typ != termSuffix || terms[3][0].inv || len(terms[3][0].text) != 3 ||
		terms[4][0].typ != termExact || !terms[4][0].inv || len(terms[4][0].text) != 3 ||
		terms[5][0].typ != termFuzzy || !terms[5][0].inv || len(terms[5][0].text) != 3 ||
		terms[6][0].typ != termPrefix || !terms[6][0].inv || len(terms[6][0].text) != 3 ||
		terms[7][0].typ != termSuffix || !terms[7][0].inv || len(terms[7][0].text) != 3 {
		t.Errorf("%v", terms)
	}
}

func TestParseTermsEmpty(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "' ^ !' !^")
	if len(terms) != 0 {
		t.Errorf("%v", terms)
	}
}

func buildPattern(fuzzy bool, fuzzyAlgo algo.Algo, extended bool, caseMode Case, normalize bool, forward bool,
	withPos bool, cacheable bool, nth []Range, delimiter Delimiter, runes []rune) *Pattern {
	return BuildPattern(NewChunkCache(), make(map[string]*Pattern),
		fuzzy, fuzzyAlgo, extended, caseMode, normalize, forward,
		withPos, cacheable, nth, delimiter, revision{}, runes, nil, 0)
}

func TestExact(t *testing.T) {
	pattern := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("'abc"))
	chars := util.ToChars([]byte("aabbcc abc"))
	res, pos := algo.ExactMatchNaive(
		pattern.caseSensitive, pattern.normalize, pattern.forward, &chars, pattern.termSets[0][0].text, true, nil)
	if res.Start != 7 || res.End != 10 {
		t.Errorf("%v / %d / %d", pattern.termSets, res.Start, res.End)
	}
	if pos != nil {
		t.Errorf("pos is expected to be nil")
	}
}

func TestEqual(t *testing.T) {
	pattern := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true, []Range{}, Delimiter{}, []rune("^AbC$"))

	match := func(str string, sidxExpected int, eidxExpected int) {
		chars := util.ToChars([]byte(str))
		res, pos := algo.EqualMatch(
			pattern.caseSensitive, pattern.normalize, pattern.forward, &chars, pattern.termSets[0][0].text, true, nil)
		if res.Start != sidxExpected || res.End != eidxExpected {
			t.Errorf("%v / %d / %d", pattern.termSets, res.Start, res.End)
		}
		if pos != nil {
			t.Errorf("pos is expected to be nil")
		}
	}
	match("ABC", -1, -1)
	match("AbC", 0, 3)
	match("AbC  ", 0, 3)
	match(" AbC ", 1, 4)
	match("  AbC", 2, 5)
}

func TestCaseSensitivity(t *testing.T) {
	pat1 := buildPattern(true, algo.FuzzyMatchV2, false, CaseSmart, false, true, false, true, []Range{}, Delimiter{}, []rune("abc"))
	pat2 := buildPattern(true, algo.FuzzyMatchV2, false, CaseSmart, false, true, false, true, []Range{}, Delimiter{}, []rune("Abc"))
	pat3 := buildPattern(true, algo.FuzzyMatchV2, false, CaseIgnore, false, true, false, true, []Range{}, Delimiter{}, []rune("abc"))
	pat4 := buildPattern(true, algo.FuzzyMatchV2, false, CaseIgnore, false, true, false, true, []Range{}, Delimiter{}, []rune("Abc"))
	pat5 := buildPattern(true, algo.FuzzyMatchV2, false, CaseRespect, false, true, false, true, []Range{}, Delimiter{}, []rune("abc"))
	pat6 := buildPattern(true, algo.FuzzyMatchV2, false, CaseRespect, false, true, false, true, []Range{}, Delimiter{}, []rune("Abc"))

	if string(pat1.text) != "abc" || pat1.caseSensitive != false ||
		string(pat2.text) != "Abc" || pat2.caseSensitive != true ||
		string(pat3.text) != "abc" || pat3.caseSensitive != false ||
		string(pat4.text) != "abc" || pat4.caseSensitive != false ||
		string(pat5.text) != "abc" || pat5.caseSensitive != true ||
		string(pat6.text) != "Abc" || pat6.caseSensitive != true {
		t.Error("Invalid case conversion")
	}
}

func TestOrigTextAndTransformed(t *testing.T) {
	pattern := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true, []Range{}, Delimiter{}, []rune("jg"))
	tokens := Tokenize("junegunn", Delimiter{})
	trans := Transform(tokens, []Range{{1, 1}})

	origBytes := []byte("junegunn.choi")
	for _, extended := range []bool{false, true} {
		chunk := Chunk{count: 1}
		chunk.items[0] = Item{
			text:        util.ToChars([]byte("junegunn")),
			origText:    &origBytes,
			transformed: &transformed{pattern.revision, trans}}
		pattern.extended = extended
		matches, _ := pattern.matchChunk(&chunk, nil, slab) // No cache
		if !(matches[0].item.text.ToString() == "junegunn" &&
			string(*matches[0].item.origText) == "junegunn.choi" &&
			reflect.DeepEqual((*matches[0].item.transformed).tokens, trans)) {
			t.Error("Invalid match result", matches)
		}

		match, offsets, pos := pattern.MatchItem(&chunk.items[0], true, slab)
		if !(match.item.text.ToString() == "junegunn" &&
			string(*match.item.origText) == "junegunn.choi" &&
			offsets[0][0] == 0 && offsets[0][1] == 5 &&
			reflect.DeepEqual((*match.item.transformed).tokens, trans)) {
			t.Error("Invalid match result", match, offsets, extended)
		}
		if !((*pos)[0] == 4 && (*pos)[1] == 0) {
			t.Error("Invalid pos array", *pos)
		}
	}
}

func TestCacheKey(t *testing.T) {
	test := func(extended bool, patStr string, expected string, cacheable bool) {
		pat := buildPattern(true, algo.FuzzyMatchV2, extended, CaseSmart, false, true, false, true, []Range{}, Delimiter{}, []rune(patStr))
		if pat.CacheKey() != expected {
			t.Errorf("Expected: %s, actual: %s", expected, pat.CacheKey())
		}
		if pat.cacheable != cacheable {
			t.Errorf("Expected: %t, actual: %t (%s)", cacheable, pat.cacheable, patStr)
		}
	}
	test(false, "foo !bar", "foo !bar", true)
	test(false, "foo | bar !baz", "foo | bar !baz", true)
	test(true, "foo  bar  baz", "foo\tbar\tbaz", true)
	test(true, "foo !bar", "foo", false)
	test(true, "foo !bar   baz", "foo\tbaz", false)
	test(true, "foo | bar baz", "baz", false)
	test(true, "foo | bar | baz", "", false)
	test(true, "foo | bar !baz", "", false)
	test(true, "| | foo", "", false)
	test(true, "| | | foo", "foo", false)
}

func TestCacheable(t *testing.T) {
	test := func(fuzzy bool, str string, expected string, cacheable bool) {
		pat := buildPattern(fuzzy, algo.FuzzyMatchV2, true, CaseSmart, true, true, false, true, []Range{}, Delimiter{}, []rune(str))
		if pat.CacheKey() != expected {
			t.Errorf("Expected: %s, actual: %s", expected, pat.CacheKey())
		}
		if cacheable != pat.cacheable {
			t.Errorf("Invalid Pattern.cacheable for \"%s\": %v (expected: %v)", str, pat.cacheable, cacheable)
		}
	}
	test(true, "foo bar", "foo\tbar", true)
	test(true, "foo 'bar", "foo\tbar", false)
	test(true, "foo !bar", "foo", false)

	test(false, "foo bar", "foo\tbar", true)
	test(false, "foo 'bar", "foo", false)
	test(false, "foo '", "foo", true)
	test(false, "foo 'bar", "foo", false)
	test(false, "foo !bar", "foo", false)
}

func buildChunks(numChunks int) []*Chunk {
	chunks := make([]*Chunk, numChunks)
	words := []string{
		"src/main/java/com/example/service/UserService.java",
		"src/test/java/com/example/service/UserServiceTest.java",
		"docs/api/reference/endpoints.md",
		"lib/internal/utils/string_helper.go",
		"pkg/server/http/handler/auth.go",
		"build/output/release/app.exe",
		"config/production/database.yml",
		"scripts/deploy/kubernetes/setup.sh",
		"vendor/github.com/junegunn/fzf/src/core.go",
		"node_modules/.cache/babel/transform.js",
	}
	for ci := range numChunks {
		chunks[ci] = &Chunk{count: chunkSize}
		for i := range chunkSize {
			text := words[(ci*chunkSize+i)%len(words)]
			chunks[ci].items[i] = Item{text: util.ToChars([]byte(text))}
			chunks[ci].items[i].text.Index = int32(ci*chunkSize + i)
		}
	}
	return chunks
}

func buildPatternWith(cache *ChunkCache, runes []rune) *Pattern {
	return BuildPattern(cache, make(map[string]*Pattern),
		true, algo.FuzzyMatchV2, true, CaseSmart, false, true,
		false, true, []Range{}, Delimiter{}, revision{}, runes, nil, 0)
}

func TestBitmapCacheBenefit(t *testing.T) {
	numChunks := 100
	chunks := buildChunks(numChunks)
	queries := []string{"s", "se", "ser", "serv", "servi"}

	// 1. Run all queries with shared cache (simulates incremental typing)
	cache := NewChunkCache()
	for _, q := range queries {
		pat := buildPatternWith(cache, []rune(q))
		for _, chunk := range chunks {
			pat.Match(chunk, slab)
		}
	}

	// 2. GC and measure memory with cache populated
	runtime.GC()
	runtime.GC()
	var memWith runtime.MemStats
	runtime.ReadMemStats(&memWith)

	// 3. Clear cache, GC, measure again
	cache.Clear()
	runtime.GC()
	runtime.GC()
	var memWithout runtime.MemStats
	runtime.ReadMemStats(&memWithout)

	cacheMem := int64(memWith.Alloc) - int64(memWithout.Alloc)
	t.Logf("Chunks: %d, Queries: %d", numChunks, len(queries))
	t.Logf("Cache memory: %d bytes (%.1f KB)", cacheMem, float64(cacheMem)/1024)
	t.Logf("Per-chunk-per-query: %.0f bytes", float64(cacheMem)/float64(numChunks*len(queries)))

	// 4. Verify correctness: cached vs uncached produce same results
	cache2 := NewChunkCache()
	for _, q := range queries {
		pat := buildPatternWith(cache2, []rune(q))
		for _, chunk := range chunks {
			pat.Match(chunk, slab)
		}
	}
	for _, q := range queries {
		patCached := buildPatternWith(cache2, []rune(q))
		patFresh := buildPatternWith(NewChunkCache(), []rune(q))
		var countCached, countFresh int
		for _, chunk := range chunks {
			countCached += len(patCached.Match(chunk, slab))
			countFresh += len(patFresh.Match(chunk, slab))
		}
		if countCached != countFresh {
			t.Errorf("query=%q: cached=%d, fresh=%d", q, countCached, countFresh)
		}
		t.Logf("query=%q: matches=%d", q, countCached)
	}
}

// ============================================================
// parseTerms: empty / whitespace / special-char-only inputs
// ============================================================

func TestParseTermsEmptyString(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "")
	if len(terms) != 0 {
		t.Errorf("empty string: expected 0 term sets, got %d", len(terms))
	}
}

func TestParseTermsWhitespaceOnly(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "   ")
	if len(terms) != 0 {
		t.Errorf("whitespace-only: expected 0 term sets, got %d", len(terms))
	}
}

func TestParseTermsOnlyPipe(t *testing.T) {
	// A lone "|" with no preceding terms: set is empty so not consumed as OR,
	// treated as a literal fuzzy term
	terms := parseTerms(true, CaseSmart, false, "|")
	if len(terms) != 1 {
		t.Errorf("single pipe: expected 1 term set (literal), got %d", len(terms))
		return
	}
	if string(terms[0][0].text) != "|" {
		t.Errorf("expected literal '|', got %q", string(terms[0][0].text))
	}
}

func TestParseTermsOnlyBang(t *testing.T) {
	// "!" → inv=true, text becomes empty → filtered out
	terms := parseTerms(true, CaseSmart, false, "!")
	if len(terms) != 0 {
		t.Errorf("single bang: expected 0 term sets, got %d", len(terms))
	}
}

func TestParseTermsOnlySpecialPrefixes(t *testing.T) {
	// "^" → prefix with empty text → filtered out
	terms := parseTerms(true, CaseSmart, false, "^")
	if len(terms) != 0 {
		t.Errorf("'^': expected 0 term sets, got %d", len(terms))
	}

	// "!^" → inverse prefix with empty text → filtered out
	terms = parseTerms(true, CaseSmart, false, "!^")
	if len(terms) != 0 {
		t.Errorf("'!^': expected 0 term sets, got %d", len(terms))
	}

	// "$" alone → NOT treated as suffix (guarded by text != "$"), kept as literal
	terms = parseTerms(true, CaseSmart, false, "$")
	if len(terms) != 1 || string(terms[0][0].text) != "$" {
		t.Errorf("'$': expected 1 term set with literal '$', got %v", terms)
	}

	// "!$" → inv=true, text="$", the "$" guard prevents suffix, kept as literal inv term
	terms = parseTerms(true, CaseSmart, false, "!$")
	if len(terms) != 1 || !terms[0][0].inv || string(terms[0][0].text) != "$" {
		t.Errorf("'!$': expected 1 inv term with literal '$', got %v", terms)
	}
}

// ============================================================
// parseTerms: escaped spaces
// ============================================================

func TestParseTermsEscapedSpace(t *testing.T) {
	// "\ " is replaced with tab, then tab is replaced back with space
	terms := parseTerms(true, CaseSmart, false, "foo\\ bar")
	if len(terms) != 1 {
		t.Errorf("expected 1 term set, got %d", len(terms))
		return
	}
	if string(terms[0][0].text) != "foo bar" {
		t.Errorf("expected 'foo bar', got %q", string(terms[0][0].text))
	}
}

func TestParseTermsMultipleEscapedSpaces(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "a\\ b\\ c")
	if len(terms) != 1 || string(terms[0][0].text) != "a b c" {
		t.Errorf("expected 'a b c', got %v", terms)
	}
}

// ============================================================
// parseTerms: exact boundary ('word' quoting)
// ============================================================

func TestParseTermsExactBoundary(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "'hello'")
	if len(terms) != 1 {
		t.Errorf("expected 1 term set, got %d", len(terms))
		return
	}
	if terms[0][0].typ != termExactBoundary {
		t.Errorf("expected termExactBoundary, got %d", terms[0][0].typ)
	}
	if string(terms[0][0].text) != "hello" {
		t.Errorf("expected 'hello', got %q", string(terms[0][0].text))
	}
}

func TestParseTermsExactBoundarySingleChar(t *testing.T) {
	// 'a' has len(text) == 3 (including quotes) but after stripping it's 1 char
	// However the condition is len(text) > 2, and "'a'" has len 3 > 2, so it matches
	terms := parseTerms(true, CaseSmart, false, "'a'")
	if len(terms) != 1 || terms[0][0].typ != termExactBoundary || string(terms[0][0].text) != "a" {
		t.Errorf("expected exact boundary 'a', got %v", terms)
	}
}

func TestParseTermsExactBoundaryTooShort(t *testing.T) {
	// "''" has len 2, not > 2, so it doesn't match the boundary pattern
	// Falls through to the 'prefix check → termExact, text becomes "'"
	// Then "'" is 1 char, so it's kept
	terms := parseTerms(true, CaseSmart, false, "''")
	if len(terms) != 1 || terms[0][0].typ != termExact {
		t.Errorf("expected termExact for '', got %v", terms)
	}
}

// ============================================================
// parseTerms: OR operator edge cases
// ============================================================

func TestParseTermsLeadingPipe(t *testing.T) {
	// "| foo" → "|" is first token, set is empty, so "|" is treated as a literal term
	// Result: 2 term sets: [{"|"}] and [{foo}]
	terms := parseTerms(true, CaseSmart, false, "| foo")
	if len(terms) != 2 {
		t.Errorf("leading pipe: expected 2 term sets, got %d", len(terms))
		return
	}
	if string(terms[0][0].text) != "|" {
		t.Errorf("term[0]: expected '|' literal, got %q", string(terms[0][0].text))
	}
	if string(terms[1][0].text) != "foo" {
		t.Errorf("term[1]: expected 'foo', got %q", string(terms[1][0].text))
	}
}

func TestParseTermsTrailingPipe(t *testing.T) {
	// "foo |" → "foo" is first term, "|" sets afterBar=true, no more tokens → single set
	terms := parseTerms(true, CaseSmart, false, "foo |")
	if len(terms) != 1 {
		t.Errorf("trailing pipe: expected 1 term set, got %d", len(terms))
		return
	}
	if len(terms[0]) != 1 || string(terms[0][0].text) != "foo" {
		t.Errorf("expected [foo], got %v", terms)
	}
}

func TestParseTermsConsecutivePipes(t *testing.T) {
	// "foo | | bar":
	//   "foo" → added to set, switchSet=true
	//   first "|" → OR separator, switchSet=false, afterBar=true
	//   second "|" → afterBar=true so NOT consumed as OR, treated as literal "|"
	//   "bar" → new term set
	// Result: 2 term sets: [foo, "|"(literal)] and [bar]
	terms := parseTerms(true, CaseSmart, false, "foo | | bar")
	if len(terms) != 2 {
		t.Errorf("consecutive pipes: expected 2 term sets, got %d", len(terms))
		return
	}
	if len(terms[0]) != 2 || string(terms[0][0].text) != "foo" || string(terms[0][1].text) != "|" {
		t.Errorf("set[0]: expected [foo, |], got %v", terms[0])
	}
	if len(terms[1]) != 1 || string(terms[1][0].text) != "bar" {
		t.Errorf("set[1]: expected [bar], got %v", terms[1])
	}
}

func TestParseTermsMultipleOrGroups(t *testing.T) {
	// "foo | bar baz | qux" → two term sets:
	//   set1: [foo, bar] (OR group)
	//   set2: [baz, qux] (OR group)
	terms := parseTerms(true, CaseSmart, false, "foo | bar baz | qux")
	if len(terms) != 2 {
		t.Errorf("expected 2 term sets, got %d", len(terms))
		return
	}
	if len(terms[0]) != 2 || string(terms[0][0].text) != "foo" || string(terms[0][1].text) != "bar" {
		t.Errorf("set[0]: expected [foo, bar], got %v", terms[0])
	}
	if len(terms[1]) != 2 || string(terms[1][0].text) != "baz" || string(terms[1][1].text) != "qux" {
		t.Errorf("set[1]: expected [baz, qux], got %v", terms[1])
	}
}

// ============================================================
// parseTerms: inverse (negation) edge cases
// ============================================================

func TestParseTermsAllInverse(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "!foo !bar !baz")
	if len(terms) != 3 {
		t.Errorf("expected 3 term sets, got %d", len(terms))
		return
	}
	for i, ts := range terms {
		if !ts[0].inv {
			t.Errorf("term[%d] should be inverse", i)
		}
		if ts[0].typ != termExact {
			t.Errorf("term[%d] should be termExact, got %d", i, ts[0].typ)
		}
	}
}

func TestParseTermsInversePrefix(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "!^foo")
	if len(terms) != 1 || !terms[0][0].inv || terms[0][0].typ != termPrefix {
		t.Errorf("expected inverse prefix, got %v", terms)
	}
	if string(terms[0][0].text) != "foo" {
		t.Errorf("expected 'foo', got %q", string(terms[0][0].text))
	}
}

func TestParseTermsInverseSuffix(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "!foo$")
	if len(terms) != 1 || !terms[0][0].inv || terms[0][0].typ != termSuffix {
		t.Errorf("expected inverse suffix, got %v", terms)
	}
	if string(terms[0][0].text) != "foo" {
		t.Errorf("expected 'foo', got %q", string(terms[0][0].text))
	}
}

func TestParseTermsInverseEqual(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "!^foo$")
	if len(terms) != 1 || !terms[0][0].inv || terms[0][0].typ != termEqual {
		t.Errorf("expected inverse equal, got %v", terms)
	}
}

func TestParseTermsInverseFuzzy(t *testing.T) {
	// !' flips to fuzzy in fuzzy mode (inv + ')
	terms := parseTerms(true, CaseSmart, false, "!'foo")
	if len(terms) != 1 || !terms[0][0].inv || terms[0][0].typ != termFuzzy {
		t.Errorf("expected inverse fuzzy, got %v", terms)
	}
}

func TestParseTermsInverseWithOr(t *testing.T) {
	// "foo | !bar" → one set with [foo(fuzzy), bar(inv exact)]
	terms := parseTerms(true, CaseSmart, false, "foo | !bar")
	if len(terms) != 1 {
		t.Errorf("expected 1 term set, got %d", len(terms))
		return
	}
	if len(terms[0]) != 2 {
		t.Errorf("expected 2 terms, got %d", len(terms[0]))
		return
	}
	if terms[0][0].inv || terms[0][0].typ != termFuzzy {
		t.Errorf("first term should be non-inv fuzzy: %v", terms[0][0])
	}
	if !terms[0][1].inv || terms[0][1].typ != termExact {
		t.Errorf("second term should be inv exact: %v", terms[0][1])
	}
}

// ============================================================
// parseTerms: Unicode patterns
// ============================================================

func TestParseTermsUnicode(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "你好 世界")
	if len(terms) != 2 {
		t.Errorf("expected 2 term sets, got %d", len(terms))
		return
	}
	if string(terms[0][0].text) != "你好" {
		t.Errorf("term[0]: expected '你好', got %q", string(terms[0][0].text))
	}
	if string(terms[1][0].text) != "世界" {
		t.Errorf("term[1]: expected '世界', got %q", string(terms[1][0].text))
	}
}

func TestParseTermsUnicodePrefix(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "^你好")
	if len(terms) != 1 || terms[0][0].typ != termPrefix {
		t.Errorf("expected prefix, got %v", terms)
	}
	if string(terms[0][0].text) != "你好" {
		t.Errorf("expected '你好', got %q", string(terms[0][0].text))
	}
}

func TestParseTermsUnicodeSuffix(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "你好$")
	if len(terms) != 1 || terms[0][0].typ != termSuffix {
		t.Errorf("expected suffix, got %v", terms)
	}
}

func TestParseTermsUnicodeExact(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "'你好")
	if len(terms) != 1 || terms[0][0].typ != termExact {
		t.Errorf("expected exact, got %v", terms)
	}
}

func TestParseTermsUnicodeInverse(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "!你好")
	if len(terms) != 1 || !terms[0][0].inv {
		t.Errorf("expected inverse, got %v", terms)
	}
}

func TestParseTermsUnicodeOr(t *testing.T) {
	terms := parseTerms(true, CaseSmart, false, "你好 | 世界")
	if len(terms) != 1 || len(terms[0]) != 2 {
		t.Errorf("expected 1 set with 2 terms, got %v", terms)
		return
	}
	if string(terms[0][0].text) != "你好" || string(terms[0][1].text) != "世界" {
		t.Errorf("expected [你好, 世界], got %v", terms)
	}
}

// ============================================================
// parseTerms: case sensitivity in extended mode
// ============================================================

func TestParseTermsCaseSmart(t *testing.T) {
	// All lowercase → case insensitive
	terms := parseTerms(true, CaseSmart, false, "abc")
	if terms[0][0].caseSensitive {
		t.Error("lowercase should be case insensitive in CaseSmart")
	}

	// Mixed case → case sensitive
	terms = parseTerms(true, CaseSmart, false, "Abc")
	if !terms[0][0].caseSensitive {
		t.Error("mixed case should be case sensitive in CaseSmart")
	}
}

func TestParseTermsCaseIgnore(t *testing.T) {
	terms := parseTerms(true, CaseIgnore, false, "Abc")
	if terms[0][0].caseSensitive {
		t.Error("CaseIgnore should always be case insensitive")
	}
	if string(terms[0][0].text) != "abc" {
		t.Errorf("expected lowercased 'abc', got %q", string(terms[0][0].text))
	}
}

func TestParseTermsCaseRespect(t *testing.T) {
	terms := parseTerms(true, CaseRespect, false, "abc")
	if !terms[0][0].caseSensitive {
		t.Error("CaseRespect should always be case sensitive")
	}
}

// ============================================================
// parseTerms: complex mixed patterns
// ============================================================

func TestParseTermsComplexMixed(t *testing.T) {
	// Mix of fuzzy, exact, prefix, suffix, inverse, OR
	terms := parseTerms(true, CaseSmart, false, "^start 'exact middle$ !^notprefix !notsuffix$ | alt")
	if len(terms) != 5 {
		t.Errorf("expected 5 term sets, got %d", len(terms))
		return
	}
	// ^start → prefix
	if terms[0][0].typ != termPrefix || terms[0][0].inv {
		t.Errorf("term[0]: expected non-inv prefix, got %v", terms[0][0])
	}
	// 'exact → exact
	if terms[1][0].typ != termExact || terms[1][0].inv {
		t.Errorf("term[1]: expected non-inv exact, got %v", terms[1][0])
	}
	// middle$ → suffix
	if terms[2][0].typ != termSuffix || terms[2][0].inv {
		t.Errorf("term[2]: expected non-inv suffix, got %v", terms[2][0])
	}
	// !^notprefix → inverse prefix
	if terms[3][0].typ != termPrefix || !terms[3][0].inv {
		t.Errorf("term[3]: expected inv prefix, got %v", terms[3][0])
	}
	// !notsuffix$ | alt → OR group with [inv suffix, fuzzy]
	if len(terms[4]) != 2 {
		t.Errorf("term[4]: expected 2 terms, got %d", len(terms[4]))
		return
	}
	if terms[4][0].typ != termSuffix || !terms[4][0].inv {
		t.Errorf("term[4][0]: expected inv suffix, got %v", terms[4][0])
	}
	if terms[4][1].typ != termFuzzy || terms[4][1].inv {
		t.Errorf("term[4][1]: expected non-inv fuzzy, got %v", terms[4][1])
	}
}

// ============================================================
// BuildPattern: IsEmpty
// ============================================================

func TestIsEmpty(t *testing.T) {
	// Empty extended pattern
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune(""))
	if !pat.IsEmpty() {
		t.Error("empty pattern should be empty")
	}

	// Non-empty extended pattern
	pat = buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("foo"))
	if pat.IsEmpty() {
		t.Error("non-empty pattern should not be empty")
	}

	// Empty non-extended pattern
	pat = buildPattern(true, algo.FuzzyMatchV2, false, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune(""))
	if !pat.IsEmpty() {
		t.Error("empty non-extended pattern should be empty")
	}

	// Whitespace-only extended pattern → terms are empty → IsEmpty
	pat = buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("   "))
	if !pat.IsEmpty() {
		t.Error("whitespace-only pattern should be empty")
	}
}

// ============================================================
// BuildPattern: sortable with all-inverse terms
// ============================================================

func TestSortableAllInverse(t *testing.T) {
	// All inverse terms → not sortable
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("!foo !bar"))
	if pat.sortable {
		t.Error("all-inverse pattern should not be sortable")
	}

	// Mixed inverse and non-inverse → sortable
	pat = buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("foo !bar"))
	if !pat.sortable {
		t.Error("mixed pattern should be sortable")
	}
}

// ============================================================
// MatchItem edge cases
// ============================================================

func TestMatchItemEmptyPattern(t *testing.T) {
	// Empty pattern should not match (IsEmpty is true, matchChunk short-circuits)
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune(""))
	item := Item{text: util.ToChars([]byte("anything"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	// Empty pattern produces 0 termSets, extendedMatch returns 0 offsets,
	// len(offsets)==0 == len(termSets)==0, so it IS a match
	if match.item == nil {
		t.Error("empty extended pattern should match everything")
	}
}

func TestMatchItemAllInverse(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("!xyz !abc"))

	// Item that does NOT contain "xyz" or "abc" → should match (inverse satisfied)
	item := Item{text: util.ToChars([]byte("hello world"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("item without excluded terms should match")
	}

	// Item that contains "xyz" → should NOT match (first inverse term fails)
	item = Item{text: util.ToChars([]byte("hello xyz world"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("item containing excluded term should not match")
	}
}

func TestMatchItemOrPattern(t *testing.T) {
	// "foo | bar" → match if either "foo" or "bar" is found
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("'foo | 'bar"))

	// Contains "foo" but not "bar"
	item := Item{text: util.ToChars([]byte("hello foo world"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match 'foo'")
	}

	// Contains "bar" but not "foo"
	item = Item{text: util.ToChars([]byte("hello bar world"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match 'bar'")
	}

	// Contains neither
	item = Item{text: util.ToChars([]byte("hello world"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match without foo or bar")
	}
}

func TestMatchItemAndPattern(t *testing.T) {
	// "foo bar" → match only if both "foo" AND "bar" are found
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("'foo 'bar"))

	// Contains both
	item := Item{text: util.ToChars([]byte("foo and bar"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match with both terms present")
	}

	// Contains only one
	item = Item{text: util.ToChars([]byte("only foo here"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match with only one term present")
	}
}

func TestMatchItemInverseAndPositive(t *testing.T) {
	// "foo !bar" → match if "foo" is found AND "bar" is NOT found
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("'foo !'bar"))

	// Has "foo", no "bar"
	item := Item{text: util.ToChars([]byte("hello foo world"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match: has foo, no bar")
	}

	// Has both "foo" and "bar"
	item = Item{text: util.ToChars([]byte("foo bar"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match: has bar (excluded)")
	}

	// Has neither
	item = Item{text: util.ToChars([]byte("hello world"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match: no foo")
	}
}

func TestMatchItemPrefixSuffix(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("^hello world$"))

	item := Item{text: util.ToChars([]byte("hello beautiful world"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match: starts with hello and ends with world")
	}

	item = Item{text: util.ToChars([]byte("say hello world now"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match: doesn't start with hello")
	}
}

func TestMatchItemEqual(t *testing.T) {
	// ^exact$ → exact equality (with whitespace tolerance)
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("^hello$"))

	item := Item{text: util.ToChars([]byte("hello"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match exact")
	}

	item = Item{text: util.ToChars([]byte("hello world"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match: extra content")
	}
}

func TestMatchItemUnicode(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("'世界"))

	item := Item{text: util.ToChars([]byte("你好世界"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match Unicode substring")
	}

	item = Item{text: util.ToChars([]byte("你好地球"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match different Unicode text")
	}
}

// ============================================================
// MatchItem with nth (tokenizer integration)
// ============================================================

func TestMatchItemWithNth(t *testing.T) {
	// Match against specific fields
	nth, _ := splitNth("2")
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		nth, Delimiter{}, []rune("'world"))

	item := Item{text: util.ToChars([]byte("hello world"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match: field 2 is 'world'")
	}

	// Search in field 1 only → should not find "world"
	// Use a fresh item to avoid cached transformInput from the previous pattern
	nth1, _ := splitNth("1")
	pat1 := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		nth1, Delimiter{}, []rune("'world"))
	item2 := Item{text: util.ToChars([]byte("hello world"))}
	match, _, _ = pat1.MatchItem(&item2, false, slab)
	if match.item != nil {
		t.Error("should not match: field 1 is 'hello', not 'world'")
	}
}

func TestMatchItemWithNthAndDelimiter(t *testing.T) {
	nth, _ := splitNth("2")
	delim := delimiterRegexp(":")
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		nth, delim, []rune("'def"))

	item := Item{text: util.ToChars([]byte("abc:def:ghi"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should match: field 2 with ':' delimiter is 'def'")
	}
}

// ============================================================
// MatchItem non-extended (basic) mode
// ============================================================

func TestMatchItemBasicMode(t *testing.T) {
	// Non-extended fuzzy match
	pat := buildPattern(true, algo.FuzzyMatchV2, false, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("fzf"))

	item := Item{text: util.ToChars([]byte("fuzzy finder"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should fuzzy match 'fzf' in 'fuzzy finder'")
	}

	item = Item{text: util.ToChars([]byte("no match here"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match")
	}
}

func TestMatchItemBasicExact(t *testing.T) {
	// Non-extended exact match
	pat := buildPattern(false, algo.FuzzyMatchV2, false, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("hello"))

	item := Item{text: util.ToChars([]byte("say hello world"))}
	match, _, _ := pat.MatchItem(&item, false, slab)
	if match.item == nil {
		t.Error("should exact match 'hello'")
	}

	item = Item{text: util.ToChars([]byte("helo world"))}
	match, _, _ = pat.MatchItem(&item, false, slab)
	if match.item != nil {
		t.Error("should not match inexact 'helo'")
	}
}

// ============================================================
// Cache key and cacheability edge cases
// ============================================================

func TestCacheKeyOnlyInverse(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("!foo !bar"))
	if pat.cacheable {
		t.Error("all-inverse should not be cacheable")
	}
	if pat.CacheKey() != "" {
		t.Errorf("all-inverse cache key should be empty, got %q", pat.CacheKey())
	}
}

func TestCacheKeyOrGroup(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("foo | bar"))
	if pat.cacheable {
		t.Error("OR group should not be cacheable")
	}
}

func TestCacheKeySingleTerm(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("foo"))
	if !pat.cacheable {
		t.Error("single fuzzy term should be cacheable")
	}
	if pat.CacheKey() != "foo" {
		t.Errorf("expected 'foo', got %q", pat.CacheKey())
	}
}

// ============================================================
// Direct algo fast path
// ============================================================

func TestDirectAlgoSingleFuzzy(t *testing.T) {
	// Single fuzzy term, no nth → direct algo should be set
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("foo"))
	if pat.directAlgo == nil {
		t.Error("single fuzzy term should use direct algo")
	}
}

func TestDirectAlgoNotForInverse(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("!foo"))
	if pat.directAlgo != nil {
		t.Error("inverse term should not use direct algo")
	}
}

func TestDirectAlgoNotForMultipleTerms(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("foo bar"))
	if pat.directAlgo != nil {
		t.Error("multiple terms should not use direct algo")
	}
}

func TestDirectAlgoNotForExact(t *testing.T) {
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]Range{}, Delimiter{}, []rune("'foo"))
	if pat.directAlgo != nil {
		t.Error("exact term should not use direct algo")
	}
}

func TestDirectAlgoNotWithNth(t *testing.T) {
	nth, _ := splitNth("1")
	pat := buildPattern(true, algo.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		nth, Delimiter{}, []rune("foo"))
	if pat.directAlgo != nil {
		t.Error("pattern with nth should not use direct algo")
	}
}

func BenchmarkWithCache(b *testing.B) {
	numChunks := 100
	chunks := buildChunks(numChunks)
	queries := []string{"s", "se", "ser", "serv", "servi"}

	b.Run("cached", func(b *testing.B) {
		for range b.N {
			cache := NewChunkCache()
			for _, q := range queries {
				pat := buildPatternWith(cache, []rune(q))
				for _, chunk := range chunks {
					pat.Match(chunk, slab)
				}
			}
		}
	})

	b.Run("uncached", func(b *testing.B) {
		for range b.N {
			for _, q := range queries {
				cache := NewChunkCache()
				pat := buildPatternWith(cache, []rune(q))
				for _, chunk := range chunks {
					pat.Match(chunk, slab)
				}
			}
		}
	})
}
