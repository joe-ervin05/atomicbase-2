package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/atombasedev/atombase/config"
	"github.com/atombasedev/atombase/definitions"
	"github.com/atombasedev/atombase/tools"
)

// SelectJSON queries rows using JSON body format.
// POST /data/query/{table} with Prefer: operation=select
func (dao *DatabaseConnection) SelectJSON(ctx context.Context, relation string, query SelectQuery, includeCount bool) (SelectResult, error) {
	return dao.selectJSON(ctx, dao.Client, relation, query, includeCount)
}

func (dao *DatabaseConnection) selectJSON(ctx context.Context, exec Executor, relation string, query SelectQuery, includeCount bool) (SelectResult, error) {
	if err := tools.ValidateTableName(relation); err != nil {
		return SelectResult{}, err
	}

	table, err := dao.Schema.SearchTbls(relation)
	if err != nil {
		return SelectResult{}, err
	}

	var sqlQuery, groupBy, agg string
	var policyArgs []any

	// Check if this is a custom join query
	if len(query.Join) > 0 {
		// Parse and build custom join query
		cjq, err := dao.Schema.ParseCustomJoinQuery(relation, query)
		if err != nil {
			return SelectResult{}, err
		}
		policies, err := dao.compileCustomJoinPolicies(ctx, cjq)
		if err != nil {
			return SelectResult{}, err
		}

		sqlQuery, groupBy, agg, policyArgs, err = dao.Schema.BuildCustomJoinSelect(cjq, policies)
		if err != nil {
			return SelectResult{}, err
		}
	} else {
		// Parse select clause for implicit FK-based joins
		rel, err := ParseSelectFromJSON(query.Select, relation)
		if err != nil {
			return SelectResult{}, err
		}
		policies, err := dao.compileSelectPolicies(ctx, rel)
		if err != nil {
			return SelectResult{}, err
		}

		// Build SELECT query
		sqlQuery, agg, policyArgs, err = dao.Schema.buildSelect(rel, policies)
		if err != nil {
			return SelectResult{}, err
		}
	}

	// Build WHERE clause
	where, args, err := table.BuildWhereFromJSON(query.Where, dao.Schema)
	if err != nil {
		return SelectResult{}, err
	}
	args = append(args, policyArgs...)

	// Build query in correct SQL order: SELECT...FROM...JOIN + WHERE + GROUP BY
	baseQuery := sqlQuery + where + groupBy

	var result SelectResult

	// Get count if requested
	if includeCount {
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s)", baseQuery)
		countQuery, countArgs := applyPolicyCTE(countQuery, args, dao, strings.Contains(countQuery, "__ab_membership"))
		row := exec.QueryRowContext(ctx, countQuery, countArgs...)
		if err := row.Scan(&result.Count); err != nil {
			return SelectResult{}, err
		}
	}

	// Add ordering
	if query.Order != nil {
		order, err := table.BuildOrderFromJSON(query.Order)
		if err != nil {
			return SelectResult{}, err
		}
		baseQuery += order
	}

	// Handle pagination
	limit := config.Cfg.DefaultLimit
	if query.Limit != nil && *query.Limit >= 0 {
		limit = *query.Limit
	}
	if config.Cfg.MaxQueryLimit > 0 && (limit > config.Cfg.MaxQueryLimit || limit == 0) {
		limit = config.Cfg.MaxQueryLimit
	}

	offset := 0
	if query.Offset != nil && *query.Offset >= 0 {
		offset = *query.Offset
	}

	if limit > 0 {
		baseQuery += fmt.Sprintf("LIMIT %d ", limit)
	}
	if offset > 0 {
		baseQuery += fmt.Sprintf("OFFSET %d ", offset)
	}

	finalQuery := fmt.Sprintf("SELECT json_group_array(%s) AS data FROM (%s)", agg, baseQuery)
	finalQuery, args = applyPolicyCTE(finalQuery, args, dao, strings.Contains(finalQuery, "__ab_membership"))
	row := exec.QueryRowContext(ctx, finalQuery, args...)
	if err := row.Scan(&result.Data); err != nil {
		return SelectResult{}, err
	}

	return result, nil
}

// InsertJSON inserts a single row using JSON body format.
// POST /data/query/{table} (no Prefer header)
func (dao *DatabaseConnection) InsertJSON(ctx context.Context, relation string, req InsertRequest) ([]byte, error) {
	return dao.insertJSON(ctx, dao.Client, relation, req)
}

func (dao *DatabaseConnection) insertJSON(ctx context.Context, exec Executor, relation string, req InsertRequest) ([]byte, error) {
	if err := tools.ValidateTableName(relation); err != nil {
		return nil, err
	}

	table, err := dao.Schema.SearchTbls(relation)
	if err != nil {
		return nil, err
	}

	if len(req.Data) == 0 {
		return nil, errors.New("insert requires at least one row")
	}

	if len(req.Data[0]) == 0 {
		return nil, errors.New("insert rows must have at least one column")
	}
	policy, err := dao.compilePolicyWithNewAlias(ctx, relation, "insert", "__ab_new")
	if err != nil {
		return nil, err
	}

	// Build column list from first row - collect into slice for consistent ordering
	columns := make([]string, 0, len(req.Data[0]))
	for col := range req.Data[0] {
		if _, err := table.SearchCols(col); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	query, args := buildInsertSelectSQL("INSERT", relation, columns, req.Data, policy)
	if err := dao.validateInsertRows(ctx, exec, columns, req.Data, policy); err != nil {
		return nil, err
	}

	if len(req.Returning) > 0 {
		retQuery, err := table.BuildReturningFromJSON(req.Returning)
		if err != nil {
			return nil, err
		}
		query += retQuery
		query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)
		return dao.queryJSONWithExec(ctx, exec, query, args...)
	}

	query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)
	result, err := ExecContextWithRetry(ctx, exec, query, args...)
	if err != nil {
		return nil, err
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return json.Marshal(map[string]any{"last_insert_id": lastInsertId})
}

// InsertIgnoreJSON inserts row(s), ignoring conflicts.
// POST /data/query/{table} with Prefer: operation=insert,on-conflict=ignore
func (dao *DatabaseConnection) InsertIgnoreJSON(ctx context.Context, relation string, req InsertRequest) ([]byte, error) {
	return dao.insertIgnoreJSON(ctx, dao.Client, relation, req)
}

func (dao *DatabaseConnection) insertIgnoreJSON(ctx context.Context, exec Executor, relation string, req InsertRequest) ([]byte, error) {
	if err := tools.ValidateTableName(relation); err != nil {
		return nil, err
	}

	table, err := dao.Schema.SearchTbls(relation)
	if err != nil {
		return nil, err
	}

	if len(req.Data) == 0 {
		return nil, errors.New("insert requires at least one row")
	}

	if len(req.Data[0]) == 0 {
		return nil, errors.New("insert rows must have at least one column")
	}
	policy, err := dao.compilePolicyWithNewAlias(ctx, relation, "insert", "__ab_new")
	if err != nil {
		return nil, err
	}

	// Build column list from first row - collect into slice for consistent ordering
	columns := make([]string, 0, len(req.Data[0]))
	for col := range req.Data[0] {
		if _, err := table.SearchCols(col); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	query, args := buildInsertSelectSQL("INSERT OR IGNORE", relation, columns, req.Data, policy)
	if err := dao.validateInsertRows(ctx, exec, columns, req.Data, policy); err != nil {
		return nil, err
	}

	if len(req.Returning) > 0 {
		retQuery, err := table.BuildReturningFromJSON(req.Returning)
		if err != nil {
			return nil, err
		}
		query += retQuery
		query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)
		return dao.queryJSONWithExec(ctx, exec, query, args...)
	}

	query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)
	result, err := ExecContextWithRetry(ctx, exec, query, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return json.Marshal(map[string]any{"rows_affected": rowsAffected})
}

// UpsertJSON inserts multiple rows, updating on conflict.
// POST /data/query/{table} with Prefer: on-conflict=replace
func (dao *DatabaseConnection) UpsertJSON(ctx context.Context, relation string, req UpsertRequest) ([]byte, error) {
	return dao.upsertJSON(ctx, dao.Client, relation, req)
}

func (dao *DatabaseConnection) upsertJSON(ctx context.Context, exec Executor, relation string, req UpsertRequest) ([]byte, error) {
	if err := tools.ValidateTableName(relation); err != nil {
		return nil, err
	}

	table, err := dao.Schema.SearchTbls(relation)
	if err != nil {
		return nil, err
	}

	if len(req.Data) == 0 {
		return nil, errors.New("upsert requires at least one row")
	}

	if len(req.Data[0]) == 0 {
		return nil, errors.New("upsert rows must have at least one column")
	}
	policy, err := dao.compilePolicyWithNewAlias(ctx, relation, "insert", "__ab_new")
	if err != nil {
		return nil, err
	}

	// Collect columns into slice for consistent ordering
	columns := make([]string, 0, len(req.Data[0]))
	for col := range req.Data[0] {
		if _, err := table.SearchCols(col); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}
	if err := validateUpsertPrimaryKeys(table.Pk, req.Data); err != nil {
		return nil, err
	}

	query, args := buildInsertSelectSQL("INSERT", relation, columns, req.Data, policy)
	if err := dao.validateInsertRows(ctx, exec, columns, req.Data, policy); err != nil {
		return nil, err
	}
	if err := dao.validateUpsertRows(ctx, exec, relation, table, req.Data); err != nil {
		return nil, err
	}

	if len(table.Pk) == 0 {
		query += " ON CONFLICT(rowid) DO UPDATE SET "
	} else {
		pkCols := make([]string, len(table.Pk))
		for i, col := range table.Pk {
			pkCols[i] = fmt.Sprintf("[%s]", col)
		}
		query += fmt.Sprintf(" ON CONFLICT(%s) DO UPDATE SET ", strings.Join(pkCols, ", "))
	}

	for _, col := range columns {
		query += fmt.Sprintf("[%s] = excluded.[%s], ", col, col)
	}

	query = query[:len(query)-2] + " "

	if len(req.Returning) > 0 {
		retQuery, err := table.BuildReturningFromJSON(req.Returning)
		if err != nil {
			return nil, err
		}
		query += retQuery
		query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)
		return dao.queryJSONWithExec(ctx, exec, query, args...)
	}

	query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)
	result, err := ExecContextWithRetry(ctx, exec, query, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return json.Marshal(map[string]any{"rows_affected": rowsAffected})
}

func (dao *DatabaseConnection) validateInsertRows(ctx context.Context, exec Executor, columns []string, rows []map[string]any, policy definitions.CompiledPredicate) error {
	if policy.SQL == "" {
		return nil
	}

	sourceSQL, args := buildInsertSourceSQL(columns, rows)
	query := "SELECT COUNT(*) FROM " + sourceSQL + " WHERE NOT (" + policy.SQL + ")"
	args = append(args, policy.Args...)
	query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)

	var rejected int
	if err := exec.QueryRowContext(ctx, query, args...).Scan(&rejected); err != nil {
		return err
	}
	if rejected > 0 {
		return tools.UnauthorizedErr("request does not satisfy definition policy")
	}
	return nil
}

func (dao *DatabaseConnection) validateUpsertRows(ctx context.Context, exec Executor, relation string, table CacheTable, rows []map[string]any) error {
	if len(table.Pk) == 0 {
		return nil
	}

	for _, row := range rows {
		existsWhere, existsArgs, err := buildPrimaryKeyWhere(table.Pk, row)
		if err != nil {
			return err
		}

		existsQuery := fmt.Sprintf("SELECT COUNT(*) FROM [%s] %s", relation, existsWhere)
		var existing int
		if err := exec.QueryRowContext(ctx, existsQuery, existsArgs...).Scan(&existing); err != nil {
			return err
		}
		if existing == 0 {
			continue
		}

		rowPolicy, err := dao.compilePolicy(ctx, relation, "update", row)
		if err != nil {
			return err
		}

		authorizedWhere, authorizedArgs := appendPolicyWhere(existsWhere, append([]any(nil), existsArgs...), rowPolicy)
		authorizedQuery := fmt.Sprintf("SELECT COUNT(*) FROM [%s] %s", relation, authorizedWhere)
		authorizedQuery, authorizedArgs = applyPolicyCTE(authorizedQuery, authorizedArgs, dao, rowPolicy.NeedsMembershipCTE)

		var allowed int
		if err := exec.QueryRowContext(ctx, authorizedQuery, authorizedArgs...).Scan(&allowed); err != nil {
			return err
		}
		if allowed == 0 {
			return tools.UnauthorizedErr("request does not satisfy definition policy")
		}
	}

	return nil
}

func buildPrimaryKeyWhere(pkCols []string, row map[string]any) (string, []any, error) {
	parts := make([]string, 0, len(pkCols))
	args := make([]any, 0, len(pkCols))
	for _, pkCol := range pkCols {
		value, ok := row[pkCol]
		if !ok {
			return "", nil, fmt.Errorf("upsert requires primary key column %q", pkCol)
		}
		if value == nil {
			return "", nil, fmt.Errorf("upsert requires primary key column %q to be non-null", pkCol)
		}
		parts = append(parts, fmt.Sprintf("[%s] = ?", pkCol))
		args = append(args, value)
	}
	return "WHERE " + strings.Join(parts, " AND ") + " ", args, nil
}

func validateUpsertPrimaryKeys(pkCols []string, rows []map[string]any) error {
	if len(pkCols) == 0 {
		return nil
	}

	seen := make(map[string]int, len(rows))
	for i, row := range rows {
		keyValues := make([]any, len(pkCols))
		for j, pkCol := range pkCols {
			value, ok := row[pkCol]
			if !ok {
				return fmt.Errorf("upsert requires primary key column %q", pkCol)
			}
			if value == nil {
				return fmt.Errorf("upsert requires primary key column %q to be non-null", pkCol)
			}
			keyValues[j] = value
		}

		key, err := primaryKeyValuesKey(keyValues)
		if err != nil {
			return err
		}
		if first, ok := seen[key]; ok {
			return fmt.Errorf("upsert data contains duplicate primary key at rows %d and %d", first, i)
		}
		seen[key] = i
	}

	return nil
}

func primaryKeyValuesKey(values []any) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to encode primary key values: %w", err)
	}
	return string(encoded), nil
}

// UpdateJSON modifies rows using JSON body format.
// PATCH /data/query/{table}
func (dao *DatabaseConnection) UpdateJSON(ctx context.Context, relation string, req UpdateRequest) ([]byte, error) {
	return dao.updateJSON(ctx, dao.Client, relation, req)
}

func (dao *DatabaseConnection) updateJSON(ctx context.Context, exec Executor, relation string, req UpdateRequest) ([]byte, error) {
	if err := tools.ValidateTableName(relation); err != nil {
		return nil, err
	}

	table, err := dao.Schema.SearchTbls(relation)
	if err != nil {
		return nil, err
	}

	if len(req.Data) == 0 {
		return nil, errors.New("update requires at least one column")
	}

	query := fmt.Sprintf("UPDATE [%s] SET ", relation)
	var args []any

	first := true
	for col, val := range req.Data {
		_, err = table.SearchCols(col)
		if err != nil {
			return nil, err
		}

		if !first {
			query += ", "
		}
		first = false
		query += fmt.Sprintf("[%s] = ?", col)
		args = append(args, val)
	}
	query += " "

	where, whereArgs, err := table.BuildWhereFromJSON(req.Where, dao.Schema)
	if err != nil {
		return nil, err
	}

	if where == "" {
		return nil, tools.ErrMissingWhereClause
	}
	policy, err := dao.compilePolicy(ctx, relation, "update", req.Data)
	if err != nil {
		return nil, err
	}
	where, whereArgs = appendPolicyWhere(where, whereArgs, policy)
	query += where
	args = append(args, whereArgs...)
	query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)

	result, err := ExecContextWithRetry(ctx, exec, query, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return json.Marshal(map[string]any{"rows_affected": rowsAffected})
}

// DeleteJSON removes rows using JSON body format.
// DELETE /data/query/{table}
func (dao *DatabaseConnection) DeleteJSON(ctx context.Context, relation string, req DeleteRequest) ([]byte, error) {
	return dao.deleteJSON(ctx, dao.Client, relation, req)
}

func (dao *DatabaseConnection) deleteJSON(ctx context.Context, exec Executor, relation string, req DeleteRequest) ([]byte, error) {
	if err := tools.ValidateTableName(relation); err != nil {
		return nil, err
	}

	table, err := dao.Schema.SearchTbls(relation)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("DELETE FROM [%s] ", relation)

	where, args, err := table.BuildWhereFromJSON(req.Where, dao.Schema)
	if err != nil {
		return nil, err
	}

	if where == "" {
		return nil, tools.ErrMissingWhereClause
	}
	policy, err := dao.compilePolicy(ctx, relation, "delete", nil)
	if err != nil {
		return nil, err
	}
	where, args = appendPolicyWhere(where, args, policy)
	query += where
	query, args = applyPolicyCTE(query, args, dao, policy.NeedsMembershipCTE)

	result, err := ExecContextWithRetry(ctx, exec, query, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return json.Marshal(map[string]any{"rows_affected": rowsAffected})
}

// queryJSONWithExec executes a query and returns JSON results using the provided executor.
func (dao *DatabaseConnection) queryJSONWithExec(ctx context.Context, exec Executor, query string, args ...any) ([]byte, error) {
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := tools.ScanRows(rows)
	if err != nil {
		return nil, err
	}

	return json.Marshal(results)
}
