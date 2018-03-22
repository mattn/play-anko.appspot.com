package play

import (
	"crypto/sha1"
	"fmt"
	"html/template"
	"net/http"

	"github.com/mattn/anko/core"
	"github.com/mattn/anko/packages"
	"github.com/mattn/anko/parser"
	"github.com/mattn/anko/vm"

	"appengine"
	"appengine/datastore"
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

	core.Import(env)
	packages.DefineImport(env)

	env.Define("println", func(a ...interface{}) {
		fmt.Fprint(w, fmt.Sprintln(a...))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	env.Define("print", func(a ...interface{}) {
		fmt.Fprint(w, fmt.Sprint(a...))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	env.Define("printf", func(a string, b ...interface{}) {
		fmt.Fprintf(w, fmt.Sprintf(a, b...))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	env.Define("panic", func(a ...interface{}) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Can't use panic()")
		return
	})
	env.Define("load", func(a ...interface{}) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Can't use load()")
		return
	})
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