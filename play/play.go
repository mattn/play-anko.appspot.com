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
		fmt.Fprintf(w, e.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	env := vm.NewEnv()
	env.Define("println", reflect.ValueOf(func(a ...interface{}) {
		fmt.Fprint(w, fmt.Sprintln(a...))
	}))
	env.Define("print", reflect.ValueOf(func(a ...interface{}) {
		fmt.Fprint(w, fmt.Sprint(a...))
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
		code = `println("こんにちわ世界")`
	}

	err := t.Execute(w, &struct{ Code string }{Code: code})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
