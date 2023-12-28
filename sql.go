package itertools

import (
	"context"
	"database/sql"
)

// SqlConn is an interface that represents a SQL connection.
type SqlConn interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// SqlRowScanner is an interface that represents a SQL row scanner.
type SqlRowScanner interface {
	Scan(dest ...any) error
}

// SqlQueryContext holds information about a SQL query,
// including the last error, context, database connection, query string, and arguments.
type SqlQueryContext struct {
	lastErr error
	ctx     context.Context
	db      SqlConn
	query   string
	args    []any
}

// LastErr returns the last error encountered during the execution of the SQL query.
func (q *SqlQueryContext) LastErr() error {
	return q.lastErr
}

// SqlQuery creates a new SqlQueryContext with the default context and provided database connection, query string, and arguments.
func SqlQuery(db SqlConn, query string, args ...any) *SqlQueryContext {
	return SqlQueryWithContext(context.Background(), db, query, args...)
}

// SqlQueryWithContext creates a new SqlQueryContext with the provided context, database connection, query string, and arguments.
func SqlQueryWithContext(ctx context.Context, db SqlConn, query string, args ...any) *SqlQueryContext {
	return &SqlQueryContext{
		ctx:   ctx,
		db:    db,
		query: query,
		args:  args,
	}
}

// SqlRows generates a sequence of values by executing a SQL query and scanning the result rows using the provided scanner function.
func SqlRows[T any](q *SqlQueryContext, scanner func(row SqlRowScanner, dest *T) error) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		rows, err := q.db.QueryContext(q.ctx, q.query, q.args...)
		if err != nil {
			q.lastErr = err
			return
		}
		defer func() {
			if err := rows.Close(); err != nil {
				q.lastErr = err
			}
		}()

		for rows.Next() {
			var v T
			if err := scanner(rows, &v); err != nil {
				q.lastErr = err
				return
			}

			yield(v)
		}
	})
}
