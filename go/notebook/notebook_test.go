package notebook

import (
	"regexp"
	"strings"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	reBeginRed   = regexp.MustCompile(`\(~~`)
	reEndRed     = regexp.MustCompile(`~~\)`)
	reBeginGreen = regexp.MustCompile(`\(\+\+`)
	reEndGreen   = regexp.MustCompile(`\+\+\)`)
	ansiRed      = "\u001b[31m"
	ansiGreen    = "\u001b[32m"
	ansiReset    = "\u001b[0m"
)

func colorize(diff string) string {
	return reEndGreen.ReplaceAllString(reBeginGreen.ReplaceAllString(
		reEndRed.ReplaceAllString(reBeginRed.ReplaceAllString(diff, ansiRed), ansiReset),
		ansiGreen), ansiReset) + ansiReset
}

func charDiff(want, got string) string {
	return colorize(diff.CharacterDiff(want, got))
}

type cellRewriteTest struct {
	name string
	// input is the list of input cells (source code).
	input []string
	// want is the list of expected output cells (source code).
	want []string
}

// createNotebook is a helper function to create
// a notebook from a list of strings following a few mnemonics.
// - A cell is 'code' by default.
// - If the source starts with "## ", it is changed to 'markdown'.
func createNotebook(src []string) *Notebook {
	var cells []*Cell
	for _, cellSource := range src {
		ty := "code"
		if strings.HasPrefix(cellSource, "## ") {
			ty = "markdown"
		}
		cells = append(cells, &Cell{
			Type:   ty,
			Source: cellSource,
		})
	}
	return &Notebook{
		Cells: cells,
	}
}

func TestToStudent(t *testing.T) {
	tests := []cellRewriteTest{
		{
			name:  "Unchanged1",
			input: []string{"# unchanged"},
			want:  []string{"# unchanged"},
		},
		{
			name:  "Unchanged2",
			input: []string{"# unchanged\nmore", "aaa\nbbb"},
			want:  []string{"# unchanged\nmore", "aaa\nbbb"},
		},
		{
			name:  "Unchanged3",
			input: []string{"## unchanged\nmore", "aaa\nbbb"},
			want:  []string{"## unchanged\nmore", "aaa\nbbb"},
		},
		{
			name:  "TestMarkerRemoved1",
			input: []string{"# TEST\n## unchanged\nmore", "aaa\nbbb"},
			want:  []string{"## unchanged\nmore", "aaa\nbbb"},
		},
		{
			name:  "MasterMarkerSkipped1",
			input: []string{"# MASTER ONLY\n## should be\n skipped", "aaa\nbbb"},
			want:  []string{"aaa\nbbb"},
		},
		{
			name:  "MasterMarkerSkipped2",
			input: []string{" # MASTER ONLY \n## should be\n skipped", "aaa\nbbb"},
			want:  []string{"aaa\nbbb"},
		},
		{
			name:  "Solution0",
			input: []string{"%%solution\na = 1\nb = 2\nc = 3"},
			want:  []string{"..."},
		},
		{
			name:  "Solution1",
			input: []string{"%%solution\n# BEGIN SOLUTION\nx = 1\n# END SOLUTION"},
			want:  []string{"..."},
		},
		{
			name:  "Solution2",
			input: []string{"%%solution\n# BEGIN SOLUTION\nx = 1\n# END SOLUTION\n# Junk"},
			want:  []string{"...\n# Junk"},
		},
		{
			name:  "Solution3",
			input: []string{"%%solution\n# Junk1\n# BEGIN SOLUTION\nx = 1\n# END SOLUTION\n# Junk2"},
			want:  []string{"# Junk1\n...\n# Junk2"},
		},
		{
			name:  "Solution4_Indent",
			input: []string{"%%solution\n  # Junk1\n  # BEGIN SOLUTION\n  x = 1\n  # END SOLUTION\n  # Junk2"},
			want:  []string{"  # Junk1\n  ...\n  # Junk2"},
		},
		{
			name:  "Solution5_IndentBroken", // Indent is matched to BEGIN SOLUTION
			input: []string{"%%solution\n  # Junk1\n  # BEGIN SOLUTION\n  x = 1\n    # END SOLUTION\n    # Junk2"},
			want:  []string{"  # Junk1\n  ...\n    # Junk2"},
		},
		{
			name: "Prompt1",
			input: []string{`%%solution
""" # BEGIN PROMPT
# Your solution here
""" # END PROMPT
# Junk1
# BEGIN SOLUTION
x = 1
# END SOLUTION
# Junk2`},
			want: []string{`# Junk1
# Your solution here
# Junk2`},
		},
		{
			name: "Prompt2",
			input: []string{`%%solution
  """ # BEGIN PROMPT
	# Your solution here
  """ # END PROMPT
	# Junk1
	# BEGIN SOLUTION
	x = 1
	# END SOLUTION
	# Junk2`},
			want: []string{`	# Junk1
	# Your solution here
	# Junk2`},
		},
		{
			name:  "Unittest1",
			input: []string{"# BEGIN UNITTEST\nx = 1\n# END UNITTEST"},
			want:  []string{},
		},
		{
			name:  "Autotest1",
			input: []string{"result, log = %autotest HelloTest\nx = 1"},
			want:  []string{},
		},
		{
			name:  "Studenttest1",
			input: []string{"%%studenttest name\naaa\nbbb"},
			want:  []string{"aaa\nbbb"},
		},
		{
			name:  "Inlinetest1",
			input: []string{"%%inlinetest name\naaa\nbbb"},
			want:  []string{},
		},
		{
			name:  "GlobalContext1",
			input: []string{"# GLOBAL CONTEXT\naaa\nbbb"},
			want:  []string{"aaa\nbbb"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := createNotebook(tt.input)
			got, err := n.ToStudent(AnyLanguage)
			if err != nil {
				t.Errorf("ToStudent([%s]) returned error %s, want success",
					strings.Join(tt.input, "]["), err)
				return
			}
			if len(got.Cells) != len(tt.want) {
				t.Errorf("got %d output cells, want %d", len(got.Cells), len(tt.want))
			}
			var gotSources []string
			for _, cell := range got.Cells {
				gotSources = append(gotSources, cell.Source)
			}
			wantText := strings.Join(tt.want, "\n")
			gotText := strings.Join(gotSources, "\n")
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(wantText, gotText, true)
			different := false
			for _, d := range diffs {
				if d.Type != diffmatchpatch.DiffEqual {
					different = true
					break
				}
			}
			if different {
				t.Logf("Got:\n%q\n--\nWant:\n%q\n--", gotText, wantText)
				t.Errorf("Diffs:\n%s", dmp.DiffPrettyText(diffs))
			}
		})
	}
}

func TestToAutograder(t *testing.T) {
	tests := []cellRewriteTest{
		{
			name:  "Ignored1",
			input: []string{"# ignored", "# ignored2", "## ignored markdown"},
			want:  []string{},
		},
		{
			name: "Extracted1",
			input: []string{`
# junk
# BEGIN UNITTEST
import unittest;

class MyTest(unittest.TestCase):
	def test1(self):
		pass
# END UNITTEST
# junk`},
			want: []string{`import unittest;

class MyTest(unittest.TestCase):
	def test1(self):
		pass
`},
		},
		{
			name: "Extracted2",
			input: []string{`
# junk
# BEGIN UNITTEST
# import submission_source
import unittest

class MyTest(unittest.TestCase):
	def test1(self):
		pass
# END UNITTEST
# junk`},
			want: []string{`import submission_source
# import submission_source
import unittest

class MyTest(unittest.TestCase):
	def test1(self):
		pass
`},
		},
		{
			name: "Extracted3",
			input: []string{`
# junk
# BEGIN UNITTEST
import unittest

#  import   submission


class MyTest(unittest.TestCase):
	def test1(self):
		pass
# END UNITTEST
# junk`},
			want: []string{`import submission
import unittest

#  import   submission


class MyTest(unittest.TestCase):
	def test1(self):
		pass
`},
		},
		{
			name:  "Inlinetest1",
			input: []string{"context1", "context2", "%%inlinetest A\ninline1\ninline2"},
			want:  []string{"context1\ncontext2\n", "inline1\ninline2\n"},
		},
		{
			name:  "Studenttest1",
			input: []string{"context1", "context2", "%%studenttest A\ninline1\ninline2"},
			want:  []string{},
		},
		{
			name:  "ShellCallouts1",
			input: []string{"%%inlinetest A\n!pip install something\naaa\nbbb"},
			want:  []string{"\n", "aaa\nbbb\n"},
		},
		{
			name:  "ShellCallouts2",
			input: []string{"%%inlinetest A\naaa\n!pip install something\nbbb"},
			want:  []string{"\n", "aaa\nbbb\n"},
		},
		{
			name:  "ShellCallouts3",
			input: []string{"%%inlinetest A\naaa\nbbb\n!pip install something"},
			want:  []string{"\n", "aaa\nbbb\n\n"},
		},
		{
			name:  "ShellCallouts4",
			input: []string{"%%inlinetest A\naaa\n!pip install something\nbbb\n!something else\nccc"},
			want:  []string{"\n", "aaa\nbbb\nccc\n"},
		},
		{
			name:  "ShellCallouts5",
			input: []string{"context", "%%inlinetest A\naaa\n!pip install something\nbbb\n!something else\nccc"},
			want:  []string{"context\n", "aaa\nbbb\nccc\n"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := createNotebook(tt.input)
			got, err := n.ToAutograder()
			if err != nil {
				t.Errorf("ToAutograder([%s]) returned error %s, want success",
					strings.Join(tt.input, "]["), err)
				return
			}
			if len(got.Cells) != len(tt.want) {
				t.Errorf("got %d output cells, want %d", len(got.Cells), len(tt.want))
				for i, gotCell := range got.Cells {
					t.Logf("got cell %d: [%s]", i, gotCell.Source)
				}
			}
			var gotSources []string
			for _, cell := range got.Cells {
				gotSources = append(gotSources, cell.Source)
			}
			wantText := strings.Join(tt.want, "\n")
			gotText := strings.Join(gotSources, "\n")
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(wantText, gotText, true)
			different := false
			for _, d := range diffs {
				if d.Type != diffmatchpatch.DiffEqual {
					different = true
					break
				}
			}
			if different {
				t.Logf("Got:\n%q\n--\nWant:\n%q\n--", gotText, wantText)
				t.Errorf("Diffs:\n%s", dmp.DiffPrettyText(diffs))
			}
		})
	}
}

func TestUncomment(t *testing.T) {
	var tests = []struct {
		source string
		want   string
	}{
		{
			source: `abc`,
			want:   `abc`,
		},
		{
			source: `import abc`,
			want:   `import abc`,
		},
		{
			source: `#import abc`,
			want:   `import abc`,
		},
		{
			source: `#import abc_cde.efg_fgh`,
			want:   `import abc_cde.efg_fgh`,
		},
		{
			source: `xyz
#import abc`,
			want: `xyz
import abc`,
		},
		{
			source: `#import abc
xyz`,
			want: `import abc
xyz`,
		},
		{
			source: `abc
#import x
cde`,
			want: `abc
import x
cde`,
		},
		{
			source: `abc
#import x
#import y.z
cde`,
			want: `abc
import x
import y.z
cde`,
		},
		{
			// Replacement should work with CRLF line separators as well.
			source: " \tabc\r\n \t# import x\r\n \tcde",
			want:   " \tabc\r\n \timport x\r\n \tcde",
		},
		{
			// Replacement should work with CRLF line separators as well.
			source: " \tabc\r\n \t# import x\r\n \tcde",
			want:   " \tabc\r\n \timport x\r\n \tcde",
		},
		{
			// Replacement should work with CRLF line separators as well.
			source: " \tabc\r\n \t# import x\r\n\t # \timport y.z\r\n \tcde",
			want:   " \tabc\r\n \timport x\r\n\t import y.z\r\n \tcde",
		},
		{
			// Replacement should work with CRLF line separators as well.
			source: " \tabc\r\n \t# import x\r\n \tcde",
			want:   " \tabc\r\n \timport x\r\n \tcde",
		},
		{
			// Whitespace after the comment sign should not be preserved.
			source: `#  import abc`,
			want:   `import abc`,
		},
		{
			// Whitespace before the comment sign should be preserved.
			source: ` #  import abc`,
			want:   ` import abc`,
		},
		{
			// Whitespace before the comment sign should be preserved.
			source: `		#  import abc`,
			want: `		import abc`,
		},
	}
	for _, tt := range tests {
		got := uncommentImports(tt.source)
		if got != tt.want {
			t.Logf("Input:\n%s", tt.source)
			t.Logf("Want:\n%s", tt.want)
			t.Logf("Got:\n%s", got)
			t.Errorf("Output is not as expected, diff: %s", charDiff(tt.want, got))
		}
	}
}
