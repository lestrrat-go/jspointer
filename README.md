# go-jspointer

[![Build Status](https://travis-ci.org/lestrrat/go-jspointer.svg?branch=master)](https://travis-ci.org/lestrrat/go-jspointer)

[![GoDoc](https://godoc.org/github.com/lestrrat/go-jspointer?status.svg)](https://godoc.org/github.com/lestrrat/go-jspointer)

JSON pointer for Go

# Features

* Compile and match against Maps, Slices, Structs
* Set values in each of those

# Usage

```go
p, _ := jspointer.New(`/foo/bar/baz`)
result, _ := p.Get(someStruct)
```

# Credits

This is almost a fork of https://github.com/xeipuuv/gojsonpointer.
