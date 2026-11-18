// Package testdb gives each test its own real, migrated and seeded Postgres
// database.
//
// Point TEST_DATABASE_URL at a Postgres server where the user may create
// databases. Locally, with the dev stack's database running
// (docker compose -f local/compose.yaml up -d db):
//
//	TEST_DATABASE_URL=postgres://liftlog:liftlog@localhost:5433/postgres?sslmode=disable go test ./...
//
// Without it, tests that need a database are skipped, except under CI
// (CI is set) where they fail, so they can't silently stop running.
//
// The first New in a test binary creates a template database, migrates and
// seeds it; every New clones that template (CREATE DATABASE ... TEMPLATE),
// so each test starts from the same clean state and tests can run in
// parallel. Packages that use New call Main from TestMain so the template
// is dropped when the package's tests finish.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/Uranury/oci-platform/internal/db"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/internal/seed"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnvVar names the admin connection string.
const EnvVar = "TEST_DATABASE_URL"

var (
	tmplOnce sync.Once
	tmplName string
	tmplErr  error
	// Serialises clones of the template within this process; Postgres
	// refuses to copy a database another session is copying or connected to.
	cloneMu sync.Mutex
)

// Main runs the package's tests, then drops the template database. Use it
// as the body of TestMain in every package that calls New.
func Main(m *testing.M) {
	code := m.Run()
	if tmplName != "" {
		if err := adminExec(context.Background(), os.Getenv(EnvVar), "DROP DATABASE IF EXISTS "+tmplName+" WITH (FORCE)"); err != nil {
			fmt.Fprintln(os.Stderr, "testdb: drop template:", err)
		}
	}
	os.Exit(code)
}

// New returns a pool on a fresh database cloned from the migrated, seeded
// template. The database is dropped when the test ends.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	admin := os.Getenv(EnvVar)
	if admin == "" {
		if os.Getenv("CI") != "" {
			t.Fatalf("%s is not set", EnvVar)
		}
		t.Skipf("%s not set; see internal/testdb", EnvVar)
	}
	tmplOnce.Do(func() { tmplName, tmplErr = createTemplate(admin) })
	if tmplErr != nil {
		t.Fatalf("testdb: template: %v", tmplErr)
	}

	ctx := context.Background()
	name := "liftlog_test_" + randHex()
	cloneMu.Lock()
	err := adminExec(ctx, admin, "CREATE DATABASE "+name+" TEMPLATE "+tmplName)
	cloneMu.Unlock()
	if err != nil {
		t.Fatalf("testdb: clone: %v", err)
	}

	pool, err := pgxpool.New(ctx, withDatabase(admin, name))
	if err != nil {
		t.Fatalf("testdb: connect: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		if err := adminExec(ctx, admin, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Logf("testdb: drop %s: %v", name, err)
		}
	})
	return pool
}

// User creates a user and returns its ID. The password hash is a
// placeholder: tests that log in go through AuthService instead.
func User(t testing.TB, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	tag := randHex()
	if err := repo.NewUser(pool).Create(context.Background(), id, "u"+tag, tag+"@example.test", "not-a-real-hash"); err != nil {
		t.Fatalf("testdb: create user: %v", err)
	}
	return id
}

func createTemplate(admin string) (string, error) {
	ctx := context.Background()
	name := "liftlog_tmpl_" + randHex()
	if err := adminExec(ctx, admin, "CREATE DATABASE "+name); err != nil {
		return "", err
	}
	dsn := withDatabase(admin, name)
	if err := db.Migrate(dsn); err != nil {
		return name, err
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return name, err
	}
	defer pool.Close() // no connections may remain for it to be cloned
	return name, seed.Sync(ctx, dbutil.NewTxProvider(pool))
}

func adminExec(ctx context.Context, admin, sql string) error {
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(ctx) }()
	_, err = conn.Exec(ctx, sql)
	return err
}

// withDatabase swaps the database name in a postgres:// URL.
func withDatabase(dsn, name string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		panic(fmt.Sprintf("testdb: %s is not a URL: %v", EnvVar, err))
	}
	u.Path = "/" + name
	return u.String()
}

// randHex is lower-case hex, safe unquoted in a database name.
func randHex() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
