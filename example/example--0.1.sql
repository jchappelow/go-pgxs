create or replace function join_strings(strs text[])
  returns text as '$libdir/example','JoinStrings'
  LANGUAGE C STRICT;

create or replace function hello()
  returns void as '$libdir/example','Hello'
  LANGUAGE C STRICT;
