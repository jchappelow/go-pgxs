# https://www.postgresql.org/docs/current/extend-pgxs.html
EXTENSION = example
DATA = example--0.1.sql
MODULES = example

# Include /usr/lib/postgresql/16/lib/pgxs/src/makefiles/pgxs.mk
PG_CONFIG = pg_config
PGXS := $(shell $(PG_CONFIG) --pgxs)
override with_llvm = no
include $(PGXS)

INCLUDEDIR_SERVER := $(shell $(PG_CONFIG) --includedir-server)

$(addsuffix $(DLSUFFIX), $(EXTENSION)): *.c *.go Makefile
	echo Building $@
	CGO_CFLAGS="-I$(INCLUDEDIR_SERVER) -I"/opt/homebrew/include" -fpic -Wno-int-conversion -Wno-incompatible-pointer-types" \
		go build -v -buildmode=c-shared -o $@
