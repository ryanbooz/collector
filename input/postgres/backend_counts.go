package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pganalyze/collector/state"
)

const backendCountsSQL string = `
 SELECT datid,
				usesysid,
				COALESCE(state, 'unknown'),
				COALESCE(backend_type, 'unknown'), COALESCE(wait_event_type, '') = 'Lock' AS waiting_for_lock,
				pg_catalog.count(*)
	 FROM %s
	GROUP BY 1, 2, 3, 4, 5`

func GetBackendCounts(ctx context.Context, c *Collection, db *sql.DB) ([]state.PostgresBackendCount, error) {
	var sourceTable string

	if c.HelperExists("get_stat_activity", nil) {
		sourceTable = "pganalyze.get_stat_activity()"
	} else {
		sourceTable = "pg_catalog.pg_stat_activity"
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(backendCountsSQL, sourceTable))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var backendCounts []state.PostgresBackendCount

	for rows.Next() {
		var row state.PostgresBackendCount

		err := rows.Scan(&row.DatabaseOid, &row.RoleOid, &row.State, &row.BackendType,
			&row.WaitingForLock, &row.Count)
		if err != nil {
			return nil, err
		}

		// Special case to avoid errors for certain backends with weird names
		if c.Config.SystemType == "azure_database" {
			row.BackendType = strings.ToValidUTF8(row.BackendType, "")
		}

		backendCounts = append(backendCounts, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return backendCounts, nil
}
