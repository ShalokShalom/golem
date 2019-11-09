//
// Copyright (C) 2019 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/golem
//

// Golem is a tool to instantiate a specific type from generic definition.
// The absence of generics in Go causes the usage of `go generate` to re-write
// abstract definition at build time, like this:
//
//   //go:generate golem -type Foo -generic github.com/fogfish/golem/stream/stream.go
//
// The command takes few arguments:
//
//   -type string   defines a parametrization to generic type.
//
//   -generic path  locates a path to generic algorithm.
//
// The command creates a file in same directory containing a parametrized definition
// of generic type.
//
// Install
//
//   go get -u github.com/fogfish/golem/cmd/golem
//
// Generics
//
// The library uses any type `interface{}` to implement valid generic Go code.
// Any other language uses a type variables to express generic types, e.g. `Stack[T]`.
// This Go library uses `genT` type aliases instead of variable for this purpose
//
//   package stack
//
//   type genT interface{}
//
//   type AnyT struct {
//	   elements []genT
//   }
//
//   func (s AnyT) push(x genT) {/* ... */}
//   func (s AnyT) pop() genT {/* ... */}
//
// Any one is able to use this generic types directly or its its parametrized version
//
//   stack.AnyT{}
//   stack.Int{}
//   stack.String{}
//
// The unsafe type definitions are replaced with a specied type, each literal `genT`
// and `AnyT` is substitute with value derived from specified type. A few replacement
// modes are supported, these modes takes an advantage of Go package naming schema
// and provides intuitive approach to reference generated types, e.g.
//
//   stack.Int{}     // generics in library
//   foobar.Stack{}  // generics in application
//
// Library
//
// As a generic library developer I want to define a generic type and supply its
// parametrized variants of standard Go type so that my generic is ready for
// application development.
// The mode implies a following rules
//
// ↣ one package defines one generic type.
//
// ↣ concrete types are named after the type, `AnyT` is replaced with `Type`
// (e.g `AnyT` -> `Int`).
//
// ↣ type alias `genT` is repaced with `genType`
// (e.g `genT` -> `genInt`).
//
// ↣ file type.go is created in the package
// (e.g. `int.go`)
//
// Application
//
// As a application developer I want to parametrise a generic types with my own
// application specific types so that the application benefits from re-use of
// generic implementations
// The mode implies a following rules
//
// ↣ one package implements various generic variants for the custom type
//
// ↣ concrete types are named after the generic, `AnyT` is replaced with `Generic`
// (e.g `AnyT` -> `Stack`).
//
// ↣ type alias `genT` is repaced with `genType`
// (e.g `genT` -> `genFooBar`).
//
// ↣ file generic.go is created in the package
// (e.g. `stack.go`)
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

//
type opts struct {
	kind    *string
	generic *string
	lib     *bool
}

func parseOpts() opts {
	spec := opts{
		flag.String("type", "", "defines a parametrization to generic type."),
		flag.String("generic", "", "locates a path to generic type."),
		flag.Bool("lib", false, "use library declaration schema."),
	}
	flag.Parse()
	return spec
}

//
func declareType(file []byte, kind string) []byte {
	a := bytes.Replace(file,
		[]byte("type genT interface{}"),
		[]byte(fmt.Sprintf("type gen%s %s", strings.Title(kind), kind)),
		1,
	)
	b := bytes.ReplaceAll(a,
		[]byte("genT"),
		[]byte(fmt.Sprintf("gen%s", strings.Title(kind))),
	)
	return b
}

//
func referenceType(file []byte, kind string) []byte {
	return bytes.ReplaceAll(file,
		[]byte("AnyT"),
		[]byte(kind),
	)
}

//
func repackage(file []byte, pkg string) []byte {
	re := regexp.MustCompile(`package (.*)\n`)
	return re.ReplaceAll(file, []byte("package "+pkg+"\n"))
}

//
func main() {
	var err error
	log.SetFlags(0)
	log.SetPrefix("==> golem: ")
	opt := parseOpts()

	pkg, err := build.Default.ImportDir(".", 0)
	if err != nil {
		log.Fatal(err)
	}

	source := filepath.Join(build.Default.GOPATH, "src", *opt.generic)
	generic := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))

	filename := fmt.Sprintf("%s.go", generic)
	typename := strings.Title(generic)
	if *opt.lib {
		filename = fmt.Sprintf("%s.go", *opt.kind)
		typename = strings.Title(*opt.kind)
	}

	input, err := ioutil.ReadFile(source)
	if err != nil {
		log.Fatal(err)
	}

	a := declareType(input, *opt.kind)
	b := referenceType(a, typename)
	c := repackage(b, pkg.Name)

	output := bytes.NewBuffer([]byte{})
	output.Write([]byte("// Code generated by `golem` package\n"))
	output.Write([]byte(fmt.Sprintf("// Source: %s\n", *opt.generic)))
	output.Write([]byte(fmt.Sprintf("// Time: %s\n\n", time.Now().UTC())))

	output.Write(c)

	ioutil.WriteFile(filepath.Join(pkg.PkgRoot, filename), output.Bytes(), 0777)
	log.Printf("%s.%s", generic, typename)
}
