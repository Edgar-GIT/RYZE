// Command mysql_to_postgres copies every row of the RYZE MariaDB database into
// a PostgreSQL database whose schema was created by the backend migrations
// (`go run . migrate up`).
//
// It is the one-off data step of the MariaDB -> PostgreSQL migration and is
// kept so the copy can be repeated (for example on another environment):
//
//   - The source is read inside one consistent snapshot.
//   - The target is written in one transaction: every application table is
//     emptied (TRUNCATE) and refilled in foreign-key order, then the row count
//     of every table is compared with the source before COMMIT. Any mismatch
//     or error rolls the whole copy back.
//   - Generated columns, views and the migration bookkeeping table are never
//     copied; PostgreSQL recomputes the generated columns.
//
// Connection settings come from the environment (the repository .env is
// loaded by scripts/migrate_mysql_to_postgres.sh): host, database, user and
// password from DB_*, overridable per side with SOURCE_DB_* and TARGET_DB_*;
// ports from the -source-port and -target-port flags. Credentials are never
// printed.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const batchSize = 200

type endpoint struct {
	host, port, name, user, password, sslmode string
}

func env(prefix, key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(prefix + key)); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func loadEndpoint(prefix, defaultPort string) endpoint {
	return endpoint{
		host:     env(prefix, "DB_HOST", "127.0.0.1"),
		port:     defaultPort,
		name:     env(prefix, "DB_NAME", ""),
		user:     env(prefix, "DB_USER", ""),
		password: os.Getenv(firstSet(prefix+"DB_PASSWORD", "DB_PASSWORD")),
		sslmode:  env(prefix, "DB_SSLMODE", "disable"),
	}
}

func firstSet(keys ...string) string {
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); ok {
			return key
		}
	}
	return keys[len(keys)-1]
}

type column struct {
	name     string
	dataType string // information_schema data_type of the target column
}

func main() {
	sourcePort := flag.String("source-port", "3306", "MariaDB port")
	targetPort := flag.String("target-port", "5432", "PostgreSQL port")
	targetName := flag.String("target-db", "", "PostgreSQL database (default: DB_NAME / TARGET_DB_NAME)")
	dryRun := flag.Bool("dry-run", false, "copy and verify, then roll back instead of committing")
	flag.Parse()

	source := loadEndpoint("SOURCE_", *sourcePort)
	target := loadEndpoint("TARGET_", *targetPort)
	if *targetName != "" {
		target.name = *targetName
	}
	if source.name == "" || source.user == "" || target.name == "" || target.user == "" {
		log.Fatal("DB_NAME and DB_USER (or the SOURCE_/TARGET_ variants) must be set")
	}

	ctx := context.Background()
	if err := run(ctx, source, target, *dryRun); err != nil {
		log.Fatalf("copy failed, nothing was committed: %v", err)
	}
}

func run(ctx context.Context, source, target endpoint, dryRun bool) error {
	srcCfg := mysql.NewConfig()
	srcCfg.User = source.user
	srcCfg.Passwd = source.password
	srcCfg.Net = "tcp"
	srcCfg.Addr = net.JoinHostPort(source.host, source.port)
	srcCfg.DBName = source.name
	srcCfg.ParseTime = true
	srcCfg.Loc = time.UTC
	srcDB, err := sql.Open("mysql", srcCfg.FormatDSN())
	if err != nil {
		return errors.New("cannot open the MariaDB source")
	}
	defer srcDB.Close()
	srcConn, err := srcDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("cannot connect to the MariaDB source %s:%s/%s: %w", source.host, source.port, source.name, err)
	}
	defer srcConn.Close()

	pgCfg, err := pgx.ParseConfig(fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		quote(target.host), quote(target.port), quote(target.name), quote(target.user), quote(target.password), quote(target.sslmode)))
	if err != nil {
		return errors.New("invalid PostgreSQL target configuration")
	}
	dst, err := pgx.ConnectConfig(ctx, pgCfg)
	if err != nil {
		return fmt.Errorf("cannot connect to the PostgreSQL target %s:%s/%s: %w", target.host, target.port, target.name, err)
	}
	defer dst.Close(ctx)

	if _, err := srcConn.ExecContext(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ"); err != nil {
		return fmt.Errorf("source isolation: %w", err)
	}
	if _, err := srcConn.ExecContext(ctx, "START TRANSACTION WITH CONSISTENT SNAPSHOT, READ ONLY"); err != nil {
		return fmt.Errorf("source snapshot: %w", err)
	}
	defer srcConn.ExecContext(ctx, "ROLLBACK") //nolint:errcheck

	var srcVersion int64
	var srcDirty bool
	if err := srcConn.QueryRowContext(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&srcVersion, &srcDirty); err != nil {
		return fmt.Errorf("read source migration version: %w", err)
	}
	var dstVersion int64
	var dstDirty bool
	if err := dst.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&dstVersion, &dstDirty); err != nil {
		return fmt.Errorf("read target migration version (run `go run . migrate up` against PostgreSQL first): %w", err)
	}
	if srcDirty || dstDirty {
		return fmt.Errorf("a migration is dirty (source=%v, target=%v)", srcDirty, dstDirty)
	}
	if dstVersion < srcVersion {
		return fmt.Errorf("target schema version %d is older than the source version %d", dstVersion, srcVersion)
	}
	log.Printf("source schema version %d, target schema version %d", srcVersion, dstVersion)

	tables, err := targetTables(ctx, dst)
	if err != nil {
		return err
	}
	sourceTables, err := mysqlTables(ctx, srcConn, source.name)
	if err != nil {
		return err
	}

	// Tables that exist only in PostgreSQL (added after the move, e.g. the
	// support history, sign-in identities, account tokens and the email outbox)
	// cannot be refilled from MariaDB. Emptying them would destroy data, so a
	// re-run against a database that already holds such rows is refused.
	for _, table := range tables {
		if sourceTables[table] {
			continue
		}
		var hasRows bool
		if err := dst.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM "+pgx.Identifier{table}.Sanitize()+")").Scan(&hasRows); err != nil {
			return fmt.Errorf("inspect target table %s: %w", table, err)
		}
		if hasRows {
			return fmt.Errorf("target table %s holds data that does not exist in MariaDB; refusing to empty it (copy into a fresh database instead)", table)
		}
	}

	tx, err := dst.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin target transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		quoted = append(quoted, pgx.Identifier{table}.Sanitize())
	}
	if _, err := tx.Exec(ctx, "TRUNCATE "+strings.Join(quoted, ", ")); err != nil {
		return fmt.Errorf("empty target tables: %w", err)
	}

	type report struct {
		table          string
		source, target int64
		skipped        bool
	}
	var reports []report
	for _, table := range tables {
		if !sourceTables[table] {
			reports = append(reports, report{table: table, skipped: true})
			continue
		}
		columns, err := copyColumns(ctx, tx, srcConn, source.name, table)
		if err != nil {
			return err
		}
		copied, err := copyTable(ctx, srcConn, tx, table, columns)
		if err != nil {
			return err
		}
		var sourceCount, targetCount int64
		if err := srcConn.QueryRowContext(ctx, "SELECT COUNT(*) FROM `"+table+"`").Scan(&sourceCount); err != nil {
			return fmt.Errorf("count source %s: %w", table, err)
		}
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM "+pgx.Identifier{table}.Sanitize()).Scan(&targetCount); err != nil {
			return fmt.Errorf("count target %s: %w", table, err)
		}
		if sourceCount != targetCount || copied != sourceCount {
			return fmt.Errorf("row count mismatch in %s: source %d, copied %d, target %d", table, sourceCount, copied, targetCount)
		}
		reports = append(reports, report{table: table, source: sourceCount, target: targetCount})
	}

	var total int64
	for _, r := range reports {
		if r.skipped {
			log.Printf("%-48s not in source (left empty)", r.table)
			continue
		}
		total += r.target
		log.Printf("%-48s %6d rows (source %d)", r.table, r.target, r.source)
	}

	if dryRun {
		log.Printf("dry run: %d rows copied and verified, rolled back", total)
		return nil
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	log.Printf("committed: %d rows copied into %s", total, target.name)
	return nil
}

func quote(value string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value) + "'"
}

// targetTables lists the application tables of the target in foreign-key
// order (parents first). The golang-migrate bookkeeping table is excluded.
func targetTables(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE' AND table_name <> 'schema_migrations'
		ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("list target tables: %w", err)
	}
	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	rows, err = conn.Query(ctx, `
		SELECT DISTINCT child.relname, parent.relname
		FROM pg_constraint c
		JOIN pg_class child ON child.oid = c.conrelid
		JOIN pg_class parent ON parent.oid = c.confrelid
		JOIN pg_namespace n ON n.oid = child.relnamespace
		WHERE c.contype = 'f' AND n.nspname = 'public' AND child.relname <> parent.relname`)
	if err != nil {
		return nil, fmt.Errorf("list foreign keys: %w", err)
	}
	parents := map[string][]string{}
	for rows.Next() {
		var child, parent string
		if err := rows.Scan(&child, &parent); err != nil {
			return nil, err
		}
		parents[child] = append(parents[child], parent)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ordered := make([]string, 0, len(tables))
	state := map[string]int{} // 1 visiting, 2 done
	var visit func(string) error
	visit = func(table string) error {
		switch state[table] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("foreign-key cycle at %s", table)
		}
		state[table] = 1
		deps := parents[table]
		sort.Strings(deps)
		for _, parent := range deps {
			if err := visit(parent); err != nil {
				return err
			}
		}
		state[table] = 2
		ordered = append(ordered, table)
		return nil
	}
	for _, table := range tables {
		if err := visit(table); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func mysqlTables(ctx context.Context, conn *sql.Conn, schema string) (map[string]bool, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE'`, schema)
	if err != nil {
		return nil, fmt.Errorf("list source tables: %w", err)
	}
	defer rows.Close()
	tables := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables[name] = true
	}
	return tables, rows.Err()
}

// copyColumns returns the target's insertable columns that also exist in the
// source table. Generated target columns are skipped; a source column the
// target no longer has is reported as an error rather than silently dropped.
func copyColumns(ctx context.Context, tx pgx.Tx, src *sql.Conn, schema, table string) ([]column, error) {
	rows, err := tx.Query(ctx, `
		SELECT column_name, data_type, is_generated FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, fmt.Errorf("target columns of %s: %w", table, err)
	}
	type targetColumn struct {
		dataType  string
		generated bool
	}
	targetCols := map[string]targetColumn{}
	var order []string
	for rows.Next() {
		var name, dataType, generated string
		if err := rows.Scan(&name, &dataType, &generated); err != nil {
			return nil, err
		}
		targetCols[name] = targetColumn{dataType: dataType, generated: generated == "ALWAYS"}
		order = append(order, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	srcRows, err := src.QueryContext(ctx, `
		SELECT column_name, extra FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("source columns of %s: %w", table, err)
	}
	defer srcRows.Close()
	sourceCols := map[string]bool{}
	for srcRows.Next() {
		var name, extra string
		if err := srcRows.Scan(&name, &extra); err != nil {
			return nil, err
		}
		generated := strings.Contains(strings.ToUpper(extra), "GENERATED")
		if _, ok := targetCols[name]; !ok && !generated {
			return nil, fmt.Errorf("source column %s.%s has no target column", table, name)
		}
		sourceCols[name] = true
	}
	if err := srcRows.Err(); err != nil {
		return nil, err
	}

	var columns []column
	for _, name := range order {
		target := targetCols[name]
		if target.generated || !sourceCols[name] {
			continue
		}
		columns = append(columns, column{name: name, dataType: target.dataType})
	}
	return columns, nil
}

func copyTable(ctx context.Context, src *sql.Conn, tx pgx.Tx, table string, columns []column) (int64, error) {
	names := make([]string, len(columns))
	sourceNames := make([]string, len(columns))
	for i, col := range columns {
		names[i] = pgx.Identifier{col.name}.Sanitize()
		sourceNames[i] = "`" + col.name + "`"
	}
	rows, err := src.QueryContext(ctx, "SELECT "+strings.Join(sourceNames, ", ")+" FROM `"+table+"`")
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", table, err)
	}
	defer rows.Close()

	insertPrefix := "INSERT INTO " + pgx.Identifier{table}.Sanitize() + " (" + strings.Join(names, ", ") + ") VALUES "
	var batch [][]any
	var copied int64
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		var sb strings.Builder
		args := make([]any, 0, len(batch)*len(columns))
		sb.WriteString(insertPrefix)
		for i, row := range batch {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("(")
			for j := range row {
				if j > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(&sb, "$%d", len(args)+1)
				args = append(args, row[j])
			}
			sb.WriteString(")")
		}
		if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
			return fmt.Errorf("insert into %s: %w", table, err)
		}
		copied += int64(len(batch))
		batch = batch[:0]
		return nil
	}

	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return 0, fmt.Errorf("scan %s: %w", table, err)
		}
		row := make([]any, len(columns))
		for i, col := range columns {
			converted, err := convert(values[i], col.dataType)
			if err != nil {
				return 0, fmt.Errorf("%s.%s: %w", table, col.name, err)
			}
			row[i] = converted
		}
		batch = append(batch, row)
		if len(batch) == batchSize {
			if err := flush(); err != nil {
				return 0, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read %s: %w", table, err)
	}
	if err := flush(); err != nil {
		return 0, err
	}
	return copied, nil
}

// convert maps a value read by the MySQL driver onto the Go type pgx encodes
// for the target column.
func convert(value any, dataType string) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch dataType {
	case "boolean":
		switch v := value.(type) {
		case int64:
			return v != 0, nil
		case []byte:
			return string(v) != "0" && string(v) != "", nil
		}
	case "jsonb", "json":
		if v, ok := value.([]byte); ok {
			if !json.Valid(v) {
				return nil, errors.New("invalid JSON value")
			}
			return json.RawMessage(append([]byte(nil), v...)), nil
		}
	case "numeric":
		var n pgtype.Numeric
		switch v := value.(type) {
		case []byte:
			if err := n.Scan(string(v)); err != nil {
				return nil, err
			}
			return n, nil
		case float64:
			if err := n.Scan(fmt.Sprintf("%v", v)); err != nil {
				return nil, err
			}
			return n, nil
		}
	case "timestamp with time zone":
		if v, ok := value.(time.Time); ok {
			return v.UTC(), nil
		}
	case "character varying", "text", "character", "USER-DEFINED":
		switch v := value.(type) {
		case []byte:
			return string(v), nil
		case string:
			return v, nil
		}
	case "smallint", "integer", "bigint":
		switch v := value.(type) {
		case int64:
			return v, nil
		case uint64:
			return int64(v), nil
		case []byte:
			return strconv.ParseInt(string(v), 10, 64)
		}
	}
	return nil, fmt.Errorf("unexpected source value %T for target type %s", value, dataType)
}
