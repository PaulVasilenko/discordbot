package smileystats

import "database/sql"

type DB interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}
