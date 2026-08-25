package database

import (
	"reflect"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	got := splitStatements("create table users (id integer);\n-- northframe:split\ninsert into users (id) values (1);")
	want := []string{"create table users (id integer);", "insert into users (id) values (1);"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitStatements() = %#v, want %#v", got, want)
	}
}

func TestPlaceholderUsesSelectedDialect(t *testing.T) {
	if got := placeholder(PostgreSQL, 2); got != "$2" {
		t.Fatalf("PostgreSQL placeholder = %q, want $2", got)
	}
	if got := placeholder(MySQL, 2); got != "?" {
		t.Fatalf("MySQL placeholder = %q, want ?", got)
	}
	if got := placeholder(SQLite, 2); got != "?" {
		t.Fatalf("SQLite placeholder = %q, want ?", got)
	}
}
