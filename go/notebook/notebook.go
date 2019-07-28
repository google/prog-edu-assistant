// Package notebook provides utility functions for working with Jupyter/IPython
// notebooks, i.e. JSON files following some conventions.
package notebook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

// Notebook represents a parsed Jupyter notebook.
type Notebook struct {
	// NBFormat is the nbformat field.
	NBFormat int `json:"nbformat"`
	// NBFormatMinor is the nbformat_minor field.
	NBFormatMinor int `json:"nbformat_minor"`
	// Data is the raw parsed JSON data. It is not written back on serialization.
	Data map[string]interface{} `json:"-"`
	// Metadat is the map of metadata.
	Metadata map[string]interface{} `json:"metadata"`
	// Cells is the list of cells.
	Cells []*Cell `json:"cells"`
}

// Cell represents one cell of a Jupyter notebook. It is limited in
// the kind of cells it can represent.
type Cell struct {
	// Type is "code" or "markdown".
	Type string
	// Data is the raw parsed JSON contents of the cell.
	// When serializing cell back to JSON, Data is ignored.
	Data map[string]interface{}
	// Metadata is the "metadata" field of the cell.
	Metadata map[string]interface{}
	// Outputs are the recorded outputs of the cell.
	Outputs map[string]string
	// Source is the raw source of the cell.
	Source string
}

// ParseFile loads a notebook file from the specified file and parses it
// into a Notebook structure.
func ParseFile(filename string) (*Notebook, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %q: %s", filename, err)
	}
	n, err := Parse(b)
	if err != nil {
		return nil, fmt.Errorf("error reading notebook from %q: %s", filename, err)
	}
	return n, nil
}

// parseText parses a piece of notebook that is expected to be textual
// (either a string directly or a list of strings).
func parseText(v interface{}) (text string, err error) {
	ss, ok := v.([]interface{})
	if !ok {
		text, ok = v.(string)
		if !ok {
			err = fmt.Errorf("cell.source is neither a list nor string but %s",
				reflect.TypeOf(v))
			return
		}
	} else {
		var lines []string
		for _, s := range ss {
			str, ok := s.(string)
			if !ok {
				err = fmt.Errorf("cell.source has not a string but %s",
					reflect.TypeOf(s))
				return
			}
			lines = append(lines, str)
		}
		text = strings.Join(lines, "")
	}
	return
}

// Parse parses a byte slice into a Notebook structure. The input data
// must be a notebook in JSON encoding.
func Parse(b []byte) (*Notebook, error) {
	data := make(map[string]interface{})
	err := json.Unmarshal(b, &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %s", err)
	}
	ret := &Notebook{
		Data: data,
	}
	if v, ok := data["nbformat"]; ok {
		val, _ := v.(float64)
		ret.NBFormat = int(val)
	}
	if v, ok := data["nbformat_minor"]; ok {
		val, _ := v.(float64)
		ret.NBFormatMinor = int(val)
	}
	ret.Metadata, _ = data["metadata"].(map[string]interface{})
	cells, ok := data["cells"]
	if ok {
		cellsList, ok := cells.([]interface{})
		if !ok {
			return nil, fmt.Errorf(".cells is not a list but %s", reflect.TypeOf(cells))
		}
		for _, x := range cellsList {
			celldata, ok := x.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("cell is not a map but %s", reflect.TypeOf(x))
			}
			cell := &Cell{}
			if v, ok := celldata["cell_type"]; ok {
				cell.Type, _ = v.(string)
			}
			if v, ok := celldata["metadata"]; ok {
				cell.Metadata, ok = v.(map[string]interface{})
			}
			cell.Source, err = parseText(celldata["source"])
			if v, ok := celldata["outputs"]; ok {
				ss, ok := v.([]interface{})
				if !ok {
					return nil, fmt.Errorf("cell.outputs is not a list but %s",
						reflect.TypeOf(v))
				}
				outputs := make(map[string]string)
				for _, s := range ss {
					m, ok := s.(map[string]interface{})
					if !ok {
						continue
					}
					nameVal, ok := m["name"]
					if !ok {
						continue
					}
					name, ok := nameVal.(string)
					if !ok {
						return nil, fmt.Errorf("output name is not a string but %s",
							reflect.TypeOf(nameVal))
					}
					outputs[name], err = parseText(m["text"])
					if err != nil {
						return nil, fmt.Errorf("could not parse text: %s", err)
					}
				}
				cell.Outputs = outputs
			}
			ret.Cells = append(ret.Cells, cell)
		}
	}
	return ret, nil
}

// marshalText serializes a multi-line text string
// into a format that is compatible with JSON encoder.
func marshalText(text string) []interface{} {
	var ret []interface{}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i == len(lines)-1 {
			ret = append(ret, line)
			break
		}
		ret = append(ret, line+"\n")
	}
	return ret
}

// json returns a JSON-like map representing a cell.
func (cell *Cell) json() map[string]interface{} {
	emptyMap := make(map[string]interface{})
	ret := make(map[string]interface{})
	var outputs []interface{}
	// TODO(salikh): Do we need to handle any other kind of output?
	for name, output := range cell.Outputs {
		o := make(map[string]interface{})
		o["name"] = name
		o["output_type"] = "stream"
		o["text"] = marshalText(output)
		outputs = append(outputs, o)
	}
	if cell.Metadata != nil {
		ret["metadata"] = cell.Metadata
	} else {
		ret["metadata"] = emptyMap
	}
	ret["cell_type"] = cell.Type
	if cell.Type == "code" {
		ret["execution_count"] = nil
		if len(outputs) > 0 {
			ret["outputs"] = outputs
		} else {
			// Empty slice.
			ret["outputs"] = []interface{}{}
		}
	}
	ret["source"] = marshalText(cell.Source)
	return ret
}

// Marshal produces a JSON content suitable for writing to .ipynb file.
func (n *Notebook) Marshal() ([]byte, error) {
	output := make(map[string]interface{})
	var cells []interface{}
	for _, cell := range n.Cells {
		cells = append(cells, cell.json())
	}
	output["nbformat"] = n.NBFormat
	output["nbformat_minor"] = n.NBFormatMinor
	output["metadata"] = n.Metadata
	output["cells"] = cells
	return json.Marshal(output)
}

// MapCells runs a function on each cell and replaces the cell with the returned values.
// If mapFunc returns error, the function terminates the iteration and returns the error.
func (n *Notebook) MapCells(mapFunc func(c *Cell) ([]*Cell, error)) (*Notebook, error) {
	var out []*Cell
	for _, cell := range n.Cells {
		ncell, err := mapFunc(cell)
		if err != nil {
			return nil, err
		}
		if len(ncell) > 0 {
			out = append(out, ncell...)
		}
	}
	return &Notebook{
		NBFormat:      n.NBFormat,
		NBFormatMinor: n.NBFormatMinor,
		Metadata:      n.Metadata,
		Cells:         out,
	}, nil
}

// TODO(salikh): Implement smarter replacement strategies similar to jassign, e.g.
// x = 1 # SOLUTION   ===>   x = ...
var (
	assignmentMetadataRegex     = regexp.MustCompile("(?m)^[ \t]*# ASSIGNMENT METADATA")
	exerciseMetadataRegex       = regexp.MustCompile("(?m)^[ \t]*# EXERCISE METADATA")
	languageMetadataRegex       = regexp.MustCompile("\\*\\*lang:([a-z]{2})\\*\\*")
	tripleBacktickedRegex       = regexp.MustCompile("(?ms)^```([^`]|`[^`]|``[^`])*^```")
	testMarkerRegex             = regexp.MustCompile("(?ms)^[ \t]*# TEST[^\n]*[\n]*")
	studentTestRegex            = regexp.MustCompile("(?ms)^[ \t]*#? ?%%studenttest(?:[ \t]+([a-zA-Z][a-zA-Z0-9_]*))[ \t]*[\n]*")
	inlineTestRegex             = regexp.MustCompile("(?ms)^[ \t]*#? ?%%inlinetest(?:[ \t]+([a-zA-Z][a-zA-Z0-9_]*))[ \t]*[\n]*")
	inlineOrStudentTestRegex    = regexp.MustCompile("(?ms)^[ \t]*#? ?%%(?:inline|student)test(?:[ \t]+([a-zA-Z][a-zA-Z0-9_]*))[ \t]*[\n]*")
	solutionMagicRegex          = regexp.MustCompile("^[ \t]*%%solution[^\n]*\n")
	solutionBeginRegex          = regexp.MustCompile("(?m)^([ \t]*)# BEGIN SOLUTION *\n")
	solutionEndRegex            = regexp.MustCompile("(?m)^[ \t]*# END SOLUTION *")
	promptBeginRegex            = regexp.MustCompile("(?m)^[ \t]*\"\"\" # BEGIN PROMPT *\n|^[ \t]*# BEGIN PROMPT *\n")
	promptEndRegex              = regexp.MustCompile("(?m)\n[ \t]*\"\"\" # END PROMPT *\n|\n[ \t]*# END PROMPT *\n")
	unittestBeginRegex          = regexp.MustCompile("(?m)^[ \t]*# BEGIN UNITTEST *\n")
	unittestEndRegex            = regexp.MustCompile("(?m)^[ \t]*# END UNITTEST *")
	autotestMarkerRegex         = regexp.MustCompile("%autotest|autotest\\(")
	submissionMarkerRegex       = regexp.MustCompile("(?ms)^[ \t]*%%(submission|solution)")
	templateOrReportMarkerRegex = regexp.MustCompile("(?ms)^[ \t]*%%(template|report)|report\\(")
	masterOnlyMarkerRegex       = regexp.MustCompile("(?ms)^[ \t]*#+ MASTER ONLY[^\n]*\n?")
	importRegex                 = regexp.MustCompile("(?m)^[ \t]*#[ \t]*import[ \t]+([a-zA-Z][a-zA-Z0-9_]*)[ \t]*$")
	templateRegex               = regexp.MustCompile("(?m)^[ \t]*%%template(?:[ \t]+([a-zA-Z][a-zA-Z0-9_]*))\n")
	reportRegex                 = regexp.MustCompile("(?m)^[ \t]*%%report.*\n *([a-zA-Z][a-zA-Z0-9_]*)$")
)

// hasMetadata detects whether the markdown block has a triple backtick-fenced block
// with a metadata marker given as a Regexp.
func hasMetadata(re *regexp.Regexp, source string) bool {
	mm := tripleBacktickedRegex.FindAllStringIndex(source, -1)
	for _, m := range mm {
		text := source[m[0]+3 : m[1]-3]
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// If text matches begin and end regexes in sequences, returns
// the text that is enclosed by matches. If the text does not match,
// return empty string. Returns error in pathological cases.
func cutText(begin, end *regexp.Regexp, text string) (string, error) {
	mbeg := begin.FindStringIndex(text)
	if mbeg == nil {
		return "", nil
	}
	mend := end.FindStringIndex(text)
	if mend == nil {
		return "", fmt.Errorf("missing %s", end)
	}
	if mend[1] < mbeg[0] {
		return "", fmt.Errorf("%s before %s", end, begin)
	}
	return text[mbeg[1]:mend[0]], nil
}

// extractMetadata extracts the metadata from the markdown cell, using the provided
// regexp to detect metadata fenced blocks. It returns nil if the source does not
// have any metadata fenced block. The second return argument is the source code
// with metadata block cut out, or the input source string if there were no metadata.
func extractMetadata(re *regexp.Regexp, source string) (metadata map[string]interface{}, newSource string, err error) {
	var outputs []string
	mm := tripleBacktickedRegex.FindAllStringIndex(source, -1)
	for i, m := range mm {
		if len(outputs) == 0 {
			outputs = append(outputs, source[0:m[0]])
		}
		text := source[m[0]+3 : m[1]-3]
		if re.MatchString(text) {
			metadata = make(map[string]interface{})
			err = yaml.Unmarshal([]byte(text), &metadata)
			if err != nil {
				err = fmt.Errorf("error parsing metadata: %s\n--\n%s\n--", err, text)
				return
			}
		} else {
			outputs = append(outputs, source[m[0]:m[1]])
		}
		if i < len(mm)-1 {
			outputs = append(outputs, source[m[1]:mm[i+1][0]])
		} else {
			outputs = append(outputs, source[m[1]:])
		}
	}
	newSource = strings.Join(outputs, "")
	return
}

// Language represents a type of natural languages in which a cell is written.
type Language int

const (
	English Language = iota
	Japanese
	AnyLanguage
)

func (l Language) String() string {
	switch l {
	case English:
		return "en"
	case Japanese:
		return "ja"
	default:
		return ""
	}
}

func (l Language) filterText(s string) string {
	// If no language is specified, use s by removing language metadata.
	if l == AnyLanguage {
		return languageMetadataRegex.ReplaceAllString(s, "")
	}

	var ms [][]byte
	if ms = languageMetadataRegex.FindSubmatch([]byte(s)); len(ms) == 0 {
		// We always use a cell as is where no language is specified.
		return s
	}

	// If the language used in the cell is different from l, return the empty string.
	if string(ms[1]) != l.String() {
		return ""
	}

	// Use s by removing language metadata.
	return languageMetadataRegex.ReplaceAllString(s, "")
}

// CleanForStudent takes a code cell and produces a clean student version,
// i.e. it removes the # TEST markers, replaces %%solution with a placeholder,
// drops the unit tests etc. If the cell needs to be dropped, this function
// returns nil.
func CleanForStudent(cell *Cell, assignmentMetadata, exerciseMetadata map[string]interface{}, lang Language) (*Cell, error) {
	if cell.Type == "markdown" {
		if masterOnlyMarkerRegex.MatchString(cell.Source) {
			// Skip # MASTER ONLY
			return nil, nil
		}
		if hasMetadata(assignmentMetadataRegex, cell.Source) {
			_, source, err := extractMetadata(assignmentMetadataRegex, cell.Source)
			if err != nil {
				return nil, err
			}
			// Replace the rewritten cell.
			cell = &Cell{
				Type:   cell.Type,
				Source: source,
			}
		}
		if hasMetadata(exerciseMetadataRegex, cell.Source) {
			_, source, err := extractMetadata(exerciseMetadataRegex, cell.Source)
			if err != nil {
				return nil, err
			}
			// Replace the rewritten cell.
			cell = &Cell{
				Type:   cell.Type,
				Source: source,
			}
		}
		// Pass through.
		return cell, nil
	}
	if cell.Type != "code" {
		// No need to do anything with non-code cells.
		return cell, nil
	}
	source := cell.Source
	if m := testMarkerRegex.FindStringIndex(source); m != nil {
		// Remove the # TEST marker.
		source = source[:m[0]] + source[m[1]:]
	}
	if m := studentTestRegex.FindStringIndex(source); m != nil {
		// Remove the %%studenttest  marker.
		source = source[:m[0]] + source[m[1]:]
	}
	if m := inlineTestRegex.FindStringIndex(source); m != nil {
		// Skip the %%inline test cell.
		return nil, nil
	}
	if m := solutionMagicRegex.FindStringIndex(source); m != nil {
		// Strip the line with %%solution magic.
		source = source[m[1]:]
		// Extract the prompt, if any.
		prompt := ""
		if mbeg := promptBeginRegex.FindStringIndex(source); mbeg != nil {
			mend := promptEndRegex.FindStringIndex(source)
			if mend == nil {
				return nil, fmt.Errorf("BEGIN PROMPT has no matching END PROMPT")
			}
			if mend[1] < mbeg[0] {
				return nil, fmt.Errorf("END PROMPT is before BEGIN  PROMPT")
			}
			prompt = source[mbeg[1]:mend[0]]
			glog.V(3).Infof("prompt = %q", prompt)
			source = strings.Join([]string{source[:mbeg[0]], source[mend[1]:]}, "")
			glog.V(3).Infof("stripped source = %q", source)
		}
		// Remove the solution.
		mbeg := solutionBeginRegex.FindAllStringSubmatchIndex(source, -1)
		if mbeg == nil {
			// No BEGIN/END SOLUTION markers. Just return "..."
			return &Cell{
				Type:     "code",
				Metadata: exerciseMetadata,
				Source:   "...",
			}, nil
		}
		// Match BEGIN SOLUTION to END SOLUTION.
		mend := solutionEndRegex.FindAllStringIndex(source, -1)
		if len(mbeg) != len(mend) {
			return nil, fmt.Errorf("cell has mismatched number of BEGIN SOLUTION and END SOLUTION, %d != %d", len(mbeg), len(mend))
		}
		var outputs []string
		for i, m := range mbeg {
			if i == 0 {
				outputs = append(outputs, source[0:m[0]])
			}
			// TODO(salikh): Fix indentation and add more heuristics.
			if prompt == "" {
				indent := source[m[2]:m[3]]
				prompt = indent + "..."
			}
			outputs = append(outputs, prompt)
			glog.V(3).Infof("prompt: %q", prompt)
			if i < len(mbeg)-1 {
				outputs = append(outputs, source[mend[i][1]:mbeg[i+1][0]])
			} else {
				outputs = append(outputs, source[mend[i][1]:])
				glog.V(3).Infof("last part: %q", source[mend[i][1]:])
			}
		}
		return &Cell{
			Type:     "code",
			Metadata: exerciseMetadata,
			Source:   strings.Join(outputs, ""),
		}, nil
	}
	// Skip # BEGIN UNITTEST, %%submission, %%solution, %autotest and # MASTER ONLY cells.
	if unittestBeginRegex.MatchString(source) ||
		autotestMarkerRegex.MatchString(source) ||
		submissionMarkerRegex.MatchString(source) ||
		templateOrReportMarkerRegex.MatchString(source) ||
		masterOnlyMarkerRegex.MatchString(source) {
		// Skip the cell.
		return nil, nil
	}
	// Source may have been modified.
	return &Cell{
		Type:   "code",
		Source: source,
	}, nil
	return &Cell{
		Type:   cell.Type,
		Source: source,
	}, nil
}

// ToStudent converts a master notebook into the student notebook.
func (n *Notebook) ToStudent(lang Language) (*Notebook, error) {
	// Assignment metadata is global for the notebook.
	assignmentMetadata := make(map[string]interface{})
	// Exercise metadata only applies to the next code block,
	// and is nil otherwise.
	var exerciseMetadata map[string]interface{}
	transformed, err := n.MapCells(func(cell *Cell) ([]*Cell, error) {
		source := cell.Source
		if cell.Type == "markdown" {
			var err error
			if hasMetadata(assignmentMetadataRegex, cell.Source) {
				var metadata map[string]interface{}
				metadata, source, err = extractMetadata(assignmentMetadataRegex, cell.Source)
				if err != nil {
					return nil, err
				}
				// Merge assignment metadata to global table.
				for k, v := range metadata {
					assignmentMetadata[k] = v
				}
			}
			if hasMetadata(exerciseMetadataRegex, cell.Source) {
				// Replace exercise metadata.
				exerciseMetadata, source, err = extractMetadata(exerciseMetadataRegex, cell.Source)
				if err != nil {
					return nil, err
				}
			}
		}
		if cell.Type == "markdown" {
			if masterOnlyMarkerRegex.MatchString(source) {
				// Skip # MASTER ONLY
				return nil, nil
			}
			if source = lang.filterText(source); len(source) == 0 {
				return nil, nil
			}
		}
		if cell.Type != "code" {
			return []*Cell{&Cell{Type: cell.Type, Source: source}}, nil
		}
		if m := testMarkerRegex.FindStringIndex(source); m != nil {
			// Remove the # TEST marker.
			source = source[:m[0]] + source[m[1]:]
		}
		if m := studentTestRegex.FindStringIndex(source); m != nil {
			// Remove the %%studenttest marker.
			source = source[:m[0]] + source[m[1]:]
		}
		if m := inlineTestRegex.FindStringIndex(source); m != nil {
			// Skip the %%inline test cell.
			return nil, nil
		}
		if m := solutionMagicRegex.FindStringIndex(source); m != nil {
			clean, err := CleanForStudent(cell, assignmentMetadata, exerciseMetadata, lang)
			if err != nil {
				return nil, err
			}
			return []*Cell{clean}, nil
		}
		// Skip # BEGIN UNITTEST, %%submission, %%solution, %autotest and # MASTER ONLY cells.
		if unittestBeginRegex.MatchString(source) ||
			autotestMarkerRegex.MatchString(source) ||
			submissionMarkerRegex.MatchString(source) ||
			templateOrReportMarkerRegex.MatchString(source) ||
			masterOnlyMarkerRegex.MatchString(source) {
			// Skip the cell.
			return nil, nil
		}
		// Source may have been modified.
		return []*Cell{&Cell{
			Type:   "code",
			Source: source,
		}}, nil
	})
	if err != nil {
		return nil, err
	}
	for k, v := range assignmentMetadata {
		transformed.Metadata[k] = v
	}
	return transformed, nil
}

// cloneMetadata makes a deep copy of the metadata in the parsed JSON format.
func cloneMetadata(metadata map[string]interface{}, extras ...interface{}) map[string]interface{} {
	ret := make(map[string]interface{})
	// Copy the metadata.
	for k, v := range metadata {
		ret[k] = v
	}
	// Add the extra values.
	for i := 0; i < len(extras); i += 2 {
		ret[extras[i].(string)] = extras[i+1]
	}
	return ret
}

var (
	// testClassRegex detects the test cases that need to be written down into a separate file.
	// The name of the file is derived from the name of the test class.
	testClassRegex = regexp.MustCompile(`(?m)^[ \t]*class ([a-zA-Z_0-9]*)\(unittest\.TestCase\):`)
)

// ToAutograder converts a master notebook into the intermediate format called "autograder notebook".
// The autograder notebook is a format where each cell corresponds to one file,
// and the file name is stored in metadata["filename"]. It is later written into the autograder directory.
// Note: the autograder notebooks do not exist in the form of notebook files, it is only a convenience
// representation that it actually saved in the directory autograder format.
func (n *Notebook) ToAutograder() (*Notebook, error) {
	// Assignment metadata is global for the notebook.
	assignmentMetadata := make(map[string]interface{})
	var assignmentID string
	// Exercise ID is state that applies to subsequent unittest cells.
	var exerciseID string
	var exerciseMetadata map[string]interface{}
	// Context cells are the code cells before the start of the first exercise,
	// and code cells from the beginning of the exercise, excluding the solution cell,
	// but including the student test cells.
	var globalContext []*Cell
	var exerciseContext []*Cell
	transformed, err := n.MapCells(func(cell *Cell) ([]*Cell, error) {
		source := cell.Source
		if cell.Type == "markdown" {
			var err error
			if hasMetadata(assignmentMetadataRegex, cell.Source) {
				var metadata map[string]interface{}
				metadata, source, err = extractMetadata(assignmentMetadataRegex, cell.Source)
				if err != nil {
					return nil, err
				}
				if v, ok := metadata["assignment_id"]; ok {
					id, ok := v.(string)
					if !ok {
						return nil, fmt.Errorf("assignment_id is not a string, but %s", reflect.TypeOf(v))
					}
					assignmentID = id
				}
				// Merge assignment metadata to global table.
				for k, v := range metadata {
					assignmentMetadata[k] = v
				}
			}
			if hasMetadata(exerciseMetadataRegex, cell.Source) {
				// Replace exercise metadata.
				exerciseMetadata, source, err = extractMetadata(exerciseMetadataRegex, cell.Source)
				if err != nil {
					return nil, err
				}
				if v, ok := exerciseMetadata["exercise_id"]; ok {
					id, ok := v.(string)
					if !ok {
						return nil, fmt.Errorf("exercise_id is not a string, but %s", reflect.TypeOf(v))
					}
					exerciseID = id
				}
				glog.V(3).Infof("parsed metadata: %s", exerciseMetadata)
				// Reset the exercise context.
				exerciseContext = nil
			}
		}
		if cell.Type != "code" {
			// We do not need to emit non-code cells.
			return nil, nil
		}
		if m := inlineTestRegex.FindStringSubmatchIndex(source); m != nil {
			// Extract the inline test name.
			name := source[m[2]:m[3]]
			// Peel off the magic string.
			source = source[m[1]:]
			var parts []string
			// Create an inline test.
			for _, c := range globalContext {
				// Create a clean student version of a code cell.
				clean, err := CleanForStudent(c, assignmentMetadata, exerciseMetadata, AnyLanguage)
				if err != nil {
					return nil, err
				}
				if clean != nil {
					// Accumulate code.
					parts = append(parts, clean.Source)
				}
			}
			return []*Cell{
				// Store the context and the inline test itself into separate files,
				// which will be used by the autograder to synthesize a complete inline test.
				&Cell{
					Type:     "code",
					Metadata: cloneMetadata(exerciseMetadata, "filename", name+"_context.py", "assignment_id", assignmentID),
					Source:   strings.Join(parts, "\n") + "\n",
				},
				&Cell{
					Type:     "code",
					Metadata: cloneMetadata(exerciseMetadata, "filename", name+"_inline.py", "assignment_id", assignmentID),
					Source:   source + "\n",
				},
			}, nil
		} else if unittestBeginRegex.MatchString(source) {
			text, err := cutText(unittestBeginRegex, unittestEndRegex, source)
			if err != nil {
				return nil, err
			}
			filename := ""
			if m := testClassRegex.FindStringSubmatch(text); m != nil {
				// HelloTest will be stored into HelloTest.py.
				filename = m[1] + ".py"
			}
			if filename == "" {
				return nil, fmt.Errorf("could not detect the test name for unittest: %s", source)
			}
			var imports []string
			for _, m := range importRegex.FindAllStringSubmatch(text, -1) {
				imports = append(imports, "import "+m[1]+"\n")
			}
			text = strings.Join(imports, "") + text
			glog.V(3).Infof("metadata: %v, exercise_id: %q", exerciseMetadata, exerciseID)
			glog.V(3).Infof("parsed unit test: %s\n", text)
			return []*Cell{&Cell{
				Type:     "code",
				Metadata: cloneMetadata(exerciseMetadata, "filename", filename, "assignment_id", assignmentID),
				Source:   text,
			}}, nil
		} else if m := solutionMagicRegex.FindStringIndex(source); m != nil {
			clean, err := CleanForStudent(cell, assignmentMetadata, exerciseMetadata, AnyLanguage)
			if err != nil {
				return nil, err
			}
			// Store the untouched source cell value.
			return []*Cell{
				// empty_source.py is an easy way to access empty submission content
				// from python code by referencing empty_source.source.
				&Cell{
					Type:     "code",
					Metadata: cloneMetadata(exerciseMetadata, "filename", "empty_source.py", "assignment_id", assignmentID),
					Source:   `source = """` + strings.Replace(clean.Source, `"""`, `\"\"\"`, -1) + `"""`,
				},
				// empty_submission.py is a plain file containing the empty submission
				// content as is. This is easier to read from Go server.
				&Cell{
					Type:     "code",
					Metadata: cloneMetadata(exerciseMetadata, "filename", "empty_submission.py", "assignment_id", assignmentID),
					Source:   clean.Source,
				},
			}, nil
		} else {
			// For every non-solution and non-inline test code cell, add it to global
			// or exercise context (for inline tests).
			if exerciseID == "" {
				// Before the first exercise, append to global context.
				globalContext = append(globalContext, cell)
			} else {
				// After an exercise ID is set, append to the exercise context.
				exerciseContext = append(exerciseContext, cell)
			}
		}
		// Extract the reporter into a script. The reporter script takes the JSON on the standard input,
		// expecting 'results' field to contain an outcome dictionary, and 'logs' field to contain the
		// dictionary of test logs, keyed by the test name.
		if m := templateRegex.FindStringSubmatchIndex(source); m != nil {
			// Extract the template name.
			name := source[m[2]:m[3]]
			filename := name + ".py"
			// Cut the magic string.
			source = source[m[1]:]
			return []*Cell{&Cell{
				Type:     "code",
				Metadata: cloneMetadata(exerciseMetadata, "filename", filename, "assignment_id", assignmentID),
				Source: `
import jinja2
import json
import sys
import submission_source
import pygments
from pygments import lexers
from pygments import formatters

template = """` + source + `"""

if __name__ == '__main__':
  input = sys.stdin.read()
  data = json.loads(input)
  source = submission_source.source
  formatted_source = pygments.highlight(source, lexers.PythonLexer(), formatters.HtmlFormatter())
  tmpl = jinja2.Template(template)
  sys.stdout.write(tmpl.render(results=data['results'], formatted_source=formatted_source, logs=data['logs']))
`,
			}}, nil
		}
		// Do not emit other code cells.
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	transformed.Metadata = assignmentMetadata
	return transformed, nil
}
