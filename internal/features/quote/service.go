package quote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/currency"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrForbidden      = errors.New("Bạn không có quyền thực hiện thao tác này")
	ErrNotFound       = errors.New("Không tìm thấy báo giá")
	ErrNotEditable    = errors.New("Chỉ sửa hoặc xoá được báo giá ở trạng thái nháp")
	ErrBadTransition  = errors.New("Không thể chuyển báo giá sang trạng thái này")
	ErrCustomerAbsent = errors.New("Không tìm thấy khách hàng")
)

type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) error { return ValidationError{Msg: fmt.Sprintf(format, a...)} }

const (
	maxItems     = 500
	maxQtyMilli  = 1_000_000_000_000
	maxUnitPrice = 1_000_000_000_000_000
	maxTextLen   = 20_000
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func parseID(s, what string) (pgtype.UUID, error) {
	id, err := pgutil.ParseUUID(s)
	if err != nil {
		return pgtype.UUID{}, invalid("%s không hợp lệ", what)
	}
	return id, nil
}

func (s *Service) access(ctx context.Context, workspaceID, callerID string, manage bool) (pgtype.UUID, pgtype.UUID, error) {
	ws, err := parseID(workspaceID, "workspaceId")
	if err != nil {
		return ws, pgtype.UUID{}, err
	}
	caller, err := parseID(callerID, "userId")
	if err != nil {
		return ws, caller, err
	}
	role, err := s.repo.GetCallerRole(ctx, ws, caller)
	if err != nil {
		return ws, caller, err
	}
	if role == "" || (manage && !role.CanManageMembers()) {
		return ws, caller, ErrForbidden
	}
	return ws, caller, nil
}

// Currencies lists every supported ISO 4217 currency.
func (s *Service) Currencies() []currency.Currency { return currency.All() }

// ---- customers ----

func validateCustomer(req *CustomerRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return invalid("Tên khách hàng không được để trống")
	}
	if len(req.Name) > 255 || len(req.ContactName) > 255 || len(req.TaxCode) > 64 || len(req.Email) > 320 || len(req.Phone) > 50 {
		return invalid("Thông tin khách hàng quá dài")
	}
	return nil
}

func (s *Service) ListCustomers(ctx context.Context, workspaceID, callerID string) ([]CustomerResponse, error) {
	ws, _, err := s.access(ctx, workspaceID, callerID, false)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListCustomers(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make([]CustomerResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, customerResponse(c))
	}
	return out, nil
}

func (s *Service) CreateCustomer(ctx context.Context, workspaceID, callerID string, req CustomerRequest) (*CustomerResponse, error) {
	ws, _, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return nil, err
	}
	if err := validateCustomer(&req); err != nil {
		return nil, err
	}
	c, err := s.repo.CreateCustomer(ctx, database.CreateCustomerParams{
		WorkspaceID: ws, Name: req.Name, TaxCode: req.TaxCode, Address: req.Address,
		ContactName: req.ContactName, Email: req.Email, Phone: req.Phone, Note: req.Note,
	})
	if err != nil {
		return nil, err
	}
	resp := customerResponse(c)
	return &resp, nil
}

func (s *Service) UpdateCustomer(ctx context.Context, workspaceID, callerID, customerID string, req CustomerRequest) (*CustomerResponse, error) {
	ws, _, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return nil, err
	}
	id, err := parseID(customerID, "customerId")
	if err != nil {
		return nil, err
	}
	if err := validateCustomer(&req); err != nil {
		return nil, err
	}
	c, ok, err := s.repo.UpdateCustomer(ctx, database.UpdateCustomerParams{
		ID: id, WorkspaceID: ws, Name: req.Name, TaxCode: req.TaxCode, Address: req.Address,
		ContactName: req.ContactName, Email: req.Email, Phone: req.Phone, Note: req.Note,
	})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCustomerAbsent
	}
	resp := customerResponse(c)
	return &resp, nil
}

func (s *Service) DeleteCustomer(ctx context.Context, workspaceID, callerID, customerID string) error {
	ws, _, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return err
	}
	id, err := parseID(customerID, "customerId")
	if err != nil {
		return err
	}
	n, err := s.repo.DeleteCustomer(ctx, id, ws)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCustomerAbsent
	}
	return nil
}

// ---- settings ----

func (s *Service) settingsOrDefault(ctx context.Context, ws pgtype.UUID) (database.QuoteSetting, error) {
	st, ok, err := s.repo.GetSettings(ctx, ws)
	if err != nil {
		return st, err
	}
	if !ok {
		return defaultSettings(ws), nil
	}
	return st, nil
}

func (s *Service) GetSettings(ctx context.Context, workspaceID, callerID string) (*QuoteSettingsResponse, error) {
	ws, _, err := s.access(ctx, workspaceID, callerID, false)
	if err != nil {
		return nil, err
	}
	st, err := s.settingsOrDefault(ctx, ws)
	if err != nil {
		return nil, err
	}
	resp := settingsResponse(st)
	return &resp, nil
}

func (s *Service) UpdateSettings(ctx context.Context, workspaceID, callerID string, req QuoteSettingsRequest) (*QuoteSettingsResponse, error) {
	ws, _, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return nil, err
	}
	req.NumberPrefix = strings.TrimSpace(req.NumberPrefix)
	if req.NumberPrefix == "" {
		req.NumberPrefix = "BG"
	}
	req.DefaultCurrency = strings.ToUpper(strings.TrimSpace(req.DefaultCurrency))
	switch {
	case len(req.NumberPrefix) > 16:
		return nil, invalid("Tiền tố số báo giá tối đa 16 ký tự")
	case !currency.IsValid(req.DefaultCurrency):
		return nil, invalid("Đơn vị tiền tệ không hợp lệ")
	case req.DefaultVatRate < 0 || req.DefaultVatRate > 100:
		return nil, invalid("Thuế suất VAT phải trong khoảng 0–100%%")
	case req.DefaultValidDays < 0 || req.DefaultValidDays > 3650:
		return nil, invalid("Số ngày hiệu lực không hợp lệ")
	}
	st, err := s.repo.UpsertSettings(ctx, database.UpsertQuoteSettingsParams{
		WorkspaceID: ws, CompanyName: req.CompanyName, Address: req.Address, TaxCode: req.TaxCode,
		Phone: req.Phone, Email: req.Email, Website: req.Website, BankInfo: req.BankInfo,
		LogoUrl: req.LogoURL, SignatureUrl: req.SignatureURL, NumberPrefix: req.NumberPrefix,
		DefaultCurrency: req.DefaultCurrency, DefaultVatBp: percentToBp(req.DefaultVatRate),
		DefaultValidDays: int32(req.DefaultValidDays), DefaultTerms: req.DefaultTerms,
	})
	if err != nil {
		return nil, err
	}
	resp := settingsResponse(st)
	return &resp, nil
}

// ---- quotes ----

type ListFilter struct {
	ModuleID  string
	ProjectID string
	Status    string
}

func optionalUUID(s, what string) (pgtype.UUID, error) {
	if s == "" {
		return pgtype.UUID{}, nil
	}
	return parseID(s, what)
}

func (s *Service) ListQuotes(ctx context.Context, workspaceID, callerID string, f ListFilter, p pagination.Params) (pagination.Response[QuoteSummaryResponse], error) {
	var empty pagination.Response[QuoteSummaryResponse]
	ws, _, err := s.access(ctx, workspaceID, callerID, false)
	if err != nil {
		return empty, err
	}
	moduleID, err := optionalUUID(f.ModuleID, "moduleId")
	if err != nil {
		return empty, err
	}
	projectID, err := optionalUUID(f.ProjectID, "projectId")
	if err != nil {
		return empty, err
	}
	var status pgtype.Text
	if f.Status != "" {
		status = pgtype.Text{String: f.Status, Valid: true}
	}
	rows, err := s.repo.ListQuotes(ctx, database.ListQuotesParams{
		WorkspaceID: ws, ModuleID: moduleID, ProjectID: projectID, Status: status,
		RowLimit: int32(p.LimitWithMore()), RowOffset: int32(p.Offset()),
	})
	if err != nil {
		return empty, err
	}
	hasMore := pagination.Process(&rows, p.Limit)
	items := make([]QuoteSummaryResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, quoteSummaryResponse(r))
	}
	return pagination.NewResponse(items, hasMore, p.Page, p.Limit), nil
}

func (s *Service) CreateQuote(ctx context.Context, workspaceID, callerID string, req CreateQuoteRequest) (*QuoteResponse, error) {
	ws, caller, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return nil, err
	}
	moduleID, err := parseID(req.ModuleID, "moduleId")
	if err != nil {
		return nil, err
	}
	mod, ok, err := s.repo.GetModuleContext(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if !ok || mod.WorkspaceID != ws {
		return nil, invalid("Không tìm thấy module trong workspace này")
	}
	var customerID pgtype.UUID
	if req.CustomerID != nil && *req.CustomerID != "" {
		if customerID, err = s.ensureCustomer(ctx, ws, *req.CustomerID); err != nil {
			return nil, err
		}
	}
	st, err := s.settingsOrDefault(ctx, ws)
	if err != nil {
		return nil, err
	}

	today := vietnamToday()
	issue := pgtype.Date{Time: today, Valid: true}
	var validUntil pgtype.Date
	if st.DefaultValidDays > 0 {
		validUntil = pgtype.Date{Time: today.AddDate(0, 0, int(st.DefaultValidDays)), Valid: true}
	}

	var quoteID pgtype.UUID
	err = s.repo.ExecTx(ctx, func(tx Repository) error {
		n, err := tx.NextNumber(ctx, ws, int32(today.Year()))
		if err != nil {
			return err
		}
		quoteID, err = tx.CreateQuote(ctx, database.CreateQuoteParams{
			WorkspaceID: ws, ProjectID: mod.ProjectID, ModuleID: mod.ModuleID, CustomerID: customerID,
			Number:   fmt.Sprintf("%s-%d-%04d", st.NumberPrefix, today.Year(), n),
			Title:    mod.ModuleName,
			Currency: st.DefaultCurrency, Language: "vi",
			IssueDate: issue, ValidUntil: validUntil,
			VatBp: st.DefaultVatBp, Terms: st.DefaultTerms, CreatedBy: caller,
		})
		if err != nil {
			return err
		}
		return tx.InsertEvent(ctx, quoteID, "created", caller, "")
	})
	if err != nil {
		return nil, err
	}
	return s.load(ctx, ws, quoteID)
}

func (s *Service) ensureCustomer(ctx context.Context, ws pgtype.UUID, customerID string) (pgtype.UUID, error) {
	id, err := parseID(customerID, "customerId")
	if err != nil {
		return id, err
	}
	_, ok, err := s.repo.GetCustomer(ctx, id, ws)
	if err != nil {
		return id, err
	}
	if !ok {
		return id, ErrCustomerAbsent
	}
	return id, nil
}

func (s *Service) load(ctx context.Context, ws, id pgtype.UUID) (*QuoteResponse, error) {
	row, ok, err := s.repo.GetQuote(ctx, id, ws)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	items, err := s.repo.ListItems(ctx, id)
	if err != nil {
		return nil, err
	}
	events, err := s.repo.ListEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	var live *database.Customer
	if row.Status == StatusDraft && row.CustomerID.Valid {
		c, ok, err := s.repo.GetCustomer(ctx, row.CustomerID, ws)
		if err != nil {
			return nil, err
		}
		if ok {
			live = &c
		}
	}
	st, err := s.settingsOrDefault(ctx, ws)
	if err != nil {
		return nil, err
	}
	resp := quoteResponse(row, items, events, live, st)
	return &resp, nil
}

func (s *Service) GetQuote(ctx context.Context, workspaceID, callerID, quoteID string) (*QuoteResponse, error) {
	ws, _, err := s.access(ctx, workspaceID, callerID, false)
	if err != nil {
		return nil, err
	}
	id, err := parseID(quoteID, "quoteId")
	if err != nil {
		return nil, err
	}
	return s.load(ctx, ws, id)
}

func (s *Service) draft(ctx context.Context, ws, id pgtype.UUID) (database.GetQuoteRow, error) {
	row, ok, err := s.repo.GetQuote(ctx, id, ws)
	if err != nil {
		return row, err
	}
	if !ok {
		return row, ErrNotFound
	}
	if row.Status != StatusDraft {
		return row, ErrNotEditable
	}
	return row, nil
}

func validateQuote(req *UpdateQuoteRequest) error {
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.Title = strings.TrimSpace(req.Title)
	switch {
	case !currency.IsValid(req.Currency):
		return invalid("Đơn vị tiền tệ không hợp lệ")
	case req.Language != "vi" && req.Language != "en":
		return invalid("Ngôn ngữ báo giá phải là vi hoặc en")
	case len(req.Title) > 255:
		return invalid("Tiêu đề tối đa 255 ký tự")
	case len(req.Notes) > maxTextLen || len(req.Terms) > maxTextLen:
		return invalid("Ghi chú hoặc điều khoản quá dài")
	case req.DiscountType != "percent" && req.DiscountType != "amount":
		return invalid("Loại chiết khấu phải là percent hoặc amount")
	case req.DiscountPercent < 0 || req.DiscountPercent > 100:
		return invalid("Chiết khấu phải trong khoảng 0–100%%")
	case req.DiscountAmount < 0 || req.DiscountAmount > maxUnitPrice:
		return invalid("Số tiền chiết khấu không hợp lệ")
	case req.VatRate < 0 || req.VatRate > 100:
		return invalid("Thuế suất VAT phải trong khoảng 0–100%%")
	case len(req.Items) > maxItems:
		return invalid("Tối đa %d dòng hàng", maxItems)
	}
	for i := range req.Items {
		it := &req.Items[i]
		it.Name = strings.TrimSpace(it.Name)
		q := qtyToMilli(it.Quantity)
		switch {
		case it.Name == "":
			return invalid("Dòng %d: tên hạng mục không được để trống", i+1)
		case len(it.Name) > 500 || len(it.Unit) > 32 || len(it.Description) > maxTextLen:
			return invalid("Dòng %d: nội dung quá dài", i+1)
		case q <= 0 || q > maxQtyMilli:
			return invalid("Dòng %d: số lượng không hợp lệ", i+1)
		case it.UnitPrice < 0 || it.UnitPrice > maxUnitPrice:
			return invalid("Dòng %d: đơn giá không hợp lệ", i+1)
		case it.DiscountPercent < 0 || it.DiscountPercent > 100:
			return invalid("Dòng %d: chiết khấu phải trong khoảng 0–100%%", i+1)
		}
	}
	return nil
}

func (s *Service) UpdateQuote(ctx context.Context, workspaceID, callerID, quoteID string, req UpdateQuoteRequest) (*QuoteResponse, error) {
	ws, caller, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return nil, err
	}
	id, err := parseID(quoteID, "quoteId")
	if err != nil {
		return nil, err
	}
	if err := validateQuote(&req); err != nil {
		return nil, err
	}
	issue, err := parseDate(req.IssueDate)
	if err != nil || !issue.Valid {
		return nil, invalid("Ngày báo giá không hợp lệ")
	}
	validUntil, err := parseDate(req.ValidUntil)
	if err != nil {
		return nil, invalid("Ngày hết hiệu lực không hợp lệ")
	}
	if validUntil.Valid && validUntil.Time.Before(issue.Time) {
		return nil, invalid("Ngày hết hiệu lực phải sau ngày báo giá")
	}
	if _, err := s.draft(ctx, ws, id); err != nil {
		return nil, err
	}
	var customerID pgtype.UUID
	if req.CustomerID != nil && *req.CustomerID != "" {
		if customerID, err = s.ensureCustomer(ctx, ws, *req.CustomerID); err != nil {
			return nil, err
		}
	}

	items := make([]database.InsertQuoteItemParams, 0, len(req.Items))
	lines := make([]int64, 0, len(req.Items))
	for i, it := range req.Items {
		q, bp := qtyToMilli(it.Quantity), percentToBp(it.DiscountPercent)
		lt := lineTotal(q, it.UnitPrice, bp)
		lines = append(lines, lt)
		items = append(items, database.InsertQuoteItemParams{
			Position: int32(i), Name: it.Name, Description: it.Description, Unit: it.Unit,
			QuantityMilli: q, UnitPrice: it.UnitPrice, DiscountBp: bp, LineTotal: lt,
		})
	}
	discountValue := req.DiscountAmount
	if req.DiscountType == "percent" {
		discountValue = int64(percentToBp(req.DiscountPercent))
	}
	vatBp := percentToBp(req.VatRate)
	t := computeTotals(lines, req.DiscountType, discountValue, vatBp)

	err = s.repo.ExecTx(ctx, func(tx Repository) error {
		if err := tx.UpdateQuoteDraft(ctx, database.UpdateQuoteDraftParams{
			ID: id, WorkspaceID: ws, CustomerID: customerID, Title: req.Title, Currency: req.Currency,
			Language: req.Language, IssueDate: issue, ValidUntil: validUntil,
			DiscountType: req.DiscountType, DiscountValue: discountValue, VatBp: vatBp,
			Subtotal: t.Subtotal, DiscountTotal: t.Discount, TaxTotal: t.Tax, GrandTotal: t.Grand,
			Notes: req.Notes, Terms: req.Terms,
		}); err != nil {
			return err
		}
		if err := tx.ReplaceItems(ctx, id, items); err != nil {
			return err
		}
		return tx.InsertEvent(ctx, id, "updated", caller, "")
	})
	if err != nil {
		return nil, err
	}
	return s.load(ctx, ws, id)
}

func (s *Service) DeleteQuote(ctx context.Context, workspaceID, callerID, quoteID string) error {
	ws, _, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return err
	}
	id, err := parseID(quoteID, "quoteId")
	if err != nil {
		return err
	}
	if _, err := s.draft(ctx, ws, id); err != nil {
		return err
	}
	_, err = s.repo.DeleteQuote(ctx, id, ws)
	return err
}

func (s *Service) SendQuote(ctx context.Context, workspaceID, callerID, quoteID string) (*QuoteResponse, error) {
	ws, caller, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return nil, err
	}
	id, err := parseID(quoteID, "quoteId")
	if err != nil {
		return nil, err
	}
	row, err := s.draft(ctx, ws, id)
	if err != nil {
		return nil, err
	}
	if !row.CustomerID.Valid {
		return nil, invalid("Chọn khách hàng trước khi gửi báo giá")
	}
	customer, ok, err := s.repo.GetCustomer(ctx, row.CustomerID, ws)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCustomerAbsent
	}
	items, err := s.repo.ListItems(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, invalid("Báo giá cần ít nhất một dòng hàng")
	}
	st, err := s.settingsOrDefault(ctx, ws)
	if err != nil {
		return nil, err
	}
	customerSnap, err := json.Marshal(customerParty(customer))
	if err != nil {
		return nil, err
	}
	issuerSnap, err := json.Marshal(issuerParty(st))
	if err != nil {
		return nil, err
	}
	err = s.repo.ExecTx(ctx, func(tx Repository) error {
		if err := tx.MarkQuoteSent(ctx, database.MarkQuoteSentParams{
			ID: id, WorkspaceID: ws, CustomerSnapshot: customerSnap, IssuerSnapshot: issuerSnap,
		}); err != nil {
			return err
		}
		return tx.InsertEvent(ctx, id, "sent", caller, "")
	})
	if err != nil {
		return nil, err
	}
	return s.load(ctx, ws, id)
}

func (s *Service) DuplicateQuote(ctx context.Context, workspaceID, callerID, quoteID string) (*QuoteResponse, error) {
	ws, caller, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return nil, err
	}
	id, err := parseID(quoteID, "quoteId")
	if err != nil {
		return nil, err
	}
	src, ok, err := s.repo.GetQuote(ctx, id, ws)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	srcItems, err := s.repo.ListItems(ctx, id)
	if err != nil {
		return nil, err
	}
	st, err := s.settingsOrDefault(ctx, ws)
	if err != nil {
		return nil, err
	}
	today := vietnamToday()
	var validUntil pgtype.Date
	if st.DefaultValidDays > 0 {
		validUntil = pgtype.Date{Time: today.AddDate(0, 0, int(st.DefaultValidDays)), Valid: true}
	}
	items := make([]database.InsertQuoteItemParams, 0, len(srcItems))
	for _, it := range srcItems {
		items = append(items, database.InsertQuoteItemParams{
			Position: it.Position, Name: it.Name, Description: it.Description, Unit: it.Unit,
			QuantityMilli: it.QuantityMilli, UnitPrice: it.UnitPrice, DiscountBp: it.DiscountBp, LineTotal: it.LineTotal,
		})
	}

	var newID pgtype.UUID
	err = s.repo.ExecTx(ctx, func(tx Repository) error {
		n, err := tx.NextNumber(ctx, ws, int32(today.Year()))
		if err != nil {
			return err
		}
		newID, err = tx.CreateQuote(ctx, database.CreateQuoteParams{
			WorkspaceID: ws, ProjectID: src.ProjectID, ModuleID: src.ModuleID, CustomerID: src.CustomerID,
			Number: fmt.Sprintf("%s-%d-%04d", st.NumberPrefix, today.Year(), n),
			Title:  src.Title, Currency: src.Currency, Language: src.Language,
			IssueDate: pgtype.Date{Time: today, Valid: true}, ValidUntil: validUntil,
			VatBp: src.VatBp, Terms: src.Terms, CreatedBy: caller,
		})
		if err != nil {
			return err
		}
		if err := tx.UpdateQuoteDraft(ctx, database.UpdateQuoteDraftParams{
			ID: newID, WorkspaceID: ws, CustomerID: src.CustomerID, Title: src.Title, Currency: src.Currency,
			Language: src.Language, IssueDate: pgtype.Date{Time: today, Valid: true}, ValidUntil: validUntil,
			DiscountType: src.DiscountType, DiscountValue: src.DiscountValue, VatBp: src.VatBp,
			Subtotal: src.Subtotal, DiscountTotal: src.DiscountTotal, TaxTotal: src.TaxTotal, GrandTotal: src.GrandTotal,
			Notes: src.Notes, Terms: src.Terms,
		}); err != nil {
			return err
		}
		if err := tx.ReplaceItems(ctx, newID, items); err != nil {
			return err
		}
		return tx.InsertEvent(ctx, newID, "created", caller, "duplicated:"+src.Number)
	})
	if err != nil {
		return nil, err
	}
	return s.load(ctx, ws, newID)
}

// SetStatus records a decision on a sent quote, or cancels a draft/sent one.
func (s *Service) SetStatus(ctx context.Context, workspaceID, callerID, quoteID string, req SetQuoteStatusRequest) (*QuoteResponse, error) {
	ws, caller, err := s.access(ctx, workspaceID, callerID, true)
	if err != nil {
		return nil, err
	}
	id, err := parseID(quoteID, "quoteId")
	if err != nil {
		return nil, err
	}
	row, ok, err := s.repo.GetQuote(ctx, id, ws)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	allowed := false
	switch req.Status {
	case StatusAccepted, StatusRejected:
		allowed = row.Status == StatusSent
	case StatusCancelled:
		allowed = row.Status == StatusDraft || row.Status == StatusSent
	}
	if !allowed {
		return nil, ErrBadTransition
	}
	err = s.repo.ExecTx(ctx, func(tx Repository) error {
		if err := tx.SetQuoteStatus(ctx, database.SetQuoteStatusParams{ID: id, WorkspaceID: ws, Status: req.Status}); err != nil {
			return err
		}
		return tx.InsertEvent(ctx, id, req.Status, caller, "")
	})
	if err != nil {
		return nil, err
	}
	return s.load(ctx, ws, id)
}
