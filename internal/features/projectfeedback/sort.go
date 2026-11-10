package projectfeedback

import (
	"fmt"
	"strings"
)

var feedbackSortColumns = map[string]string{
	"code":        "f.sequence_no",
	"priority":    "f.priority",
	"feature":     "lower(f.feature_name)",
	"submitter":   "lower(f.submitter_name)",
	"company":     "lower(f.submitter_company)",
	"status":      "array_position(ARRAY['pending', 'acknowledged', 'resolved', 'canceled']::text[], f.status)",
	"expectedFix": "f.expected_fix_date",
	"submitted":   "f.created_at",
}

var feedbackSortFieldNames = []string{
	"code", "priority", "feature", "submitter", "company", "status", "expectedFix", "submitted",
}

var nullableFeedbackSortColumns = map[string]bool{
	"expectedFix": true,
}

type FeedbackSortRule struct {
	Field string
	Order string
}

func defaultFeedbackSortRules() []FeedbackSortRule {
	return []FeedbackSortRule{
		{Field: "submitted", Order: "desc"},
	}
}

func parseFeedbackSortParam(raw string) ([]FeedbackSortRule, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultFeedbackSortRules(), nil
	}

	parts := strings.Split(raw, ",")
	rules := make([]FeedbackSortRule, 0, len(parts))
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
		if _, ok := feedbackSortColumns[field]; !ok {
			return nil, fmt.Errorf("Trường sắp xếp %q phải là một trong: %s", field, strings.Join(feedbackSortFieldNames, ", "))
		}
		if order != "asc" && order != "desc" {
			return nil, fmt.Errorf("Thứ tự sắp xếp của %q phải là asc hoặc desc (hiện là %q)", field, order)
		}
		if seen[field] {
			return nil, fmt.Errorf("Trường sắp xếp %q bị khai báo nhiều lần", field)
		}
		seen[field] = true
		rules = append(rules, FeedbackSortRule{Field: field, Order: order})
	}
	if len(rules) == 0 {
		return defaultFeedbackSortRules(), nil
	}
	return rules, nil
}

func buildFeedbackOrderByClause(rules []FeedbackSortRule) string {
	if len(rules) == 0 {
		rules = defaultFeedbackSortRules()
	}
	var b strings.Builder
	b.WriteString("ORDER BY ")
	for _, rule := range rules {
		expr := feedbackSortColumns[rule.Field]
		dir := "ASC"
		if rule.Order == "desc" {
			dir = "DESC"
		}
		b.WriteString(expr)
		b.WriteString(" ")
		b.WriteString(dir)
		if nullableFeedbackSortColumns[rule.Field] {
			b.WriteString(" NULLS LAST")
		}
		b.WriteString(", ")
	}
	b.WriteString("f.created_at DESC, f.id")
	return b.String()
}

type sqlArgs struct {
	values []interface{}
}

func (a *sqlArgs) bind(v interface{}) string {
	a.values = append(a.values, v)
	return fmt.Sprintf("$%d", len(a.values))
}
