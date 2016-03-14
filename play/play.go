package play

import (
	"appengine"
	"appengine/datastore"
	"crypto/sha1"
	"fmt"
	"github.com/mattn/anko/parser"
	"github.com/mattn/anko/vm"
	"html/template"
	"net/http"
	"reflect"

	anko_core "github.com/mattn/anko/builtins"
	anko_encoding_json "github.com/mattn/anko/builtins/encoding/json"
	anko_flag "github.com/mattn/anko/builtins/flag"
	anko_fmt "github.com/mattn/anko/builtins/fmt"
	anko_io "github.com/mattn/anko/builtins/io"
	anko_io_ioutil "github.com/mattn/anko/builtins/io/ioutil"
	anko_math "github.com/mattn/anko/builtins/math"
	anko_net "github.com/mattn/anko/builtins/net"
	anko_net_http "github.com/mattn/anko/builtins/net/http"
	anko_net_url "github.com/mattn/anko/builtins/net/url"
	anko_os "github.com/mattn/anko/builtins/os"
	anko_os_exec "github.com/mattn/anko/builtins/os/exec"
	anko_path "github.com/mattn/anko/builtins/path"
	anko_path_filepath "github.com/mattn/anko/builtins/path/filepath"
	anko_regexp "github.com/mattn/anko/builtins/regexp"
	anko_sort "github.com/mattn/anko/builtins/sort"
	anko_strings "github.com/mattn/anko/builtins/strings"

	anko_colortext "github.com/mattn/anko/builtins/github.com/daviddengcn/go-colortext"
)

type Record struct {
	Code string
}

var t = template.Must(template.ParseFiles("tmpl/index.tpl"))

func init() {
	http.HandleFunc("/api/play", serveApiPlay)
	http.HandleFunc("/api/save", serveApiSave)
	http.HandleFunc("/p/", servePermalink)
	http.HandleFunc("/", servePermalink)
}

func serveApiSave(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	h := sha1.New()
	fmt.Fprintf(h, "%s", code)
	hid := fmt.Sprintf("%x", h.Sum(nil))
	c := appengine.NewContext(r)
	key := datastore.NewKey(c, "Anko", hid, 0, nil)
	_, err := datastore.Put(c, key, &Record{code})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", key.StringID())
}

func serveApiPlay(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	scanner := new(parser.Scanner)
	scanner.Init(code)
	stmts, err := parser.Parse(scanner)
	if e, ok := err.(*parser.Error); ok {
		w.WriteHeader(500)
		fmt.Fprintf(w, "%d: %s\n", e.Pos.Line, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	env := vm.NewEnv()

	anko_core.Import(env)

	tbl := map[string]func(env *vm.Env) *vm.Env{
		"encoding/json": anko_encoding_json.Import,
		"flag":          anko_flag.Import,
		"fmt":           anko_fmt.Import,
		"io":            anko_io.Import,
		"io/ioutil":     anko_io_ioutil.Import,
		"math":          anko_math.Import,
		"net":           anko_net.Import,
		"net/http":      anko_net_http.Import,
		"net/url":       anko_net_url.Import,
		"os":            anko_os.Import,
		"os/exec":       anko_os_exec.Import,
		"path":          anko_path.Import,
		"path/filepath": anko_path_filepath.Import,
		"regexp":        anko_regexp.Import,
		"sort":          anko_sort.Import,
		"strings":       anko_strings.Import,
		"github.com/daviddengcn/go-colortext": anko_colortext.Import,
	}

	env.Define("import", reflect.ValueOf(func(s string) interface{} {
		if loader, ok := tbl[s]; ok {
			return loader(env)
		}
		panic(fmt.Sprintf("package '%s' not found", s))
	}))

	env.Define("println", reflect.ValueOf(func(a ...interface{}) {
		fmt.Fprint(w, fmt.Sprintln(a...))
	}))
	env.Define("print", reflect.ValueOf(func(a ...interface{}) {
		fmt.Fprint(w, fmt.Sprint(a...))
	}))
	env.Define("prinf", reflect.ValueOf(func(a string, b ...interface{}) {
		fmt.Fprintf(w, fmt.Sprintf(a, b...))
	}))
	env.Define("panic", reflect.ValueOf(func(a ...interface{}) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Can't use panic()")
		return
	}))
	env.Define("load", reflect.ValueOf(func(a ...interface{}) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Can't use load()")
		return
	}))
	defer env.Destroy()
	_, err = vm.Run(stmts, env)
	if err != nil {
		w.WriteHeader(500)
		if e, ok := err.(*vm.Error); ok {
			fmt.Fprintf(w, "%d: %s\n", e.Pos.Line, err)
		} else if e, ok := err.(*parser.Error); ok {
			fmt.Fprintf(w, "%d: %s\n", e.Pos.Line, err)
		} else {
			fmt.Fprintln(w, e.Error())
		}
	}
}

func servePermalink(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	var code string
	if len(path) > 3 {
		id := path[3:]
		c := appengine.NewContext(r)
		var record Record
		err := datastore.Get(c, datastore.NewKey(c, "Anko", id, 0, nil), &record)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		code = record.Code
	} else {
		code = `var fmt = import("fmt")

println(fmt.Sprintf("こんにちわ世界 %05d", 123))`
	}

	err := t.Execute(w, &struct{ Code string }{Code: code})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
