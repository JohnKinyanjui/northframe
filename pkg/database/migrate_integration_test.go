package database_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/JohnKinyanjui/northframe/pkg/database"
	"github.com/JohnKinyanjui/northframe/pkg/database/sqlite"
)

func TestSQLiteMigrationIsAppliedExactlyOnce(t *testing.T) {
	db, err := sqlite.Open("file:migration-test?mode=memory&cache=shared", database.Pool{MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"001_users.sql": {Data: []byte("create table users (id integer primary key, name text not null);-- northframe:split insert into users (name) values ('Amina');")},
	}
	for range 2 {
		if err := database.Migrate(context.Background(), db, database.SQLite, migrations); err != nil {
			t.Fatal(err)
		}
	}

	var users int
	if err := db.QueryRow("select count(1) from users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 1 {
		t.Fatalf("user count = %d, want 1", users)
	}
}
