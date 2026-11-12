-- name: ListCustomers :many
SELECT id, workspace_id, name, tax_code, address, contact_name, email, phone, note, created_at, updated_at
FROM customers
WHERE workspace_id = $1
ORDER BY name;

-- name: GetCustomer :one
SELECT id, workspace_id, name, tax_code, address, contact_name, email, phone, note, created_at, updated_at
FROM customers
WHERE id = $1 AND workspace_id = $2;

-- name: CreateCustomer :one
INSERT INTO customers (workspace_id, name, tax_code, address, contact_name, email, phone, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, workspace_id, name, tax_code, address, contact_name, email, phone, note, created_at, updated_at;

-- name: UpdateCustomer :one
UPDATE customers
SET name = $3, tax_code = $4, address = $5, contact_name = $6, email = $7, phone = $8, note = $9
WHERE id = $1 AND workspace_id = $2
RETURNING id, workspace_id, name, tax_code, address, contact_name, email, phone, note, created_at, updated_at;

-- name: DeleteCustomer :execrows
DELETE FROM customers WHERE id = $1 AND workspace_id = $2;

-- name: GetQuoteSettings :one
SELECT workspace_id, company_name, address, tax_code, phone, email, website, bank_info, logo_url, signature_url,
       number_prefix, default_currency, default_vat_bp, default_valid_days, default_terms, created_at, updated_at
FROM quote_settings
WHERE workspace_id = $1;

-- name: UpsertQuoteSettings :one
INSERT INTO quote_settings (workspace_id, company_name, address, tax_code, phone, email, website, bank_info,
                            logo_url, signature_url, number_prefix, default_currency, default_vat_bp,
                            default_valid_days, default_terms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (workspace_id) DO UPDATE
SET company_name = EXCLUDED.company_name, address = EXCLUDED.address, tax_code = EXCLUDED.tax_code,
    phone = EXCLUDED.phone, email = EXCLUDED.email, website = EXCLUDED.website, bank_info = EXCLUDED.bank_info,
    logo_url = EXCLUDED.logo_url, signature_url = EXCLUDED.signature_url, number_prefix = EXCLUDED.number_prefix,
    default_currency = EXCLUDED.default_currency, default_vat_bp = EXCLUDED.default_vat_bp,
    default_valid_days = EXCLUDED.default_valid_days, default_terms = EXCLUDED.default_terms
RETURNING workspace_id, company_name, address, tax_code, phone, email, website, bank_info, logo_url, signature_url,
          number_prefix, default_currency, default_vat_bp, default_valid_days, default_terms, created_at, updated_at;

-- name: NextQuoteNumber :one
INSERT INTO quote_counters (workspace_id, year, last_number)
VALUES ($1, $2, 1)
ON CONFLICT (workspace_id, year) DO UPDATE SET last_number = quote_counters.last_number + 1
RETURNING last_number;

-- name: GetModuleContext :one
SELECT m.id AS module_id, m.name AS module_name, p.id AS project_id, p.name AS project_name, p.workspace_id
FROM modules m
JOIN projects p ON p.id = m.project_id
WHERE m.id = $1 AND m.deleted_at IS NULL AND p.deleted_at IS NULL;

-- name: CreateQuote :one
INSERT INTO quotes (workspace_id, project_id, module_id, customer_id, number, title, currency, language,
                    issue_date, valid_until, vat_bp, terms, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id;

-- name: GetQuote :one
SELECT q.id, q.workspace_id, q.project_id, q.module_id, q.customer_id, q.number, q.title, q.status, q.currency,
       q.language, q.issue_date, q.valid_until, q.discount_type, q.discount_value, q.vat_bp, q.subtotal,
       q.discount_total, q.tax_total, q.grand_total, q.customer_snapshot, q.issuer_snapshot, q.notes, q.terms,
       q.created_by, q.sent_at, q.decided_at, q.created_at, q.updated_at,
       m.name AS module_name, p.name AS project_name,
       COALESCE(c.name, '') AS customer_name
FROM quotes q
JOIN modules m ON m.id = q.module_id
JOIN projects p ON p.id = q.project_id
LEFT JOIN customers c ON c.id = q.customer_id
WHERE q.id = $1 AND q.workspace_id = $2;

-- name: ListQuotes :many
SELECT q.id, q.project_id, q.module_id, q.customer_id, q.number, q.title, q.status, q.currency, q.issue_date,
       q.valid_until, q.grand_total, q.customer_snapshot, q.sent_at, q.created_at, q.updated_at,
       m.name AS module_name, p.name AS project_name,
       COALESCE(c.name, '') AS customer_name
FROM quotes q
JOIN modules m ON m.id = q.module_id
JOIN projects p ON p.id = q.project_id
LEFT JOIN customers c ON c.id = q.customer_id
WHERE q.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.narg(module_id)::uuid IS NULL OR q.module_id = sqlc.narg(module_id))
  AND (sqlc.narg(project_id)::uuid IS NULL OR q.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(status)::text IS NULL OR q.status = sqlc.narg(status))
ORDER BY q.created_at DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: UpdateQuoteDraft :exec
UPDATE quotes
SET customer_id = $3, title = $4, currency = $5, language = $6, issue_date = $7, valid_until = $8,
    discount_type = $9, discount_value = $10, vat_bp = $11, subtotal = $12, discount_total = $13,
    tax_total = $14, grand_total = $15, notes = $16, terms = $17
WHERE id = $1 AND workspace_id = $2 AND status = 'draft';

-- name: MarkQuoteSent :exec
UPDATE quotes
SET status = 'sent', sent_at = now(), customer_snapshot = $3, issuer_snapshot = $4
WHERE id = $1 AND workspace_id = $2 AND status = 'draft';

-- name: SetQuoteStatus :exec
UPDATE quotes
SET status = sqlc.arg(status)::text,
    decided_at = CASE WHEN sqlc.arg(status)::text IN ('accepted', 'rejected') THEN now() ELSE decided_at END
WHERE id = $1 AND workspace_id = $2;

-- name: DeleteQuote :execrows
DELETE FROM quotes WHERE id = $1 AND workspace_id = $2 AND status = 'draft';

-- name: ListQuoteItems :many
SELECT id, quote_id, "position", name, description, unit, quantity_milli, unit_price, discount_bp, line_total
FROM quote_items
WHERE quote_id = $1
ORDER BY "position";

-- name: DeleteQuoteItems :exec
DELETE FROM quote_items WHERE quote_id = $1;

-- name: InsertQuoteItem :exec
INSERT INTO quote_items (quote_id, "position", name, description, unit, quantity_milli, unit_price, discount_bp, line_total)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: InsertQuoteEvent :exec
INSERT INTO quote_events (quote_id, type, actor_id, note)
VALUES ($1, $2, $3, $4);

-- name: ListQuoteEvents :many
SELECT e.id, e.type, e.actor_id, e.note, e.created_at, COALESCE(u.name, '') AS actor_name
FROM quote_events e
LEFT JOIN users u ON u.id = e.actor_id
WHERE e.quote_id = $1
ORDER BY e.created_at;
