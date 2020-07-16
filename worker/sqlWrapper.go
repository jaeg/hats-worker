package worker

import (
	"database/sql"

	"github.com/robertkrimen/otto"

	//This is how you import sql drivers
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

// SQLWrapper wrapper struct
type SQLWrapper struct {
	db *sql.DB
}

func newSQLWrapper(connectionString string, driverName string) *SQLWrapper {
	db, err := sql.Open(driverName, connectionString)
	sw := &SQLWrapper{db: db}
	if err != nil {
		log.WithError(err).Error("Failed to open db")
	} else {
		err = sw.Ping()
	}

	return sw
}

// Ping - library implementation of sql pings
func (sw *SQLWrapper) Ping() error {
	err := sw.db.Ping()
	if err != nil {
		log.WithError(err).Error("Failed to ping db")
	}
	return err
}

// Close - library implementation of sql close
func (sw *SQLWrapper) Close() error {
	err := sw.db.Close()
	if err != nil {
		log.WithError(err).Error("Failed to close db")
	}
	return err
}

//Exec - library implementation of sql queries
func (sw *SQLWrapper) Exec(query string, args ...interface{}) otto.Value {
	statement, err := sw.db.Prepare(query)
	if err != nil {
		log.WithError(err).Error("Failed to prepare statement")
		return otto.UndefinedValue()
	}

	defer statement.Close()
	res, err := statement.Exec(args...)
	if err != nil {
		log.WithError(err).Error("Error executing query")
		return otto.UndefinedValue()
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.WithError(err).Error("Error getting rows affected")
		return otto.UndefinedValue()
	}

	value, _ := otto.ToValue(rows)
	return value
}

// Query - library implementation of sql queries
func (sw *SQLWrapper) Query(query string, args ...interface{}) []map[string]otto.Value {
	outputRows := make([]map[string]otto.Value, 0)
	statement, err := sw.db.Prepare(query)
	if err != nil {
		log.WithError(err).Error("Failed to prepare statement")
		return nil
	}

	defer statement.Close()
	rows, err := statement.Query(args...)
	if err != nil {
		log.WithError(err).Error("Failed to query db")
		return nil
	}

	cols, err := rows.Columns()
	if err != nil {
		log.WithError(err).Error("Failed to get columns")
		return nil
	}

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			log.WithError(err).Error("Failed to scan")
			return nil
		}

		m := make(map[string]otto.Value)
		for i, colName := range cols {
			byteArray, ok := columns[i].([]byte)
			if ok {
				ottoVal, _ := otto.ToValue(string(byteArray))
				m[colName] = ottoVal
			} else {
				ottoVal, _ := otto.ToValue(columns[i])
				m[colName] = ottoVal
			}
		}

		outputRows = append(outputRows, m)
	}
	return outputRows
}
