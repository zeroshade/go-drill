package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type data struct {
	In interface{}
	D  listValue
}

type listValue map[string]string

func (l listValue) String() string {
	res := make([]string, 0, len(l))
	for k, v := range l {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(res, ", ")
}

func (l listValue) Set(v string) error {
	nv := strings.Split(v, "=")
	if len(nv) != 2 {
		return fmt.Errorf("expected NAME=VALUE, got %s", v)
	}
	l[nv[0]] = nv[1]
	return nil
}

func parsePath(path string) (string, string) {
	p := strings.IndexByte(path, '=')
	if p == -1 {
		if filepath.Ext(path) != ".tmpl" {
			log.Fatalf("template file '%s' must have .tmpl extension", path)
		}
		return path, path[:len(path)-len(".tmpl")]
	}

	return path[:p], path[p+1:]
}

func main() {
	var (
		dataArg = flag.String("data", "", "input JSON data")
		in      = &data{D: make(listValue)}
	)

	flag.Var(&in.D, "d", "-d NAME=VALUE")
	flag.Parse()
	if *dataArg == "" {
		log.Fatal("data option is required")
	}

	paths := flag.Args()
	if len(paths) == 0 {
		log.Fatal("no tmpl files specified")
	}

	in.In = readData(*dataArg)
	process(in, paths)
}

func readData(path string) interface{} {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Read Data: ", err)
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		log.Fatal("Unmarshal: ", err)
	}
	return v
}

func process(data interface{}, paths []string) {
	for _, p := range paths {
		var (
			t   *template.Template
			err error
		)

		in, out := parsePath(p)

		contents, _ := ioutil.ReadFile(in)
		t, err = template.New("gen").Parse(string(contents))
		if err != nil {
			log.Fatal("Template Parse: ", err)
		}

		var buf bytes.Buffer
		fmt.Fprintf(&buf, "// Code generated by %s. DO NOT EDIT.\n", p)

		err = t.Execute(&buf, data)
		if err != nil {
			log.Fatal("Tmpl Execute: ", err)
		}

		generated := buf.Bytes()
		generated, err = format.Source(generated)
		if err != nil {
			log.Fatal("Format: ", err)
		}

		ioutil.WriteFile(out, generated, os.ModePerm)
	}
}
