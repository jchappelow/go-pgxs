// Package main is an example PostgreSQL C language extension, written using cgo
// to create a shared library from Go functions.
package main

/*
#cgo CFLAGS: -I"/usr/include/postgresql/16/server" -fpic
#cgo LDFLAGS: -shared
#cgo darwin LDFLAGS: -undefined dynamic_lookup

// C.Datum from postgres.h
#include "postgres.h"

// C.FunctionCallBaseData
#include "fmgr.h"

// #include "utils/elog.h"
// extern void elog_error(char* string);
*/
import "C"
import (
	"log"
	"strings"
	"unsafe"

	"github.com/jchappelow/go-pgxs"
)

type funcInfo = C.FunctionCallInfoBaseData
type datum = C.Datum

// NOTE: if we used pgxs.Datum in the function signature, the compiler would not
// allow it because it is a "Go type". It's in a different packages, so it
// understandably does no realize that they are the same structure.

//export Hello
func Hello(fcinfo *funcInfo) datum {
	logger := pgxs.NewNoticeLogger("", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Println("hello")
	return (datum)(0) // datum(pgxs.ToDatum(nil))
}

func convFI(fcinfo *funcInfo) *pgxs.FuncInfo {
	return (*pgxs.FuncInfo)(unsafe.Pointer(fcinfo))
}

//export JoinStrings
func JoinStrings(fcinfo *funcInfo) datum {
	var strs []string
	fi := convFI(fcinfo)
	err := fi.Scan(&strs)
	if err != nil {
		pgxs.LogError(err.Error())
		return datum(pgxs.ToDatum(""))
	}

	ret := strings.Join(strs, "")
	return datum(pgxs.ToDatum(ret))
}

func main() {} // required with -buildmode=c-shared
