package db

import (
	"encoding/json"
	"fmt"

	"github.com/bendy/file-gateway/internal/wasm"
)

// QueryRow executes a SQL query and returns a single row as a map.
func QueryRow(query string, args ...interface{}) (map[string]interface{}, error) {
	result, err := wasm.DBQuery(query, args)
	if err != nil {
		return nil, fmt.Errorf("db query: %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result[0], nil
}

// QueryRows executes a SQL query and returns all rows as maps.
func QueryRows(query string, args ...interface{}) ([]map[string]interface{}, error) {
	result, err := wasm.DBQuery(query, args)
	if err != nil {
		return nil, fmt.Errorf("db query: %w", err)
	}
	return result, nil
}

// Exec executes a SQL statement and returns the number of affected rows.
func Exec(query string, args ...interface{}) (int64, error) {
	rows, err := wasm.DBExec(query, args)
	if err != nil {
		return 0, fmt.Errorf("db exec: %w", err)
	}
	return rows, nil
}

// ScanRow unmarshals a single row JSON into the target struct.
func ScanRow(row map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("db scan marshal: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("db scan unmarshal: %w", err)
	}
	return nil
}

// ScanRows unmarshals multiple rows JSON into the target slice.
func ScanRows(rows []map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("db scan marshal: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("db scan unmarshal: %w", err)
	}
	return nil
}
