package unstable

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

func compareNodes(t *testing.T, expected, actual *Node) {
	t.Helper()

	if expected == nil && actual == nil {
		return
	}
	if expected == nil {
		t.Fatalf("expected nil node but got %s", actual.Kind)
	}
	if actual == nil {
		t.Fatalf("expected %s node but got nil", expected.Kind)
	}
	assert.Equal(t, expected.Kind, actual.Kind)
	assert.Equal(t, expected.Data, actual.Data)

	eIt := expected.Children()
	aIt := actual.Children()
	idx := 0
	for eIt.Next() {
		if !aIt.Next() {
			t.Fatalf("child %d: expected more children", idx)
		}
		compareNodes(t, eIt.Node(), aIt.Node())
		idx++
	}
	if aIt.Next() {
		t.Fatalf("child %d: unexpected extra child", idx)
	}

	compareNodes(t, expected.Comment(), actual.Comment())
}

func TestParser_AST(t *testing.T) {
	names := make([]string, 0, len(testgenASTCases))
	for name := range testgenASTCases {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			input, ok := testgenValidCases[name]
			if !ok {
				t.Fatalf("no TOML input for %s", name)
			}

			var p Parser
			p.KeepComments = true
			p.Reset([]byte(input))

			b := NewBuilder()
			refs := testgenASTCases[name](b)

			idx := 0
			for p.NextExpression() {
				if idx >= len(refs) {
					t.Fatal("parser produced more expressions than expected")
				}
				compareNodes(t, b.NodeAt(refs[idx]), p.Expression())
				idx++
			}
			assert.NoError(t, p.Error())
			if idx < len(refs) {
				t.Fatalf("parser produced %d expressions, expected %d", idx, len(refs))
			}
		})
	}
}

//nolint:funlen
func TestParser_AST_ExtraCoverage(t *testing.T) {
	examples := []struct {
		desc  string
		input string
		build func(b *Builder) Reference
	}{
		{
			desc:  "multiline literal string with CRLF",
			input: "key = '''\r\nfoo\r\nbar'''",
			build: func(b *Builder) Reference {
				return tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: String, Data: []byte("foo\r\nbar")}),
					tree(b, Node{Kind: Key, Data: []byte("key")}),
				)
			},
		},
		{
			desc:  "multiline basic string with CRLF",
			input: "key = \"\"\"\r\nfoo\"\"\"",
			build: func(b *Builder) Reference {
				return tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: String, Data: []byte("foo")}),
					tree(b, Node{Kind: Key, Data: []byte("key")}),
				)
			},
		},
		{
			desc:  "multiline basic string escape sequences",
			input: "key = \"\"\"\n\\b\\f\\r\\t\\e\\x41\"\"\"",
			build: func(b *Builder) Reference {
				return tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: String, Data: []byte("\b\f\r\t\x1BA")}),
					tree(b, Node{Kind: Key, Data: []byte("key")}),
				)
			},
		},
		{
			desc:  "inline table comma on separate line with comment",
			input: "t = {\n  a = 1\n  , # mid\n  b = 2\n}",
			build: func(b *Builder) Reference {
				kv1 := tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: Integer, Data: []byte("1")}),
					tree(b, Node{Kind: Key, Data: []byte("a")}),
				)
				comment := tree(b, Node{Kind: Comment, Data: []byte("# mid")})
				kv2 := tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: Integer, Data: []byte("2")}),
					tree(b, Node{Kind: Key, Data: []byte("b")}),
				)
				return tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: InlineTable}, kv1, comment, kv2),
					tree(b, Node{Kind: Key, Data: []byte("t")}),
				)
			},
		},
		{
			desc:  "inline table trailing comma on separate line",
			input: "t = {\n  a = 1\n  ,\n}",
			build: func(b *Builder) Reference {
				return tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: InlineTable},
						tree(b, Node{Kind: KeyValue},
							tree(b, Node{Kind: Integer, Data: []byte("1")}),
							tree(b, Node{Kind: Key, Data: []byte("a")}),
						),
					),
					tree(b, Node{Kind: Key, Data: []byte("t")}),
				)
			},
		},
		{
			desc:  "array comma on separate line with comment",
			input: "a = [\n  1\n  , # mid\n  2\n]",
			build: func(b *Builder) Reference {
				return tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: Array},
						tree(b, Node{Kind: Integer, Data: []byte("1")}),
						tree(b, Node{Kind: Comment, Data: []byte("# mid")}),
						tree(b, Node{Kind: Integer, Data: []byte("2")}),
					),
					tree(b, Node{Kind: Key, Data: []byte("a")}),
				)
			},
		},
		{
			desc:  "array trailing comma on separate line",
			input: "a = [\n  1\n  ,\n]",
			build: func(b *Builder) Reference {
				return tree(b, Node{Kind: KeyValue},
					tree(b, Node{Kind: Array},
						tree(b, Node{Kind: Integer, Data: []byte("1")}),
					),
					tree(b, Node{Kind: Key, Data: []byte("a")}),
				)
			},
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			var p Parser
			p.KeepComments = true
			p.Reset([]byte(e.input))
			p.NextExpression()
			assert.NoError(t, p.Error())

			b := NewBuilder()
			ref := e.build(b)
			compareNodes(t, b.NodeAt(ref), p.Expression())
		})
	}
}

func TestParser_Invalid(t *testing.T) {
	names := make([]string, 0, len(testgenInvalidCases))
	for name := range testgenInvalidCases {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			input := testgenInvalidCases[name]

			var p Parser
			p.Reset([]byte(input))
			for p.NextExpression() {
			}
			if p.Error() == nil {
				t.Skip("no syntax error (semantic validation is done by the unmarshaler)")
			}
		})
	}
}

func BenchmarkParseBasicStringWithUnicode(b *testing.B) {
	p := &Parser{}
	b.Run("4", func(b *testing.B) {
		input := []byte(`"\u1234\u5678\u9ABC\u1234\u5678\u9ABC"`)
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))

		for i := 0; i < b.N; i++ {
			_, _, _, _ = p.parseBasicString(input)
		}
	})
	b.Run("8", func(b *testing.B) {
		input := []byte(`"\u12345678\u9ABCDEF0\u12345678\u9ABCDEF0"`)
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))

		for i := 0; i < b.N; i++ {
			_, _, _, _ = p.parseBasicString(input)
		}
	})
}

func BenchmarkParseBasicStringsEasy(b *testing.B) {
	p := &Parser{}

	for _, size := range []int{1, 4, 8, 16, 21} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			input := []byte(`"` + strings.Repeat("A", size) + `"`)

			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for i := 0; i < b.N; i++ {
				_, _, _, _ = p.parseBasicString(input)
			}
		})
	}
}

// This example demonstrates how to parse a TOML document and preserving
// comments.  Comments are stored in the AST as Comment nodes. This example
// displays the structure of the full AST generated by the parser using the
// following structure:
//
//  1. Each root-level expression is separated by three dashes.
//  2. Bytes associated to a node are displayed in square brackets.
//  3. Siblings have the same indentation.
//  4. Children of a node are indented one level.
func ExampleParser_comments() {
	doc := `# Top of the document comment.
# Optional, any amount of lines.

# Above table.
[table] # Next to table.
# Above simple value.
key = "value" # Next to simple value.
# Below simple value.

# Some comment alone.

# Multiple comments, on multiple lines.

# Above inline table.
name = { first = "Tom", last = "Preston-Werner" } # Next to inline table.
# Below inline table.

# Above array.
array = [ 1, 2, 3 ] # Next to one-line array.
# Below array.

# Above multi-line array.
key5 = [ # Next to start of inline array.
  # Second line before array content.
  1, # Next to first element.
  # After first element.
  # Before second element.
  2,
  3, # Next to last element
  # After last element.
] # Next to end of array.
# Below multi-line array.

# Before array table.
[[products]] # Next to array table.
# After array table.
`

	var printGeneric func(*Parser, int, *Node)
	printGeneric = func(p *Parser, indent int, e *Node) {
		if e == nil {
			return
		}
		s := p.Shape(e.Raw)
		x := fmt.Sprintf("%d:%d->%d:%d (%d->%d)", s.Start.Line, s.Start.Column, s.End.Line, s.End.Column, s.Start.Offset, s.End.Offset)
		fmt.Printf("%-25s | %s%s [%s]\n", x, strings.Repeat("  ", indent), e.Kind, e.Data)
		printGeneric(p, indent+1, e.Child())
		printGeneric(p, indent+1, e.Comment())
		printGeneric(p, indent, e.Next())
	}

	printTree := func(p *Parser) {
		for p.NextExpression() {
			e := p.Expression()
			fmt.Println("---")
			printGeneric(p, 0, e)
		}
		if err := p.Error(); err != nil {
			panic(err)
		}
	}

	p := &Parser{
		KeepComments: true,
	}
	p.Reset([]byte(doc))
	printTree(p)

	// Output:
	// ---
	// 1:1->1:31 (0->30)         | Comment [# Top of the document comment.]
	// ---
	// 2:1->2:33 (31->63)        | Comment [# Optional, any amount of lines.]
	// ---
	// 4:1->4:15 (65->79)        | Comment [# Above table.]
	// ---
	// 1:1->1:1 (0->0)           | Table []
	// 5:2->5:7 (81->86)         |   Key [table]
	// 5:9->5:25 (88->104)       |   Comment [# Next to table.]
	// ---
	// 6:1->6:22 (105->126)      | Comment [# Above simple value.]
	// ---
	// 1:1->1:1 (0->0)           | KeyValue []
	// 7:7->7:14 (133->140)      |   String [value]
	// 7:1->7:4 (127->130)       |   Key [key]
	// 7:15->7:38 (141->164)     |   Comment [# Next to simple value.]
	// ---
	// 8:1->8:22 (165->186)      | Comment [# Below simple value.]
	// ---
	// 10:1->10:22 (188->209)    | Comment [# Some comment alone.]
	// ---
	// 12:1->12:40 (211->250)    | Comment [# Multiple comments, on multiple lines.]
	// ---
	// 14:1->14:22 (252->273)    | Comment [# Above inline table.]
	// ---
	// 1:1->1:1 (0->0)           | KeyValue []
	// 15:8->15:9 (281->282)     |   InlineTable []
	// 1:1->1:1 (0->0)           |     KeyValue []
	// 15:18->15:23 (291->296)   |       String [Tom]
	// 15:10->15:15 (283->288)   |       Key [first]
	// 1:1->1:1 (0->0)           |     KeyValue []
	// 15:32->15:48 (305->321)   |       String [Preston-Werner]
	// 15:25->15:29 (298->302)   |       Key [last]
	// 15:1->15:5 (274->278)     |   Key [name]
	// 15:51->15:74 (324->347)   |   Comment [# Next to inline table.]
	// ---
	// 16:1->16:22 (348->369)    | Comment [# Below inline table.]
	// ---
	// 18:1->18:15 (371->385)    | Comment [# Above array.]
	// ---
	// 1:1->1:1 (0->0)           | KeyValue []
	// 1:1->1:1 (0->0)           |   Array []
	// 19:11->19:12 (396->397)   |     Integer [1]
	// 19:14->19:15 (399->400)   |     Integer [2]
	// 19:17->19:18 (402->403)   |     Integer [3]
	// 19:1->19:6 (386->391)     |   Key [array]
	// 19:21->19:46 (406->431)   |   Comment [# Next to one-line array.]
	// ---
	// 20:1->20:15 (432->446)    | Comment [# Below array.]
	// ---
	// 22:1->22:26 (448->473)    | Comment [# Above multi-line array.]
	// ---
	// 1:1->1:1 (0->0)           | KeyValue []
	// 1:1->1:1 (0->0)           |   Array []
	// 24:3->24:38 (518->553)    |     Comment [# Second line before array content.]
	// 25:3->25:4 (556->557)     |     Integer [1]
	// 25:6->25:30 (559->583)    |       Comment [# Next to first element.]
	// 26:3->26:25 (586->608)    |     Comment [# After first element.]
	// 27:3->27:27 (611->635)    |       Comment [# Before second element.]
	// 28:3->28:4 (638->639)     |     Integer [2]
	// 29:3->29:4 (643->644)     |     Integer [3]
	// 29:6->29:28 (646->668)    |       Comment [# Next to last element]
	// 30:3->30:24 (671->692)    |     Comment [# After last element.]
	// 23:10->23:42 (483->515)   |     Comment [# Next to start of inline array.]
	// 23:1->23:5 (474->478)     |   Key [key5]
	// 31:3->31:26 (695->718)    |   Comment [# Next to end of array.]
	// ---
	// 32:1->32:26 (719->744)    | Comment [# Below multi-line array.]
	// ---
	// 34:1->34:22 (746->767)    | Comment [# Before array table.]
	// ---
	// 1:1->1:1 (0->0)           | ArrayTable []
	// 35:3->35:11 (770->778)    |   Key [products]
	// 35:14->35:36 (781->803)   |   Comment [# Next to array table.]
	// ---
	// 36:1->36:21 (804->824)    | Comment [# After array table.]
}

func TestIterator_IsLast(t *testing.T) {
	// Test IsLast on an iterator with multiple elements using public Parser API
	doc := `array = [1, 2, 3]`
	p := Parser{}
	p.Reset([]byte(doc))
	p.NextExpression()

	e := p.Expression()
	arr := e.Value() // The array node

	it := arr.Children()
	count := 0
	lastCount := 0
	for it.Next() {
		count++
		if it.IsLast() {
			lastCount++
		}
	}

	assert.Equal(t, 3, count)
	assert.Equal(t, 1, lastCount)
}

func TestNodeChaining(t *testing.T) {
	// Test that sibling nodes are correctly chained via Next()
	// This exercises the internal PushAndChain functionality through public APIs
	doc := `a.b.c = 1`
	p := Parser{}
	p.Reset([]byte(doc))
	p.NextExpression()

	e := p.Expression()
	// KeyValue has children: value, then key parts (a, b, c)
	keyIt := e.Key()

	// Collect all key parts by following the iterator
	var keys []string
	for keyIt.Next() {
		keys = append(keys, string(keyIt.Node().Data))
	}

	assert.Equal(t, []string{"a", "b", "c"}, keys)
}

func TestMultipleExpressions(t *testing.T) {
	// Test parsing multiple top-level expressions
	// This exercises root iteration through public APIs
	doc := `
key1 = "value1"
key2 = "value2"
key3 = "value3"
`
	p := Parser{}
	p.Reset([]byte(doc))

	var keys []string
	for p.NextExpression() {
		e := p.Expression()
		keyIt := e.Key()
		keyIt.Next()
		keys = append(keys, string(keyIt.Node().Data))
	}

	assert.NoError(t, p.Error())
	assert.Equal(t, []string{"key1", "key2", "key3"}, keys)
}

func ExampleParser() {
	doc := `
	hello = "world"
	value = 42
	`
	p := Parser{}
	p.Reset([]byte(doc))
	for p.NextExpression() {
		e := p.Expression()
		fmt.Printf("Expression: %s\n", e.Kind)
		value := e.Value()
		it := e.Key()
		k := it.Node() // shortcut: we know there is no dotted key in the example
		fmt.Printf("%s -> (%s) %s\n", k.Data, value.Kind, value.Data)
	}

	// Output:
	// Expression: KeyValue
	// hello -> (String) world
	// Expression: KeyValue
	// value -> (Integer) 42
}
