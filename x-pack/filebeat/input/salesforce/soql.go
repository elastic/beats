package salesforce

import (
	"github.com/g8rswimmer/go-sfdc/soql"
)

const (
	Select = "SELECT"
	From   = "FROM"
	Where  = "WHERE"
	Order  = "ORDER"
	Asc    = "ASC"
	Null   = "NULL"
)

type querier struct {
	expression string
}

func (q querier) Format() (string, error) {
	return q.expression, nil
}

func (s *SoqlConfig) getQueryFormatter() (querier, error) {
	if s.Query != "" {
		q := querier{
			expression: s.Query,
		}
		return q, nil
	}

	return querier{}, nil
}

func (s *SoqlConfig) addWhere() *soql.WhereClause {
	var whereClause *soql.WhereClause
	for _, where := range s.Where.And {
		newWhereClause := getWhereClause(where.Field, where.Values, where.Op)
		whereClause.And(newWhereClause)
	}

	for _, where := range s.Where.Or {
		newWhereClause := getWhereClause(where.Field, where.Values, where.Op)
		whereClause.Or(newWhereClause)
		whereClause.Clause()
	}

	return whereClause
}

func getWhereClause(field string, values []interface{}, op string) *soql.WhereClause {
	var newWhereClause *soql.WhereClause
	switch op {
	case "=":
		newWhereClause, _ = soql.WhereEquals(field, values)
	case "!=":
		newWhereClause, _ = soql.WhereNotEquals(field, values)
	case ">":
		newWhereClause, _ = soql.WhereGreaterThan(field, values, false)
	case ">=":
		newWhereClause, _ = soql.WhereGreaterThan(field, values, true)
	case "<":
		newWhereClause, _ = soql.WhereLessThan(field, values, false)
	case "<=":
		newWhereClause, _ = soql.WhereLessThan(field, values, true)
	case "IN":
		newWhereClause, _ = soql.WhereIn(field, values)
	case "NOT IN":
		newWhereClause, _ = soql.WhereNotIn(field, values)
	}

	return newWhereClause
}
