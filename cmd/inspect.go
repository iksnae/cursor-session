package cmd

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

var (
	inspectFormat     string
	inspectSampleRows int
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect [database-path]",
	Short: "Inspect database schema and structure",
	Long: `Inspect the schema and structure of Cursor storage databases.

This command provides detailed information about:
  • Database schema (tables, columns, types)
  • Sample data from each table
  • Row counts and statistics
  • Differences between desktop and agent storage formats

Examples:
  cursor-session inspect                                    # Auto-detect and inspect
  cursor-session inspect --storage /path/to/store.db       # Inspect specific database
  cursor-session inspect --format json --sample 5          # JSON output with 5 sample rows`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var dbPath string
		if len(args) > 0 {
			dbPath = args[0]
		}

		// If no path provided, try to auto-detect
		if dbPath == "" {
			paths, err := internal.GetStoragePaths(storagePath)
			if err != nil {
				return fmt.Errorf("failed to detect storage: %w", err)
			}

			// Try desktop storage first
			if paths.GlobalStorageExists() {
				dbPath = paths.GetGlobalStorageDBPath()
				fmt.Printf("📊 Inspecting desktop storage: %s\n\n", dbPath)
			} else if paths.HasAgentStorage() {
				// Get first agent storage database
				storeDBs, err := paths.FindAgentStoreDBs()
				if err != nil || len(storeDBs) == 0 {
					return fmt.Errorf("no agent storage databases found")
				}
				dbPath = storeDBs[0]
				fmt.Printf("📊 Inspecting agent storage: %s\n\n", dbPath)
			} else {
				return fmt.Errorf("no storage found - use --storage to specify a database path")
			}
		}

		return inspectDatabase(dbPath)
	},
}

func inspectDatabase(dbPath string) error {
	db, err := internal.OpenDatabase(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Get all tables
	tables, err := getTables(db)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	if len(tables) == 0 {
		fmt.Println("⚠️  No tables found in database")
		return nil
	}

	fmt.Printf("📋 Database: %s\n", dbPath)
	fmt.Printf("📊 Found %d table(s)\n\n", len(tables))

	for _, tableName := range tables {
		if err := inspectTable(db, tableName); err != nil {
			fmt.Printf("⚠️  Error inspecting table %s: %v\n", tableName, err)
			continue
		}
		fmt.Println()
	}

	return nil
}

func getTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func inspectTable(db *sql.DB, tableName string) error {
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📦 Table: %s\n", tableName)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Get row count
	var rowCount int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount); err != nil {
		return fmt.Errorf("failed to get row count: %w", err)
	}
	fmt.Printf("📊 Rows: %d\n\n", rowCount)

	// Get schema
	columns, err := getTableSchema(db, tableName)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	fmt.Printf("📐 Schema:\n")
	for _, col := range columns {
		pk := ""
		if col.PrimaryKey {
			pk = " [PRIMARY KEY]"
		}
		notNull := ""
		if col.NotNull {
			notNull = " NOT NULL"
		}
		fmt.Printf("  • %s: %s%s%s\n", col.Name, col.Type, notNull, pk)
	}
	fmt.Println()

	// Show sample data
	if rowCount > 0 && inspectSampleRows > 0 {
		if err := showSampleData(db, tableName, columns, inspectSampleRows); err != nil {
			fmt.Printf("⚠️  Error showing sample data: %v\n", err)
		}
	}

	return nil
}

type ColumnInfo struct {
	Name       string
	Type       string
	NotNull    bool
	PrimaryKey bool
}

func getTableSchema(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var cid int
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &col.Name, &col.Type, &notNull, &defaultValue, &pk); err != nil {
			continue
		}
		col.NotNull = notNull == 1
		col.PrimaryKey = pk == 1
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

func showSampleData(db *sql.DB, tableName string, columns []ColumnInfo, limit int) error {
	if len(columns) == 0 {
		return nil
	}

	// Build column list
	colNames := make([]string, len(columns))
	for i, col := range columns {
		colNames[i] = col.Name
	}

	query := fmt.Sprintf("SELECT %s FROM %s LIMIT %d", strings.Join(colNames, ", "), tableName, limit)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	fmt.Printf("📄 Sample Data (first %d rows):\n", limit)
	rowNum := 0
	for rows.Next() {
		rowNum++
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			fmt.Printf("  ⚠️  Row %d: error scanning: %v\n", rowNum, err)
			continue
		}

		fmt.Printf("\n  Row %d:\n", rowNum)
		for i, col := range columns {
			val := values[i]
			var valStr string
			if val == nil {
				valStr = "<NULL>"
			} else {
				valStr = fmt.Sprintf("%v", val)

				// Special handling for meta table value column (hex-encoded JSON)
				if tableName == "meta" && col.Name == "value" && valStr != "" {
					// Try to decode as hex and parse as JSON
					if decoded, err := hex.DecodeString(valStr); err == nil {
						var metaData map[string]interface{}
						if json.Unmarshal(decoded, &metaData) == nil {
							// Successfully decoded - show formatted JSON
							if jsonBytes, err := json.MarshalIndent(metaData, "      ", "  "); err == nil {
								fmt.Printf("    %s (hex-encoded JSON):\n%s\n", col.Name, string(jsonBytes))
								continue
							}
						}
					}
				}

				// Truncate long values
				if len(valStr) > 200 {
					valStr = valStr[:200] + "..."
				}
				// Show first line only for multi-line values
				if strings.Contains(valStr, "\n") {
					valStr = strings.Split(valStr, "\n")[0] + "..."
				}
			}
			fmt.Printf("    %s: %s\n", col.Name, valStr)
		}
	}

	return rows.Err()
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().StringVar(&inspectFormat, "format", "text", "Output format (text, json)")
	inspectCmd.Flags().IntVar(&inspectSampleRows, "sample", 3, "Number of sample rows to show")
}
