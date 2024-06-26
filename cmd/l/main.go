package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/dayvonjersen/git4go"
	"github.com/dayvonjersen/linguist"
)

func checkErr(err error) {
	if err != nil {
		if output_debug {
			log.Panicln(err)
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

// flag vars
var (
	input_mode_git          bool
	input_mode_fs           bool
	input_git_tree          string
	output_json             bool
	output_json_with_colors bool
	output_limit            int
	output_debug            bool
	unignore_filenames      bool
	unignore_contents       bool
)

// write a function to create a map with string keys
func makeMap(results []*language) map[string]*language {
	m := make(map[string]*language)
	for i := 0; i < len(results); i++ {
		m[results[i].Language] = results[i]
	}
	return m
}

// used for displaying results
type (
	language struct {
		Language string  `json:"language"`
		Percent  float64 `json:"percent"`
		Percentage string `json:"percentage"`
		Size int `json:"size"`
	}

	language_color struct {
		Language string  `json:"language"`
		Percent  float64 `json:"percent"`
		Color    string  `json:"color"`
	}
)

type sortableResult []*language

func (s sortableResult) Len() int {
	return len(s)
}

func (s sortableResult) Less(i, j int) bool {
	return s[i].Percent < s[j].Percent
}

func (s sortableResult) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

var (
	langs         map[string]int = make(map[string]int)
	total_size    int            = 0
	num_files     int            = 0
	max_len       int            = 0
	ignored_paths int            = 0
)

func putResult(language string, size int) {
	langs[language] += size
	total_size += size
	num_files++
	if len(language) > max_len {
		max_len = len(language)
	}
}
func pluralize(num int) string {
	if num == 1 {
		return ""
	}
	return "s"
}

func main() {
	flag.BoolVar(
		&output_debug,
		"debug", false,
		"Print debug information.",
	)
	flag.BoolVar(
		&input_mode_git,
		"git", false,
		"Scan for files using git ls-tree and cat-file, rather than filesystem.",
	)
	flag.BoolVar(
		&input_mode_fs,
		"fs", false,
		"Scan for files using filesystem.",
	)
	flag.StringVar(
		&input_git_tree,
		"git-tree", "HEAD",
		"tree-ish root to scan. See also man git(1).",
	)
	flag.BoolVar(
		&output_json,
		"json", false,
		"Output results in JSON format.",
	)
	flag.BoolVar(
		&output_json_with_colors,
		"json-with-colors", false,
		"Output results in JSON format, including any HTML color codes defined for associated languages.",
	)
	flag.IntVar(
		&output_limit,
		"limit", 10,
		"Limit number of languages to n results. n <= 0 for unlimited.",
	)
	flag.BoolVar(
		&unignore_filenames,
		"unignore-filenames", false,
		"Do NOT skip processing ignored file types based on filename (NOT RECOMMENDED)",
	)
	flag.BoolVar(
		&unignore_contents,
		"unignore-contents", false,
		"Do NOT skip processing ignored file types based on contents (NOT RECOMMENDED)",
	)

	flag.Parse()

	output_json = output_json || output_json_with_colors

	if !output_debug {
		log.SetOutput(ioutil.Discard)
	}

	var (
		default_input_mode_git bool
		default_input_mode_fs  bool
	)

	if !input_mode_fs && findGitDir() { // side-effect: cd's to GIT_DIR!
		default_input_mode_git = true
		default_input_mode_fs = false
	} else {
		default_input_mode_git = false
		default_input_mode_fs = true
	}

	if !input_mode_git && !input_mode_fs {
		input_mode_git = default_input_mode_git
		input_mode_fs = default_input_mode_fs
	}

	if !input_mode_git && input_git_tree != "HEAD" {
		input_mode_git = true
		input_mode_fs = false
	}

	if input_mode_git && input_mode_fs {
		fmt.Println("Please choose one of -git or -fs as flags, but not both.")
		fmt.Println("You can omit the flags to get the default behavior,")
		fmt.Printf("which for the current directory is %s\n", func() string {
			switch {
			case default_input_mode_git:
				return "git"
			case default_input_mode_fs:
				return "fs"
			}
			return "undefined"
		}())
		os.Exit(1)
	}

	if input_mode_fs {
		initGitIgnore()
		processDir(".")
	}

	if input_mode_git {
		repo, err := git4go.OpenRepository(".")
		checkErr(err)
		ref, err := repo.DwimReference(input_git_tree)
		checkErr(err)
		resolved, err := ref.Resolve()
		checkErr(err)
		odb, err := repo.Odb()
		checkErr(err)
		processTree(repo, odb, resolved.Target(), []string{})
	}

	results := []*language{}
	for lang, size := range langs {
		percent := float64 (float64(size) / float64(total_size)) * 100.0
		results = append(results, &language{
			Language: lang,
			Percent: percent, 
                        Percentage: fmt.Sprintf("%.2f", percent),
			Size: size,
		})
	}

	sort.Sort(sort.Reverse(sortableResult(results)))

	if output_limit > 0 && len(results) > output_limit {
		other := &language{
			Language: "Other",
		}
		for i := output_limit; i < len(results); i++ {
			other.Percent += results[i].Percent
			other.Size += 0
		}
		results = append(results[0:output_limit], other)
	}

	if output_json {
		var (
			json_bytes []byte
			err        error
		)
		if output_json_with_colors {
			out := []*language_color{}
			for _, lang := range results {
				out = append(out, &language_color{lang.Language, lang.Percent, linguist.LanguageColor(lang.Language)})
			}
			json_bytes, err = json.MarshalIndent(out, "", "  ")
		} else {
			json_bytes, err = json.MarshalIndent(makeMap(results), "", "  ")
		}
		checkErr(err)
		fmt.Println(string(json_bytes))
		os.Exit(0)
	}
	fmtstr := fmt.Sprintf("%% %ds", max_len)
	fmtstr += ": %07.4f%%\n"

	for _, l := range results {
		fmt.Printf(fmtstr, l.Language, l.Percent)
	}

	fmt.Printf("\n%d language%s detected in %d file%s\n", len(results), pluralize(len(results)), num_files, pluralize(num_files))
	fmt.Printf("%d ignored path%s\n", ignored_paths, pluralize(ignored_paths))
}
