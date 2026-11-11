// Package i18n holds the server-side message catalog used to render
// notifications (and push payloads) in a user's locale. Clients (web, mobile)
// keep their own mirrors of these keys; this package is the source of truth for
// anything the server must render itself — notably push notifications, where
// the recipient's device shows text the server produced.
package i18n

// Default is the locale used when a user has none set or an unknown one.
const Default = "vi"

var supported = map[string]bool{"vi": true, "en": true}

// Supported returns the list of locales the backend can render.
func Supported() []string { return []string{"vi", "en"} }

// IsSupported reports whether locale has a catalog.
func IsSupported(locale string) bool { return supported[locale] }

// Normalize returns locale if supported, otherwise Default.
func Normalize(locale string) string {
	if supported[locale] {
		return locale
	}
	return Default
}

var workspaceRole = map[string]map[string]string{
	"vi": {"holder": "chủ workspace", "admin": "quản trị viên", "member": "thành viên"},
	"en": {"holder": "owner", "admin": "admin", "member": "member"},
}

var projectRole = map[string]map[string]string{
	"vi": {"holder": "chủ dự án", "admin": "quản trị viên", "member": "thành viên"},
	"en": {"holder": "owner", "admin": "admin", "member": "member"},
}

var taskStatus = map[string]map[string]string{
	"vi": {
		"backlog": "Backlog", "todo": "Cần làm", "in_progress": "Đang làm",
		"review": "Chờ duyệt", "done": "Hoàn thành", "canceled": "Đã huỷ",
	},
	"en": {
		"backlog": "Backlog", "todo": "To do", "in_progress": "In progress",
		"review": "In review", "done": "Done", "canceled": "Canceled",
	},
}

func lookup(table map[string]map[string]string, locale, key string) string {
	if m, ok := table[Normalize(locale)]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}

// WorkspaceRoleLabel localizes a workspace role value (holder/admin/member).
func WorkspaceRoleLabel(locale, role string) string { return lookup(workspaceRole, locale, role) }

// ProjectRoleLabel localizes a project role value.
func ProjectRoleLabel(locale, role string) string { return lookup(projectRole, locale, role) }

// TaskStatusLabel localizes a task status value.
func TaskStatusLabel(locale, status string) string { return lookup(taskStatus, locale, status) }
