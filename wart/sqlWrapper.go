package wart

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"
	"github.com/robertkrimen/otto"

	//This is how you import sql drivers
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

type SQLWrapper struct {
	db *sqlx.DB
}

func newSQLWrapper(connectionString string, driverName string) *SQLWrapper {
	db, err := sqlx.Open(driverName, connectionString)
	sw := &SQLWrapper{db: db}
	if err != nil {
		log.WithError(err).Error("Failed to open db")
	} else {
		err = sw.Ping()
	}

	return sw
}

func (sw *SQLWrapper) Ping() error {
	err := sw.db.Ping()
	if err != nil {
		log.WithError(err).Error("Failed to ping db")
	}
	return err
}

func (sw *SQLWrapper) close() error {
	err := sw.db.Close()
	if err != nil {
		log.WithError(err).Error("Failed to close db")
	}
	return err
}

func (sw *SQLWrapper) Query(query string, args ...interface{}) []map[string]otto.Value {

	rows, err := sw.db.Queryx(query, args...)
	if err != nil {
		log.WithError(err).Error("Failed to query db")
	}

	outputRows := make([]map[string]otto.Value, 0)

	//Figure out the types of the fields..
	typeMap := make(map[string]reflect.Type, 0)
	cols, err := rows.Columns()
	if err != nil {
		log.WithError(err).Error("Failed to get columns")
		return nil
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.WithError(err).Error("Failed to get columns")
		return nil
	}
	for i, v := range cols {
		typeMap[v] = colTypes[i].ScanType()
	}

	for rows.Next() {
		// cols, err := rows.Columns()
		// if err != nil {
		// 	log.WithError(err).Error("Failed to get columns")
		// 	break
		// }
		//values := make([]interface{}, len(cols))
		value := map[string]interface{}{}
		//err = rows.Scan(values...)
		err = rows.MapScan(value)
		if err != nil {
			log.WithError(err).Error("Failed to scan db")
			break
		}

		convertedValue := map[string]otto.Value{}

		for k, v := range value {
			t := typeMap[k]
			var cv otto.Value
			fmt.Println("Type ", t.String())
			switch t.String() {
			case "int32":
				cv, _ = otto.ToValue(v.(int32))
			case "sql.NullInt64":
				cv, _ = otto.ToValue(v.(sql.NullInt64))
			case "sql.RawBytes":
				cv, _ = otto.ToValue(v.(sql.RawBytes))
			}

			convertedValue[k] = cv
		}
		outputRows = append(outputRows, convertedValue)
	}
	fmt.Println(outputRows)
	return outputRows
}
