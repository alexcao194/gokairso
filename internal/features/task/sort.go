package task

import (
	"fmt"
	"strings"
)

var taskSortColumns = map[string]string{
	"title":       "lower(t.title)",
	"project":     "lower(p.name)",
	"module":      "m.start_date",
	"status":      "array_position(ARRAY['backlog', 'todo', 'in_progress', 'review', 'done', 'canceled']::task_status[], t.status)",
	"priority":    "t.priority",
	"estimation":  "t.estimation",
	"returnCount": "t.return_count",
	"startDate":   "t.start_date",
	"endDate":     "t.end_date",
}

var taskSortFieldNames = []string{
	"title", "project", "module", "status", "priority", "estimation", "returnCount", "startDate", "endDate",
}

var nullableSortColumns = map[string]bool{
	"title":      true,
	"project":    true,
	"module":     true,
	"estimation": true,
	"startDate":  true,
	"endDate":    true,
}

type TaskSortRule struct {
	Field string
	Order string
}

func defaultTaskSortRules() []TaskSortRule {
	return []TaskSortRule{
		{Field: "status", Order: "asc"},
		{Field: "endDate", Order: "asc"},
	}
}

func parseTaskSortParam(raw string) ([]TaskSortRule, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultTaskSortRules(), nil
	}

	parts := strings.Split(raw, ",")
	rules := make([]TaskSortRule, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		field, order, _ := strings.Cut(part, ":")
		field = strings.TrimSpace(field)
		order = strings.ToLower(strings.TrimSpace(order))
		if order == "" {
			order = "asc"
		}
		if _, ok := taskSortColumns[field]; !ok {
			return nil, fmt.Errorf("Trường sắp xếp %q phải là một trong: %s", field, strings.Join(taskSortFieldNames, ", "))
		}
		if order != "asc" && order != "desc" {
			return nil, fmt.Errorf("Thứ tự sắp xếp của %q phải là asc hoặc desc (hiện là %q)", field, order)
		}
		if seen[field] {
			return nil, fmt.Errorf("Trường sắp xếp %q bị khai báo nhiều lần", field)
		}
		seen[field] = true
		rules = append(rules, TaskSortRule{Field: field, Order: order})
	}
	if len(rules) == 0 {
		return defaultTaskSortRules(), nil
	}
	return rules, nil
}

func buildTaskOrderByClause(rules []TaskSortRule) string {
	if len(rules) == 0 {
		rules = defaultTaskSortRules()
	}
	var b strings.Builder
	b.WriteString("ORDER BY ")
	for _, rule := range rules {
		expr := taskSortColumns[rule.Field]
		dir := "ASC"
		if rule.Order == "desc" {
			dir = "DESC"
		}
		b.WriteString(expr)
		b.WriteString(" ")
		b.WriteString(dir)
		if nullableSortColumns[rule.Field] {
			b.WriteString(" NULLS LAST")
		}
		b.WriteString(", ")
	}
	b.WriteString("t.created_at DESC, t.id")
	return b.String()
}

type sqlArgs struct {
	values []interface{}
}

func (a *sqlArgs) bind(v interface{}) string {
	a.values = append(a.values, v)
	return fmt.Sprintf("$%d", len(a.values))
}
