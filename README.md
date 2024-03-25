# Go PostgreSQL C language extension

## Background: What can we do with an extension?

- create custom functions
- handle triggers (relation, old data, new data => do something)
- use SPI (server programming interface) to run queries on databases

## `gopgxs`

PostgreSQL provides a build infrastructure for extensions called "PGXS",
supporting either pure SQL or C language extensions. Other languages are
supported with wrappers. This package (`gopgxs`) aims to provide importable
types and functions for writing a PostgreSQL extension in the Go language.

The code in pgext.go is based on gitlab.com/microo8/plgo, with several
modifications primarily so that it can just be imported by an extension project
without using a generation tool to actually copy the boilerplate into each
extension project. The original plgp package would also generate wrapper code
and SQL initialization scripts for vanilla Go functions, which we do ourselves
to be aware of the conversions and understand the calling convention.

The extension in the `example` folder may be used as a template to create a new
extension. The rest of this README describes how.

## Define the Shared Library (.so)

We will create:

- `example.go` containing our Go exported functions and some boilerplate to work with `go-pgxs` and export our functions to C libraries
- `example.c` to register the functions defined in `example.go` with the PostgreSQL function manager

To build a PostgreSQL extension using the `go-pgxs` package:

- Create a new `package main` with the following boilerplate from `example.go`:

    ```go
    // example.go
    package main

    /*
    #cgo LDFLAGS: -shared
    #cgo darwin LDFLAGS: -undefined dynamic_lookup

    // C.Datum from postgres.h
    #include "postgres.h"

    // C.FunctionCallBaseData
    #include "fmgr.h"
    */
    import "C"
    import (
        "log"
        "strings"
        "unsafe"

        "github.com/jchappelow/go-pgxs"
    )

    func main() {} // required with -buildmode=c-shared

    type funcInfo = C.FunctionCallInfoBaseData // input
    type datum = C.Datum // return
    ```

- Implement a Go function that accepts a `*funcInfo` and return a `datum`. Also, export it with a comment.
  For instance:

    ```go
    // still example.go

    //export JoinStrings
    func JoinStrings(fcinfo *funcInfo) datum { 
    ```

- Use the `pgxs` helpers to retrieve input values from the `*funcInfo`, and to create a `datum` from Go typed variables in the function.

  - Get a `pgxs.FuncInfo` from the input `*funcInfo`. Use the appropriate method for for accessing the input arguments from the C struct: `Scan`, `CalledAsTrigger`, and `TriggerData`. For example, see the `JoinStrings` function in example.go:

    ```go
      var strs []string
      fi := convFI(fcinfo)  // (*pgxs.FuncInfo)(unsafe.Pointer(fcinfo))
      err := fi.Scan(&strs) // TEXT[] => []string
      if err != nil { ... }
    ```
  
  - Perform whatever operations required of the function. For this `JoinStrings` example:

    ```go
    ret := strings.Join(strs, "")
    ```

  - Prepare the returned `datum` using `pgx.ToDatum`. For example, given a `ret` value of type `string`: `return datum(pgxs.ToDatum(ret))`.

    ```go
    ret := strings.Join(strs, "")
    return datum(pgxs.ToDatum(ret))
    ```

  **NOTE:** using `//export` requires that the function signature use only C types, requiring the extra type conversions above.

- Create a "module magic" C source file, as in `example.c`.  This must follow this simple template with the names of the functions listed in `PG_FUNCTION_INFO_V1` declarations:

  ```c
  // example.c
  #include "postgres.h"
  #include "fmgr.h"

  PG_MODULE_MAGIC;

  // After the above required boilerplate, register each of the exported Go
  // functions using PG_FUNCTION_INFO_V1.

  PG_FUNCTION_INFO_V1(HelloWorld);
  PG_FUNCTION_INFO_V1(JoinStrings);
  ...
  ```

  Here we are using the `PG_FUNCTION_INFO_V1` macro provided by the postgres headers to register our `HelloWorld` and `JoinStrings` functions.

**Alternative to importing `pgxs`** and using it's types, you can also copy the entire file into your extension package, and change it to `package main`. Then you can use the types directly with the inputs and return. However, you will need to update this file if there are revisions to `gopgxs`. As such is it preferable to import a revision of `gopgx` with go.mod.

## Build the Shared Library (.so)

With an `example.go` and `example.c` we can create the shared library, which is the extension itself.

```sh
CGO_CFLAGS="-I$(pg_config --includedir-server) -fpic" \
    go build -v -buildmode=c-shared -o example.so
```

Note that we specify custom `CFLAGS` to use the postgres headers. Also, all `.c` files including `example.c` are included in the build.

In the next section, we install the on a postgres server.

## Adding the Extension to a Database

For demonstrative purposes, you would add this extension to a database on a `postgres` instance with SQL:

```sql
CREATE OR REPLACE FUNCTION hello()
  RETURNS void AS '/path/to/example.so','HelloWorld'
  LANGUAGE C STRICT;
```

Connect to the database and "create" the extension:

```sql
CREATE EXTENSION example;
```

Then you can use the function from SQL or pl/pgSQL:

```sql
SELECT hello();
```

or

```sql
DO $$
BEGIN
    PERFORM hello();
END;
$$ LANGUAGE plpgsql;
```

The above is walkthrough of a basic manually-created extension and its functions. The following sections describe a proper system installation.

## Installing the Extension Shared Library

To permanently install an extension, the shared library and an SQL initialization script would be installed in the PostgreSQL system paths, requiring only a `CREATE EXTENSION ...` command within a database to use it. The folder containing all installed extension `.so` files is returned by `pg_config --pkglibdir`, which for PostgreSQL 16 would typically be `/usr/lib/postgresql/16/lib` (have a look at the many already installed).

First, we we create two additional files: `example--0.1.sql` is the SQL script containing the `CREATE OR REPLACE FUNCTION` statements to run when the extension is created, and `example.control` sets a few basic properties of the extension. Copy and modify these files as needed for you extension.

## Initialization SQL Script

In the folder specified by `$(pg_config --sharedir)/extension`, which for PostgreSQL 16 would typically be `/usr/share/postgresql/16/extension`, there are `.sql` files that initialize the extension functions exported by the extension shared libraries.

If the `example.so` file were installed to the postgres system library folder, the SQL script would use the `$libdir` variable:

```sql
-- /usr/share/postgresql/16/extension/example--0.1.sql
CREATE OR REPLACE FUNCTION hello()
  RETURNS void AS '$libdir/example','HelloWorld'
  LANGUAGE C STRICT;
CREATE OR REPLACE FUNCTION join_strings(strs text[])
  RETURNS TEXT AS '$libdir/example','JoinStrings'
  LANGUAGE C STRICT;
```

The initialization script can contain many such `CREATE OR REPLACE FUNCTION` statements for each of the exported functions. Define the input arguments and `RETURNS` types as required, using `PostgreSQL` types (and arrays of types, etc.).

## Makefile Helper

Since you got here, you are now granted permissions to use the Makefile infrastructure to automate the building and installation of the extension.

In the `example` extension, there is a `Makefile` that includes the PostgreSQL `pgxs.mk` extension build framework Makefile. It will install the extension library and initialization SQL script into the correct paths determined automatically with the `pg_config` utility).

```sh
$ make clean
rm -f example.so example.o  \
    example.bc

$ make
CGO_CFLAGS="-I/usr/include/postgresql/16/server -fpic" \
    go build -v -buildmode=c-shared -o example.so
runtime/cgo
github.com/jchappelow/go-pgxs
github.com/jchappelow/go-pgxs/example

$ sudo make install
/bin/mkdir -p '/usr/share/postgresql/16/extension'
/bin/mkdir -p '/usr/share/postgresql/16/extension'
/bin/mkdir -p '/usr/lib/postgresql/16/lib'
/usr/bin/install -c -m 644 .//example.control '/usr/share/postgresql/16/extension/'
/usr/bin/install -c -m 644 .//example--0.1.sql  '/usr/share/postgresql/16/extension/'
/usr/bin/install -c -m 755  example.so '/usr/lib/postgresql/16/lib/'
```

The extension library is installed to the path from `pg_config --pkglibdir`, and the `.control` and `.sql` files are installed to the path from `pg_config --sharedir`.

Then you can connect to the database and do `CREATE EXTENSION example` and begin using the functions.

If you want to rebuild the extension, be sure to first `DROP EXTENSION example` **and disconnect** before doing the `make install` step.
