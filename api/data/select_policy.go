package data

import (
	"context"

	"github.com/atombasedev/atombase/definitions"
)

type selectPolicySet map[string]definitions.CompiledPredicate

func (dao *DatabaseConnection) compileSelectPolicies(ctx context.Context, rel Relation) (selectPolicySet, error) {
	policies := make(selectPolicySet)
	seen := make(map[string]bool)
	var walk func(Relation) error
	walk = func(curr Relation) error {
		if !seen[curr.name] {
			predicate, err := dao.compilePolicy(ctx, curr.name, "select", nil)
			if err != nil {
				return err
			}
			policies[curr.name] = predicate
			seen[curr.name] = true
		}
		for _, join := range curr.joins {
			if err := walk(*join); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(rel); err != nil {
		return nil, err
	}
	return policies, nil
}

func (dao *DatabaseConnection) compileCustomJoinPolicies(ctx context.Context, cjq *CustomJoinQuery) (selectPolicySet, error) {
	policies := make(selectPolicySet)
	tables := map[string]bool{cjq.BaseTable: true}
	for _, join := range cjq.Joins {
		tables[join.table] = true
	}
	for table := range tables {
		predicate, err := dao.compilePolicy(ctx, table, "select", nil)
		if err != nil {
			return nil, err
		}
		policies[table] = predicate
	}
	return policies, nil
}
