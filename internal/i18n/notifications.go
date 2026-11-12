package i18n

import "strings"

type notifTmpl struct{ title, body string }

// notifCatalog[locale][type] holds the title/body templates. Placeholders are
// written as {name} and filled from the params map. Keys match the
// NotificationType* constants defined per feature.
var notifCatalog = map[string]map[string]notifTmpl{
	"vi": {
		"work_schedule_registration":              {"Đăng ký lịch làm việc", "{registrant} đã đăng ký lịch làm việc từ {fromDate} đến {toDate}"},
		"work_schedule_approved":                  {"Lịch làm việc đã được duyệt", "Lịch làm việc từ {fromDate} đến {toDate} của bạn đã được duyệt."},
		"work_schedule_rejected":                  {"Lịch làm việc bị từ chối", "Lịch làm việc từ {fromDate} đến {toDate} của bạn bị từ chối."},
		"daily_report_reminder":                   {"Nhắc nộp báo cáo cuối ngày", "Bạn sắp hết giờ làm hôm nay mà chưa nộp báo cáo cuối ngày. Hãy điền báo cáo khi check-out."},
		"attendance_check_in_reminder":            {"Nhắc chấm công", "Ca làm việc của bạn bắt đầu lúc {sessionTime}. Hãy check-in."},
		"attendance_check_out_reminder":           {"Nhắc chấm công", "Ca làm việc của bạn kết thúc lúc {sessionTime}. Hãy check-out."},
		"attendance_correction_request_submitted": {"Yêu cầu bổ sung công", "{requester} đã gửi yêu cầu bổ sung công: {session}"},
		"attendance_correction_request_approved":  {"Yêu cầu bổ sung công đã được duyệt", "Yêu cầu bổ sung công ({session}) của bạn đã được duyệt."},
		"attendance_correction_request_rejected":  {"Yêu cầu bổ sung công bị từ chối", "Yêu cầu bổ sung công ({session}) của bạn bị từ chối."},

		"workspace_invitation":          {"Lời mời vào workspace", "{actor} đã mời bạn vào {workspaceName} với vai trò {role}"},
		"workspace_member_role_updated": {"Vai trò trong workspace đã thay đổi", "Vai trò của bạn trong {workspaceName} đã được đổi thành {role}"},
		"workspace_member_removed":      {"Bạn đã bị xoá khỏi workspace", "Bạn đã bị xoá khỏi {workspaceName}"},
		"workspace_updated":             {"Workspace có cập nhật", "Một workspace đã thay đổi"},

		"project_member_added":        {"Bạn được thêm vào dự án", "Bạn đã được thêm vào {projectName} với vai trò {role}"},
		"project_member_role_updated": {"Vai trò trong dự án đã thay đổi", "Vai trò của bạn trong {projectName} đã được đổi thành {role}"},
		"project_member_removed":      {"Bạn đã bị xoá khỏi dự án", "Bạn đã bị xoá khỏi {projectName}"},
		"project_updated":             {"Dự án có cập nhật", "Một dự án đã thay đổi"},

		"task_assigned":                {"Bạn được giao task", "{caller} đã giao task \"{taskTitle}\" cho bạn"},
		"task_unassigned":              {"Bạn được gỡ khỏi task", "{caller} đã gỡ bạn khỏi task \"{taskTitle}\""},
		"task_status_updated":          {"Trạng thái task đã thay đổi", "{caller} đã chuyển \"{taskTitle}\" sang trạng thái {status}"},
		"task_fault_holder_assigned":   {"Task được trả lại cho bạn", "{caller} đã chỉ định bạn chịu lỗi cho task \"{taskTitle}\""},
		"task_fault_holder_unassigned": {"Bạn không còn chịu lỗi task", "{caller} đã gỡ bạn khỏi vai trò chịu lỗi của task \"{taskTitle}\""},
		"task_deadline_reminder":       {"Task sắp đến hạn", "Task \"{taskTitle}\" đến hạn vào {dueAt}"},
		"task_comment_added":           {"Bình luận mới", "{caller} đã bình luận trong \"{taskTitle}\": {snippet}"},
		"task_mentioned":               {"Bạn được nhắc đến", "{caller} đã nhắc đến bạn trong \"{taskTitle}\": {snippet}"},
		"task_updated":                 {"Task có cập nhật", "Task \"{taskTitle}\" đã được cập nhật"},

		"initiative_created":            {"Có sáng kiến mới", "Một sáng kiến mới vừa được gửi để xem xét"},
		"late_log_created":              {"Có ghi nhận đi muộn mới", "Một ghi nhận đi muộn mới vừa được thêm"},
		"customer_feedback_created":     {"Có feedback khách hàng mới", "Một feedback khách hàng mới vừa được thêm"},
		"workspace_performance_updated": {"Cập nhật hiệu suất workspace", "Một bản ghi hiệu suất workspace vừa được tạo"},
	},
	"en": {
		"work_schedule_registration":              {"Work schedule registered", "{registrant} registered a work schedule from {fromDate} to {toDate}"},
		"work_schedule_approved":                  {"Work schedule approved", "Your work schedule from {fromDate} to {toDate} was approved."},
		"work_schedule_rejected":                  {"Work schedule rejected", "Your work schedule from {fromDate} to {toDate} was rejected."},
		"daily_report_reminder":                   {"End-of-day report reminder", "Your working hours are almost over and you haven't submitted your end-of-day report. Please fill it in when you check out."},
		"attendance_check_in_reminder":            {"Attendance reminder", "Your shift starts at {sessionTime}. Please check in."},
		"attendance_check_out_reminder":           {"Attendance reminder", "Your shift ends at {sessionTime}. Please check out."},
		"attendance_correction_request_submitted": {"Attendance correction request", "{requester} submitted an attendance correction request: {session}"},
		"attendance_correction_request_approved":  {"Attendance correction approved", "Your attendance correction request ({session}) was approved."},
		"attendance_correction_request_rejected":  {"Attendance correction rejected", "Your attendance correction request ({session}) was rejected."},

		"workspace_invitation":          {"Workspace invitation", "{actor} invited you to {workspaceName} as {role}"},
		"workspace_member_role_updated": {"Workspace role changed", "Your role in {workspaceName} was changed to {role}"},
		"workspace_member_removed":      {"Removed from workspace", "You were removed from {workspaceName}"},
		"workspace_updated":             {"Workspace updated", "A workspace has changed"},

		"project_member_added":        {"Added to a project", "You were added to {projectName} as {role}"},
		"project_member_role_updated": {"Project role changed", "Your role in {projectName} was changed to {role}"},
		"project_member_removed":      {"Removed from project", "You were removed from {projectName}"},
		"project_updated":             {"Project updated", "A project has changed"},

		"task_assigned":                {"Task assigned to you", "{caller} assigned the task \"{taskTitle}\" to you"},
		"task_unassigned":              {"Removed from a task", "{caller} removed you from the task \"{taskTitle}\""},
		"task_status_updated":          {"Task status changed", "{caller} moved \"{taskTitle}\" to {status}"},
		"task_fault_holder_assigned":   {"Task returned to you", "{caller} assigned you as the fault holder for \"{taskTitle}\""},
		"task_fault_holder_unassigned": {"No longer the fault holder", "{caller} removed you as the fault holder for \"{taskTitle}\""},
		"task_deadline_reminder":       {"Task due soon", "Task \"{taskTitle}\" is due at {dueAt}"},
		"task_comment_added":           {"New comment", "{caller} commented on \"{taskTitle}\": {snippet}"},
		"task_mentioned":               {"You were mentioned", "{caller} mentioned you in \"{taskTitle}\": {snippet}"},
		"task_updated":                 {"Task updated", "Task \"{taskTitle}\" was updated"},

		"initiative_created":            {"New initiative", "A new initiative was submitted for review"},
		"late_log_created":              {"New late arrival log", "A new late arrival log was added"},
		"customer_feedback_created":     {"New customer feedback", "New customer feedback was added"},
		"workspace_performance_updated": {"Workspace performance updated", "A workspace performance record was created"},
	},
}

func applyParams(s string, p map[string]string) string {
	if p == nil {
		return s
	}
	for k, v := range p {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// RenderNotification builds the localized (title, body) for a notification
// type and its params. The "role" and "status" params are treated as raw enum
// values and localized to `locale` before substitution, so callers pass the
// raw value (e.g. "holder", "in_progress"), not pre-translated text.
func RenderNotification(locale, ntype string, params map[string]string) (string, string) {
	locale = Normalize(locale)
	p := make(map[string]string, len(params)+1)
	for k, v := range params {
		p[k] = v
	}
	if role, ok := p["role"]; ok {
		if strings.HasPrefix(ntype, "project_") {
			p["role"] = ProjectRoleLabel(locale, role)
		} else {
			p["role"] = WorkspaceRoleLabel(locale, role)
		}
	}
	if status, ok := p["status"]; ok {
		p["status"] = TaskStatusLabel(locale, status)
	}
	t, ok := notifCatalog[locale][ntype]
	if !ok {
		if t, ok = notifCatalog[Default][ntype]; !ok {
			return "", ""
		}
	}
	return applyParams(t.title, p), applyParams(t.body, p)
}
