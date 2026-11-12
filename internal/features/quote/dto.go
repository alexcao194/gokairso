package quote

import (
	"encoding/json"
	"math"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/currency"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	StatusDraft     = "draft"
	StatusSent      = "sent"
	StatusAccepted  = "accepted"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
	// StatusExpired is derived on read: a sent quote past its validity date.
	StatusExpired = "expired"
)

// ---- customers ----

type CustomerRequest struct {
	Name        string `json:"name"`
	TaxCode     string `json:"taxCode"`
	Address     string `json:"address"`
	ContactName string `json:"contactName"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Note        string `json:"note"`
}

type CustomerResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TaxCode     string    `json:"taxCode"`
	Address     string    `json:"address"`
	ContactName string    `json:"contactName"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

// PartyInfo is the customer or issuer block printed on a quote. For sent quotes
// it comes from the snapshot taken at send time.
type PartyInfo struct {
	Name         string `json:"name"`
	TaxCode      string `json:"taxCode"`
	Address      string `json:"address"`
	ContactName  string `json:"contactName,omitempty"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Website      string `json:"website,omitempty"`
	BankInfo     string `json:"bankInfo,omitempty"`
	LogoURL      string `json:"logoUrl,omitempty"`
	SignatureURL string `json:"signatureUrl,omitempty"`
}

// ---- settings ----

type QuoteSettingsRequest struct {
	CompanyName      string  `json:"companyName"`
	Address          string  `json:"address"`
	TaxCode          string  `json:"taxCode"`
	Phone            string  `json:"phone"`
	Email            string  `json:"email"`
	Website          string  `json:"website"`
	BankInfo         string  `json:"bankInfo"`
	LogoURL          string  `json:"logoUrl"`
	SignatureURL     string  `json:"signatureUrl"`
	NumberPrefix     string  `json:"numberPrefix"`
	DefaultCurrency  string  `json:"defaultCurrency"`
	DefaultVatRate   float64 `json:"defaultVatRate"`
	DefaultValidDays int     `json:"defaultValidDays"`
	DefaultTerms     string  `json:"defaultTerms"`
}

type QuoteSettingsResponse struct {
	CompanyName      string  `json:"companyName"`
	Address          string  `json:"address"`
	TaxCode          string  `json:"taxCode"`
	Phone            string  `json:"phone"`
	Email            string  `json:"email"`
	Website          string  `json:"website"`
	BankInfo         string  `json:"bankInfo"`
	LogoURL          string  `json:"logoUrl"`
	SignatureURL     string  `json:"signatureUrl"`
	NumberPrefix     string  `json:"numberPrefix"`
	DefaultCurrency  string  `json:"defaultCurrency"`
	DefaultVatRate   float64 `json:"defaultVatRate"`
	DefaultValidDays int     `json:"defaultValidDays"`
	DefaultTerms     string  `json:"defaultTerms"`
}

// ---- quotes ----

type CreateQuoteRequest struct {
	ModuleID   string  `json:"moduleId"`
	CustomerID *string `json:"customerId,omitempty"`
}

type QuoteItemRequest struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Unit            string  `json:"unit"`
	Quantity        float64 `json:"quantity"`
	UnitPrice       int64   `json:"unitPrice"`
	DiscountPercent float64 `json:"discountPercent"`
}

// UpdateQuoteRequest replaces a draft's header and all of its lines; the
// server recomputes every total. Money is in the currency's minor unit.
type UpdateQuoteRequest struct {
	CustomerID      *string            `json:"customerId"`
	Title           string             `json:"title"`
	Currency        string             `json:"currency"`
	Language        string             `json:"language"`
	IssueDate       string             `json:"issueDate" format:"date"`
	ValidUntil      string             `json:"validUntil" format:"date"`
	DiscountType    string             `json:"discountType"`
	DiscountPercent float64            `json:"discountPercent"`
	DiscountAmount  int64              `json:"discountAmount"`
	VatRate         float64            `json:"vatRate"`
	Notes           string             `json:"notes"`
	Terms           string             `json:"terms"`
	Items           []QuoteItemRequest `json:"items"`
}

type SetQuoteStatusRequest struct {
	Status string `json:"status"`
}

type QuoteItemResponse struct {
	ID              string  `json:"id"`
	Position        int     `json:"position"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Unit            string  `json:"unit"`
	Quantity        float64 `json:"quantity"`
	UnitPrice       int64   `json:"unitPrice"`
	DiscountPercent float64 `json:"discountPercent"`
	LineTotal       int64   `json:"lineTotal"`
}

type QuoteEventResponse struct {
	Type      string    `json:"type"`
	ActorName string    `json:"actorName"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
}

type QuoteResponse struct {
	ID              string               `json:"id"`
	Number          string               `json:"number"`
	Title           string               `json:"title"`
	Status          string               `json:"status"`
	Editable        bool                 `json:"editable"`
	Currency        string               `json:"currency"`
	MinorUnits      int                  `json:"minorUnits"`
	Language        string               `json:"language"`
	IssueDate       string               `json:"issueDate" format:"date"`
	ValidUntil      string               `json:"validUntil,omitempty" format:"date"`
	DiscountType    string               `json:"discountType"`
	DiscountPercent float64              `json:"discountPercent"`
	DiscountAmount  int64                `json:"discountAmount"`
	VatRate         float64              `json:"vatRate"`
	Subtotal        int64                `json:"subtotal"`
	DiscountTotal   int64                `json:"discountTotal"`
	TaxTotal        int64                `json:"taxTotal"`
	GrandTotal      int64                `json:"grandTotal"`
	AmountInWords   string               `json:"amountInWords"`
	ProjectID       string               `json:"projectId"`
	ProjectName     string               `json:"projectName"`
	ModuleID        string               `json:"moduleId"`
	ModuleName      string               `json:"moduleName"`
	CustomerID      string               `json:"customerId,omitempty"`
	Customer        PartyInfo            `json:"customer"`
	Issuer          PartyInfo            `json:"issuer"`
	Notes           string               `json:"notes"`
	Terms           string               `json:"terms"`
	Items           []QuoteItemResponse  `json:"items"`
	Events          []QuoteEventResponse `json:"events"`
	SentAt          *time.Time           `json:"sentAt,omitempty" swaggertype:"string" format:"date-time"`
	DecidedAt       *time.Time           `json:"decidedAt,omitempty" swaggertype:"string" format:"date-time"`
	CreatedAt       time.Time            `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt       time.Time            `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

type QuoteSummaryResponse struct {
	ID           string     `json:"id"`
	Number       string     `json:"number"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	Currency     string     `json:"currency"`
	MinorUnits   int        `json:"minorUnits"`
	IssueDate    string     `json:"issueDate" format:"date"`
	ValidUntil   string     `json:"validUntil,omitempty" format:"date"`
	GrandTotal   int64      `json:"grandTotal"`
	ProjectID    string     `json:"projectId"`
	ProjectName  string     `json:"projectName"`
	ModuleID     string     `json:"moduleId"`
	ModuleName   string     `json:"moduleName"`
	CustomerName string     `json:"customerName"`
	SentAt       *time.Time `json:"sentAt,omitempty" swaggertype:"string" format:"date-time"`
	CreatedAt    time.Time  `json:"createdAt" swaggertype:"string" format:"date-time"`
}

// ---- conversions ----

func bpToPercent(bp int32) float64 { return float64(bp) / 100 }
func percentToBp(p float64) int32  { return int32(math.Round(p * 100)) }
func milliToQty(m int64) float64   { return float64(m) / qtyScale }
func qtyToMilli(q float64) int64   { return int64(math.Round(q * qtyScale)) }

func dateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(time.DateOnly)
}

func parseDate(s string) (pgtype.Date, error) {
	if s == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func timePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func vietnamToday() time.Time {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// effectiveStatus reports "expired" for a sent quote whose validity has passed.
func effectiveStatus(status string, validUntil pgtype.Date) string {
	if status == StatusSent && validUntil.Valid && validUntil.Time.Before(vietnamToday()) {
		return StatusExpired
	}
	return status
}

func customerResponse(c database.Customer) CustomerResponse {
	return CustomerResponse{
		ID:          pgutil.UUIDToString(c.ID),
		Name:        c.Name,
		TaxCode:     c.TaxCode,
		Address:     c.Address,
		ContactName: c.ContactName,
		Email:       c.Email,
		Phone:       c.Phone,
		Note:        c.Note,
		CreatedAt:   c.CreatedAt.Time,
		UpdatedAt:   c.UpdatedAt.Time,
	}
}

func customerParty(c database.Customer) PartyInfo {
	return PartyInfo{
		Name:        c.Name,
		TaxCode:     c.TaxCode,
		Address:     c.Address,
		ContactName: c.ContactName,
		Email:       c.Email,
		Phone:       c.Phone,
	}
}

func issuerParty(s database.QuoteSetting) PartyInfo {
	return PartyInfo{
		Name:         s.CompanyName,
		TaxCode:      s.TaxCode,
		Address:      s.Address,
		Email:        s.Email,
		Phone:        s.Phone,
		Website:      s.Website,
		BankInfo:     s.BankInfo,
		LogoURL:      s.LogoUrl,
		SignatureURL: s.SignatureUrl,
	}
}

func partyFromSnapshot(raw []byte) PartyInfo {
	var p PartyInfo
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	return p
}

func defaultSettings(workspaceID pgtype.UUID) database.QuoteSetting {
	return database.QuoteSetting{
		WorkspaceID:      workspaceID,
		NumberPrefix:     "BG",
		DefaultCurrency:  "VND",
		DefaultValidDays: 30,
	}
}

func settingsResponse(s database.QuoteSetting) QuoteSettingsResponse {
	return QuoteSettingsResponse{
		CompanyName:      s.CompanyName,
		Address:          s.Address,
		TaxCode:          s.TaxCode,
		Phone:            s.Phone,
		Email:            s.Email,
		Website:          s.Website,
		BankInfo:         s.BankInfo,
		LogoURL:          s.LogoUrl,
		SignatureURL:     s.SignatureUrl,
		NumberPrefix:     s.NumberPrefix,
		DefaultCurrency:  s.DefaultCurrency,
		DefaultVatRate:   bpToPercent(s.DefaultVatBp),
		DefaultValidDays: int(s.DefaultValidDays),
		DefaultTerms:     s.DefaultTerms,
	}
}

func itemResponse(it database.QuoteItem) QuoteItemResponse {
	return QuoteItemResponse{
		ID:              pgutil.UUIDToString(it.ID),
		Position:        int(it.Position),
		Name:            it.Name,
		Description:     it.Description,
		Unit:            it.Unit,
		Quantity:        milliToQty(it.QuantityMilli),
		UnitPrice:       it.UnitPrice,
		DiscountPercent: bpToPercent(it.DiscountBp),
		LineTotal:       it.LineTotal,
	}
}

func eventResponse(e database.ListQuoteEventsRow) QuoteEventResponse {
	return QuoteEventResponse{Type: e.Type, ActorName: e.ActorName, Note: e.Note, CreatedAt: e.CreatedAt.Time}
}

// quoteResponse maps a quote row; customer/issuer come from the snapshot once
// sent, otherwise from the live customer and current settings.
func quoteResponse(row database.GetQuoteRow, items []database.QuoteItem, events []database.ListQuoteEventsRow, liveCustomer *database.Customer, settings database.QuoteSetting) QuoteResponse {
	resp := QuoteResponse{
		ID:            pgutil.UUIDToString(row.ID),
		Number:        row.Number,
		Title:         row.Title,
		Status:        effectiveStatus(row.Status, row.ValidUntil),
		Editable:      row.Status == StatusDraft,
		Currency:      row.Currency,
		MinorUnits:    currency.MinorUnits(row.Currency),
		Language:      row.Language,
		IssueDate:     dateString(row.IssueDate),
		ValidUntil:    dateString(row.ValidUntil),
		DiscountType:  row.DiscountType,
		VatRate:       bpToPercent(row.VatBp),
		Subtotal:      row.Subtotal,
		DiscountTotal: row.DiscountTotal,
		TaxTotal:      row.TaxTotal,
		GrandTotal:    row.GrandTotal,
		AmountInWords: amountInWords(row.GrandTotal, row.Currency, row.Language),
		ProjectID:     pgutil.UUIDToString(row.ProjectID),
		ProjectName:   row.ProjectName,
		ModuleID:      pgutil.UUIDToString(row.ModuleID),
		ModuleName:    row.ModuleName,
		CustomerID:    pgutil.UUIDToString(row.CustomerID),
		Notes:         row.Notes,
		Terms:         row.Terms,
		Items:         make([]QuoteItemResponse, 0, len(items)),
		Events:        make([]QuoteEventResponse, 0, len(events)),
		SentAt:        timePtr(row.SentAt),
		DecidedAt:     timePtr(row.DecidedAt),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
	if row.DiscountType == "amount" {
		resp.DiscountAmount = row.DiscountValue
	} else {
		resp.DiscountPercent = bpToPercent(int32(row.DiscountValue))
	}
	if row.Status == StatusDraft {
		if liveCustomer != nil {
			resp.Customer = customerParty(*liveCustomer)
		}
		resp.Issuer = issuerParty(settings)
	} else {
		resp.Customer = partyFromSnapshot(row.CustomerSnapshot)
		resp.Issuer = partyFromSnapshot(row.IssuerSnapshot)
	}
	for _, it := range items {
		resp.Items = append(resp.Items, itemResponse(it))
	}
	for _, e := range events {
		resp.Events = append(resp.Events, eventResponse(e))
	}
	return resp
}

func quoteSummaryResponse(row database.ListQuotesRow) QuoteSummaryResponse {
	customerName := row.CustomerName
	if snap := partyFromSnapshot(row.CustomerSnapshot); snap.Name != "" {
		customerName = snap.Name
	}
	return QuoteSummaryResponse{
		ID:           pgutil.UUIDToString(row.ID),
		Number:       row.Number,
		Title:        row.Title,
		Status:       effectiveStatus(row.Status, row.ValidUntil),
		Currency:     row.Currency,
		MinorUnits:   currency.MinorUnits(row.Currency),
		IssueDate:    dateString(row.IssueDate),
		ValidUntil:   dateString(row.ValidUntil),
		GrandTotal:   row.GrandTotal,
		ProjectID:    pgutil.UUIDToString(row.ProjectID),
		ProjectName:  row.ProjectName,
		ModuleID:     pgutil.UUIDToString(row.ModuleID),
		ModuleName:   row.ModuleName,
		CustomerName: customerName,
		SentAt:       timePtr(row.SentAt),
		CreatedAt:    row.CreatedAt.Time,
	}
}
