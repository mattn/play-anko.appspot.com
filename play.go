package play

import (
	"crypto/sha1"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattn/anko/core"
	"github.com/mattn/anko/env"
	_ "github.com/mattn/anko/packages"
	"github.com/mattn/anko/parser"
	"github.com/mattn/anko/vm"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

type Record struct {
	Code string
}

var (
	t      = template.Must(template.ParseFiles("tmpl/index.tpl"))
	commit string
)

func init() {
	b, err := ioutil.ReadFile("VERSION")
	if err != nil {
		panic(err)
	}
	commit = strings.TrimSpace(string(b))

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
	e := env.NewEnv()

	core.Import(e)

	e.Define("println", func(a ...interface{}) {
		fmt.Fprint(w, fmt.Sprintln(a...))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	e.Define("print", func(a ...interface{}) {
		fmt.Fprint(w, fmt.Sprint(a...))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	e.Define("printf", func(a string, b ...interface{}) {
		fmt.Fprintf(w, fmt.Sprintf(a, b...))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	e.Define("panic", func(a ...interface{}) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Can't use panic()")
		return
	})
	e.Define("load", func(a ...interface{}) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Can't use load()")
		return
	})
	_, err = vm.Run(e, nil, stmts)
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

	err := t.Execute(w, &struct {
		Code        string
		Commit      string
		CommitShort string
	}{
		Code:        code,
		Commit:      commit,
		CommitShort: commit[:8],
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
