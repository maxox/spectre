package main

import (
	"bytes"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/knieriem/markdown"
	"html/template"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type Language struct {
	Title, Name         string   `json:",omitempty"`
	Formatter           string   `json:"-"`
	Names               []string `json:",omitempty"`
	Extensions          []string `json:"-"`
	MIMETypes           []string `json:"-" yaml:"mimetypes"`
	DisplayStyle        string   `json:"-" yaml:"display_style"`
	SuppressLineNumbers bool     `json:"-" yaml:"suppress_line_numbers"`
}

type LanguageList []*Language

func (l LanguageList) Len() int {
	return len(l)
}

func (l LanguageList) Less(i, j int) bool {
	return l[i].Title < l[j].Title
}

func (l LanguageList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func LanguageNamed(name string) *Language {
	v, ok := languageConfig.languageMap[name]
	if !ok {
		return unknownLanguage
	}
	return v
}

type _LanguageConfiguration struct {
	LanguageGroups []*struct {
		Title     string
		Languages LanguageList
	} `yaml:"languageGroups"`
	Formatters map[string]*Formatter

	languageMap  map[string]*Language
	modtime      time.Time
	languageJSON []byte
}

var languageConfig _LanguageConfiguration
var unknownLanguage *Language = &Language{
	Title:     "Unknown",
	Name:      "unknown",
	Formatter: "text",
}

type FormatFunc func(*Formatter, io.Reader, ...string) (string, error)

type Formatter struct {
	Name string
	Func string
	Env  []string
	Args []string
	fn   FormatFunc
}

func (f *Formatter) Format(stream io.Reader, lang string) (string, error) {
	myargs := make([]string, len(f.Args))
	for i, v := range f.Args {
		n := v
		if n == "%LANG%" {
			n = lang
		}
		myargs[i] = n
	}
	return f.fn(f, stream, myargs...)
}

func commandFormatter(formatter *Formatter, stream io.Reader, args ...string) (output string, err error) {
	var outbuf, errbuf bytes.Buffer
	command := exec.Command(args[0], args[1:]...)
	command.Stdin = stream
	command.Stdout = &outbuf
	command.Stderr = &errbuf
	command.Env = formatter.Env
	err = command.Run()
	output = strings.TrimSpace(outbuf.String())
	if err != nil {
		output = strings.TrimSpace(errbuf.String())
	}
	return
}

func markdownFormatter(formatter *Formatter, stream io.Reader, args ...string) (string, error) {
	buf := &bytes.Buffer{}
	markdownParser := markdown.NewParser(&markdown.Extensions{
		Smart:      true,
		FilterHTML: true,
	})
	markdownParser.Markdown(stream, markdown.ToHTML(buf))
	return buf.String(), nil
}

func plainTextFormatter(formatter *Formatter, stream io.Reader, args ...string) (string, error) {
	buf := &bytes.Buffer{}
	io.Copy(buf, stream)
	return template.HTMLEscapeString(buf.String()), nil
}

var formatFunctions map[string]FormatFunc = map[string]FormatFunc{
	"commandFormatter": commandFormatter,
	"plainText":        plainTextFormatter,
	"markdown":         markdownFormatter,
}

func FormatPaste(p *Paste) (string, error) {
	var formatter *Formatter
	var ok bool
	if formatter, ok = languageConfig.Formatters[p.Language.Formatter]; !ok {
		formatter = languageConfig.Formatters["default"]
	}

	reader, _ := p.Reader()
	defer reader.Close()
	return formatter.Format(reader, p.Language.Name)
}

func loadLanguageConfig() {
	languageConfig = _LanguageConfiguration{}

	err := YAMLUnmarshalFile("languages.yml", &languageConfig)
	if err != nil {
		panic(err)
	}

	languageConfig.languageMap = make(map[string]*Language)
	for _, g := range languageConfig.LanguageGroups {
		for _, v := range g.Languages {
			languageConfig.languageMap[v.Name] = v
			for _, langname := range v.Names {
				languageConfig.languageMap[langname] = v
			}
		}
		sort.Sort(g.Languages)
	}

	for _, v := range languageConfig.Formatters {
		v.fn = formatFunctions[v.Func]
	}
	glog.Info("Loaded ", len(languageConfig.languageMap), " languages.")
	glog.Info("Loaded ", len(languageConfig.Formatters), " formatters.")

	fi, _ := os.Stat("languages.yml")
	languageConfig.modtime = fi.ModTime()
	languageConfig.languageJSON, _ = json.Marshal(languageConfig.LanguageGroups)
}

func init() {
	RegisterTemplateFunction("langByLexer", LanguageNamed)

	RegisterReloadFunction(loadLanguageConfig)
}