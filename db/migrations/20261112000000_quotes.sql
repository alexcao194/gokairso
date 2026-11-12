-- +goose Up
CREATE TABLE customers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    tax_code character varying(64) DEFAULT '' NOT NULL,
    address text DEFAULT '' NOT NULL,
    contact_name character varying(255) DEFAULT '' NOT NULL,
    email character varying(320) DEFAULT '' NOT NULL,
    phone character varying(50) DEFAULT '' NOT NULL,
    note text DEFAULT '' NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT customers_pkey PRIMARY KEY (id),
    CONSTRAINT customers_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE INDEX idx_customers_workspace_id ON customers USING btree (workspace_id);
CREATE TRIGGER update_customers_updated_at BEFORE UPDATE ON customers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE quote_settings (
    workspace_id uuid NOT NULL,
    company_name character varying(255) DEFAULT '' NOT NULL,
    address text DEFAULT '' NOT NULL,
    tax_code character varying(64) DEFAULT '' NOT NULL,
    phone character varying(50) DEFAULT '' NOT NULL,
    email character varying(320) DEFAULT '' NOT NULL,
    website character varying(255) DEFAULT '' NOT NULL,
    bank_info text DEFAULT '' NOT NULL,
    logo_url character varying(2048) DEFAULT '' NOT NULL,
    signature_url character varying(2048) DEFAULT '' NOT NULL,
    number_prefix character varying(16) DEFAULT 'BG' NOT NULL,
    default_currency character(3) DEFAULT 'VND' NOT NULL,
    default_vat_bp integer DEFAULT 0 NOT NULL,
    default_valid_days integer DEFAULT 30 NOT NULL,
    default_terms text DEFAULT '' NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT quote_settings_pkey PRIMARY KEY (workspace_id),
    CONSTRAINT quote_settings_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE TRIGGER update_quote_settings_updated_at BEFORE UPDATE ON quote_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE quote_counters (
    workspace_id uuid NOT NULL,
    year integer NOT NULL,
    last_number integer NOT NULL,
    CONSTRAINT quote_counters_pkey PRIMARY KEY (workspace_id, year),
    CONSTRAINT quote_counters_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE TABLE quotes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    project_id uuid NOT NULL,
    module_id uuid NOT NULL,
    customer_id uuid,
    number character varying(64) NOT NULL,
    title character varying(255) DEFAULT '' NOT NULL,
    status character varying(16) DEFAULT 'draft' NOT NULL,
    currency character(3) NOT NULL,
    language character varying(8) DEFAULT 'vi' NOT NULL,
    issue_date date DEFAULT CURRENT_DATE NOT NULL,
    valid_until date,
    discount_type character varying(8) DEFAULT 'percent' NOT NULL,
    discount_value bigint DEFAULT 0 NOT NULL,
    vat_bp integer DEFAULT 0 NOT NULL,
    subtotal bigint DEFAULT 0 NOT NULL,
    discount_total bigint DEFAULT 0 NOT NULL,
    tax_total bigint DEFAULT 0 NOT NULL,
    grand_total bigint DEFAULT 0 NOT NULL,
    customer_snapshot jsonb DEFAULT '{}'::jsonb NOT NULL,
    issuer_snapshot jsonb DEFAULT '{}'::jsonb NOT NULL,
    notes text DEFAULT '' NOT NULL,
    terms text DEFAULT '' NOT NULL,
    created_by uuid,
    sent_at timestamp with time zone,
    decided_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT quotes_pkey PRIMARY KEY (id),
    CONSTRAINT quotes_workspace_number_key UNIQUE (workspace_id, number),
    CONSTRAINT quotes_status_check CHECK (status IN ('draft', 'sent', 'accepted', 'rejected', 'cancelled')),
    CONSTRAINT quotes_discount_type_check CHECK (discount_type IN ('percent', 'amount')),
    CONSTRAINT quotes_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT quotes_project_id_fkey FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT quotes_module_id_fkey FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
    CONSTRAINT quotes_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL,
    CONSTRAINT quotes_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX idx_quotes_workspace_created_at ON quotes USING btree (workspace_id, created_at DESC);
CREATE INDEX idx_quotes_module_id ON quotes USING btree (module_id);
CREATE TRIGGER update_quotes_updated_at BEFORE UPDATE ON quotes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE quote_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    quote_id uuid NOT NULL,
    "position" integer NOT NULL,
    name character varying(500) NOT NULL,
    description text DEFAULT '' NOT NULL,
    unit character varying(32) DEFAULT '' NOT NULL,
    quantity_milli bigint DEFAULT 1000 NOT NULL,
    unit_price bigint DEFAULT 0 NOT NULL,
    discount_bp integer DEFAULT 0 NOT NULL,
    line_total bigint DEFAULT 0 NOT NULL,
    CONSTRAINT quote_items_pkey PRIMARY KEY (id),
    CONSTRAINT quote_items_quote_id_fkey FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE
);
CREATE INDEX idx_quote_items_quote_id ON quote_items USING btree (quote_id, "position");

CREATE TABLE quote_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    quote_id uuid NOT NULL,
    type character varying(32) NOT NULL,
    actor_id uuid,
    note text DEFAULT '' NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT quote_events_pkey PRIMARY KEY (id),
    CONSTRAINT quote_events_quote_id_fkey FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE,
    CONSTRAINT quote_events_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX idx_quote_events_quote_id ON quote_events USING btree (quote_id, created_at);

-- +goose Down
DROP TABLE quote_events;
DROP TABLE quote_items;
DROP TABLE quotes;
DROP TABLE quote_counters;
DROP TABLE quote_settings;
DROP TABLE customers;
