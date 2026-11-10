-- +goose Up

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public;

CREATE TYPE public.module_scrum_status AS ENUM (
    'backlog',
    'in_progress',
    'done',
    'released'
);

CREATE TYPE public.module_sprint_status AS ENUM (
    'planned',
    'active',
    'completed'
);

CREATE TYPE public.project_role AS ENUM (
    'holder',
    'admin',
    'member'
);

CREATE TYPE public.task_status AS ENUM (
    'backlog',
    'todo',
    'in_progress',
    'review',
    'done',
    'canceled'
);

CREATE TYPE public.workspace_role AS ENUM (
    'holder',
    'admin',
    'member'
);

-- +goose StatementBegin
CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TABLE public.attendance_correction_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    reason text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    decided_by uuid,
    decided_at timestamp with time zone,
    check_in_at timestamp with time zone,
    check_out_at timestamp with time zone,
    target_check_in_record_id uuid,
    target_check_out_record_id uuid,
    original_check_in_at timestamp with time zone,
    original_check_out_at timestamp with time zone,
    result_check_in_record_id uuid,
    result_check_out_record_id uuid,
    CONSTRAINT attendance_correction_requests_has_time CHECK (((check_in_at IS NOT NULL) OR (check_out_at IS NOT NULL))),
    CONSTRAINT attendance_correction_requests_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text])))
);

CREATE TABLE public.attendance_records (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    type character varying(32) NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    latitude double precision NOT NULL,
    longitude double precision NOT NULL,
    wifi_bssid character varying(255) NOT NULL,
    wifi_ssid character varying(255) DEFAULT ''::character varying NOT NULL,
    distance_meters double precision NOT NULL,
    CONSTRAINT attendance_records_type_check CHECK (((type)::text = ANY ((ARRAY['check_in'::character varying, 'check_out'::character varying])::text[])))
);

CREATE TABLE public.attendance_reminders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    session_id uuid NOT NULL,
    reminder_type character varying(32) NOT NULL,
    CONSTRAINT attendance_reminders_reminder_type_check CHECK (((reminder_type)::text = ANY ((ARRAY['check_in'::character varying, 'check_out'::character varying])::text[])))
);

CREATE TABLE public.customer_feedback_beneficiaries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    feedback_id uuid NOT NULL,
    user_id uuid NOT NULL,
    weight double precision DEFAULT 1 NOT NULL
);

CREATE TABLE public.customer_feedbacks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    creator_id uuid NOT NULL,
    feedback_text text NOT NULL,
    content text NOT NULL,
    estimate double precision DEFAULT 0 NOT NULL
);

CREATE TABLE public.daily_report_reminders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    report_date date NOT NULL
);

CREATE TABLE public.daily_reports (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    report_date date NOT NULL,
    done text DEFAULT ''::text NOT NULL,
    in_progress text DEFAULT ''::text NOT NULL,
    issues text DEFAULT ''::text NOT NULL
);

CREATE TABLE public.employee_monthly_attitudes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    month date NOT NULL,
    score double precision DEFAULT 0 NOT NULL,
    comment text,
    updated_by_id uuid
);

CREATE TABLE public.entity_attachments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    entity_type character varying(32) NOT NULL,
    entity_id uuid NOT NULL,
    uploader_id uuid NOT NULL,
    file_name character varying(255) NOT NULL,
    file_url character varying(2048) NOT NULL,
    content_type character varying(100),
    size_bytes bigint
);

CREATE TABLE public.entity_update_histories (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    entity_type character varying(32) NOT NULL,
    entity_id uuid NOT NULL,
    changed_by_id uuid NOT NULL,
    field character varying(64) NOT NULL,
    old_value text,
    new_value text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.file_folders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    workspace_id uuid NOT NULL,
    parent_id uuid,
    name character varying(255) NOT NULL,
    kind character varying(16) DEFAULT 'custom'::character varying NOT NULL,
    project_id uuid,
    module_id uuid,
    is_public boolean DEFAULT false NOT NULL,
    created_by uuid,
    CONSTRAINT file_folders_kind_check CHECK (((kind)::text = ANY ((ARRAY['project'::character varying, 'module'::character varying, 'custom'::character varying])::text[])))
);

CREATE TABLE public.file_versions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    file_id uuid NOT NULL,
    object_name character varying(1024) NOT NULL,
    content_type character varying(255) DEFAULT ''::character varying NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    created_by uuid
);

CREATE TABLE public.files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    workspace_id uuid NOT NULL,
    folder_id uuid,
    name character varying(255) NOT NULL,
    object_name character varying(1024) NOT NULL,
    content_type character varying(255) DEFAULT ''::character varying NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    is_public boolean DEFAULT false NOT NULL,
    created_by uuid
);

CREATE TABLE public.holidays (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    name_en character varying(255),
    type character varying(50) NOT NULL,
    date_type character varying(20) DEFAULT 'solar'::character varying NOT NULL,
    day integer NOT NULL,
    month integer NOT NULL,
    year integer,
    is_public_holiday boolean DEFAULT false NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.initiatives (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    proposer_id uuid NOT NULL,
    ref_member_id uuid NOT NULL,
    content text NOT NULL,
    status text DEFAULT 'proposed'::text NOT NULL,
    changes_after text,
    earliest_approval_date date,
    approved_at timestamp with time zone,
    estimate double precision DEFAULT 0 NOT NULL
);

CREATE TABLE public.late_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    late_user_id uuid NOT NULL,
    creator_id uuid NOT NULL,
    content text NOT NULL,
    late_time timestamp with time zone NOT NULL,
    estimate double precision DEFAULT 0 NOT NULL
);

CREATE TABLE public.meeting_minutes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    project_id uuid,
    title character varying(255) NOT NULL,
    meeting_date date NOT NULL,
    location text DEFAULT ''::text NOT NULL,
    attendees jsonb DEFAULT '[]'::jsonb NOT NULL,
    recorders jsonb DEFAULT '[]'::jsonb NOT NULL,
    summary text DEFAULT ''::text NOT NULL,
    items jsonb DEFAULT '[]'::jsonb NOT NULL,
    section_notes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_by_id uuid
);

CREATE TABLE public.member_work_days (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    day_of_week smallint NOT NULL,
    CONSTRAINT member_work_days_day_of_week_check CHECK (((day_of_week >= 0) AND (day_of_week <= 6)))
);

CREATE TABLE public.member_work_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    day_id uuid NOT NULL,
    start_time time without time zone NOT NULL,
    end_time time without time zone NOT NULL,
    CONSTRAINT member_work_sessions_check CHECK ((start_time < end_time))
);

CREATE TABLE public.module_sprints (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    module_id uuid NOT NULL,
    name character varying(120) NOT NULL,
    goal text DEFAULT ''::text NOT NULL,
    start_date date NOT NULL,
    end_date date NOT NULL,
    status public.module_sprint_status DEFAULT 'planned'::public.module_sprint_status NOT NULL,
    review_note text DEFAULT ''::text NOT NULL,
    retro_note text DEFAULT ''::text NOT NULL,
    CONSTRAINT module_sprints_check CHECK ((end_date >= start_date))
);

CREATE TABLE public.modules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    name character varying(255) NOT NULL,
    project_id uuid NOT NULL,
    start_date timestamp with time zone,
    end_date timestamp with time zone,
    scrum_status public.module_scrum_status DEFAULT 'backlog'::public.module_scrum_status NOT NULL,
    released_at timestamp with time zone
);

CREATE TABLE public.notifications (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    user_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    body text DEFAULT ''::text NOT NULL,
    data jsonb DEFAULT '{}'::jsonb NOT NULL,
    read_at timestamp with time zone
);

CREATE TABLE public.project_feedback_images (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    feedback_id uuid NOT NULL,
    url text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.project_feedback_settings (
    project_id uuid NOT NULL,
    token text NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    show_module_timeline boolean DEFAULT false NOT NULL
);

CREATE TABLE public.project_feedback_tasks (
    feedback_id uuid NOT NULL,
    task_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.project_feedbacks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    project_id uuid NOT NULL,
    submitter_name character varying(255) NOT NULL,
    submitter_company character varying(255) NOT NULL,
    feature_name character varying(255) NOT NULL,
    content text NOT NULL,
    priority integer NOT NULL,
    response text,
    responded_at timestamp with time zone,
    responded_by_id uuid,
    expected_fix_date timestamp with time zone,
    status text DEFAULT 'pending'::text NOT NULL,
    sequence_no integer NOT NULL,
    feedback_type text DEFAULT 'feature'::text NOT NULL,
    CONSTRAINT project_feedbacks_feedback_type_check CHECK ((feedback_type = ANY (ARRAY['feature'::text, 'improvement'::text, 'bug'::text]))),
    CONSTRAINT project_feedbacks_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'discussing'::text, 'acknowledged'::text, 'in_progress'::text, 'resolved'::text, 'deployed'::text, 'canceled'::text])))
);

CREATE TABLE public.project_files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    file_url text NOT NULL,
    file_name character varying(255) NOT NULL,
    content_type character varying(255) DEFAULT ''::character varying NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    created_by_id uuid,
    CONSTRAINT project_files_size_bytes_check CHECK ((size_bytes >= 0))
);

CREATE TABLE public.project_links (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    url text NOT NULL,
    created_by_id uuid
);

CREATE TABLE public.project_notes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    content text NOT NULL,
    created_by_id uuid
);

CREATE TABLE public.project_users (
    project_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role public.project_role NOT NULL
);

CREATE TABLE public.projects (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    name character varying(255) NOT NULL,
    workspace_id uuid NOT NULL,
    avatar character varying(2048),
    task_sequence_counter integer DEFAULT 0 NOT NULL,
    feedback_sequence_counter integer DEFAULT 0 NOT NULL
);

CREATE TABLE public.task_comment_mentions (
    comment_id uuid NOT NULL,
    task_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.task_comment_reads (
    task_id uuid NOT NULL,
    user_id uuid NOT NULL,
    last_read_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.task_comments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_id uuid NOT NULL,
    author_id uuid NOT NULL,
    content text NOT NULL
);

CREATE TABLE public.task_deadline_reminders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    task_id uuid NOT NULL,
    end_date timestamp with time zone NOT NULL
);

CREATE TABLE public.task_fault_holders (
    user_id uuid NOT NULL,
    task_id uuid NOT NULL,
    weight integer DEFAULT 1 NOT NULL
);

CREATE TABLE public.task_label_tasks (
    task_id uuid NOT NULL,
    label_id uuid NOT NULL
);

CREATE TABLE public.task_labels (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    created_by_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    color character varying(7) DEFAULT '#000000'::character varying NOT NULL,
    project_id uuid NOT NULL
);

CREATE TABLE public.task_workers (
    user_id uuid NOT NULL,
    task_id uuid NOT NULL,
    weight integer DEFAULT 1 NOT NULL
);

CREATE TABLE public.tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    project_id uuid NOT NULL,
    module_id uuid,
    created_by_id uuid NOT NULL,
    parent_id uuid,
    title character varying(255) NOT NULL,
    description text,
    status public.task_status DEFAULT 'todo'::public.task_status NOT NULL,
    priority integer NOT NULL,
    start_date timestamp with time zone,
    end_date timestamp with time zone,
    estimation integer DEFAULT 0 NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    first_completed_at timestamp with time zone,
    return_count integer DEFAULT 0 NOT NULL,
    fault_holder_id uuid,
    sequence_no integer NOT NULL
);

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    name character varying(255) NOT NULL,
    alias character varying(255) NOT NULL,
    avatar character varying(2048),
    firebase_uid character varying(255) NOT NULL,
    email character varying(320),
    email_verified boolean DEFAULT false NOT NULL,
    firebase_provider character varying(64) DEFAULT 'firebase'::character varying NOT NULL,
    last_sign_in_at timestamp with time zone,
    birthday date,
    bio text,
    phone character varying(50),
    nickname character varying(255)
);

CREATE TABLE public.whats_new_release_notes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    release_id uuid NOT NULL,
    type character varying(32) NOT NULL,
    content text NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT whats_new_release_notes_type_check CHECK (((type)::text = ANY ((ARRAY['new_feature'::character varying, 'improvement'::character varying, 'bug_fix'::character varying])::text[])))
);

CREATE TABLE public.whats_new_releases (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    version character varying(50) NOT NULL,
    published_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.work_schedule_days (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    schedule_date date NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    registered_by uuid,
    note text DEFAULT ''::text NOT NULL,
    week_note text DEFAULT ''::text NOT NULL,
    CONSTRAINT work_schedule_days_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text])))
);

CREATE TABLE public.work_schedule_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    day_id uuid NOT NULL,
    start_at timestamp with time zone NOT NULL,
    end_at timestamp with time zone NOT NULL,
    CONSTRAINT work_schedule_session_instants_order CHECK ((start_at < end_at))
);

CREATE TABLE public.workspace_attendance_settings (
    workspace_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    company_wifi_bssid character varying(255) NOT NULL,
    company_wifi_ssid character varying(255) DEFAULT ''::character varying NOT NULL,
    company_latitude double precision NOT NULL,
    company_longitude double precision NOT NULL,
    allowed_radius_meters integer NOT NULL,
    validation_enabled boolean DEFAULT true NOT NULL
);

CREATE TABLE public.workspace_file_settings (
    workspace_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    max_file_size_mb integer DEFAULT 50 NOT NULL,
    allowed_types text DEFAULT ''::text NOT NULL
);

CREATE TABLE public.workspace_files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    file_url text NOT NULL,
    file_name character varying(255) NOT NULL,
    content_type character varying(255) DEFAULT ''::character varying NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    created_by_id uuid,
    CONSTRAINT workspace_files_size_bytes_check CHECK ((size_bytes >= 0))
);

CREATE TABLE public.workspace_invitations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    invitee_id uuid NOT NULL,
    inviter_id uuid NOT NULL,
    role public.workspace_role DEFAULT 'member'::public.workspace_role NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT workspace_invitations_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'accepted'::text, 'rejected'::text, 'canceled'::text])))
);

CREATE TABLE public.workspace_kpi_settings (
    workspace_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    customer_weight double precision DEFAULT 10 NOT NULL,
    progress_weight double precision DEFAULT 35 NOT NULL,
    quality_weight double precision DEFAULT 30 NOT NULL,
    initiative_weight double precision DEFAULT 5 NOT NULL,
    punctuality_weight double precision DEFAULT 15 NOT NULL,
    attitude_weight double precision DEFAULT 5 NOT NULL
);

CREATE TABLE public.workspace_links (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    url text NOT NULL,
    created_by_id uuid
);

CREATE TABLE public.workspace_notes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    content text NOT NULL,
    created_by_id uuid
);

CREATE TABLE public.workspace_users (
    workspace_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role public.workspace_role NOT NULL
);

CREATE TABLE public.workspace_work_days (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    workspace_id uuid NOT NULL,
    day_of_week smallint NOT NULL,
    CONSTRAINT workspace_work_days_day_of_week_check CHECK (((day_of_week >= 0) AND (day_of_week <= 6)))
);

CREATE TABLE public.workspace_work_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    day_id uuid NOT NULL,
    start_time time without time zone NOT NULL,
    end_time time without time zone NOT NULL,
    CONSTRAINT workspace_work_sessions_check CHECK ((start_time < end_time))
);

CREATE TABLE public.workspaces (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    created_by_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    avatar character varying(2048)
);

ALTER TABLE ONLY public.attendance_correction_requests
    ADD CONSTRAINT attendance_correction_requests_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.attendance_records
    ADD CONSTRAINT attendance_records_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.attendance_reminders
    ADD CONSTRAINT attendance_reminders_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.attendance_reminders
    ADD CONSTRAINT attendance_reminders_session_id_reminder_type_key UNIQUE (session_id, reminder_type);

ALTER TABLE ONLY public.customer_feedback_beneficiaries
    ADD CONSTRAINT customer_feedback_beneficiaries_feedback_id_user_id_key UNIQUE (feedback_id, user_id);

ALTER TABLE ONLY public.customer_feedback_beneficiaries
    ADD CONSTRAINT customer_feedback_beneficiaries_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.customer_feedbacks
    ADD CONSTRAINT customer_feedbacks_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.daily_report_reminders
    ADD CONSTRAINT daily_report_reminders_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.daily_report_reminders
    ADD CONSTRAINT daily_report_reminders_workspace_id_member_id_report_date_key UNIQUE (workspace_id, member_id, report_date);

ALTER TABLE ONLY public.daily_reports
    ADD CONSTRAINT daily_reports_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.daily_reports
    ADD CONSTRAINT daily_reports_workspace_id_member_id_report_date_key UNIQUE (workspace_id, member_id, report_date);

ALTER TABLE ONLY public.employee_monthly_attitudes
    ADD CONSTRAINT employee_monthly_attitudes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.employee_monthly_attitudes
    ADD CONSTRAINT employee_monthly_attitudes_workspace_id_member_id_month_key UNIQUE (workspace_id, member_id, month);

ALTER TABLE ONLY public.entity_attachments
    ADD CONSTRAINT entity_attachments_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.entity_update_histories
    ADD CONSTRAINT entity_update_histories_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.file_folders
    ADD CONSTRAINT file_folders_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.file_versions
    ADD CONSTRAINT file_versions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.holidays
    ADD CONSTRAINT holidays_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.initiatives
    ADD CONSTRAINT initiatives_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.late_logs
    ADD CONSTRAINT late_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.meeting_minutes
    ADD CONSTRAINT meeting_minutes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.member_work_days
    ADD CONSTRAINT member_work_days_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.member_work_days
    ADD CONSTRAINT member_work_days_workspace_id_member_id_day_of_week_key UNIQUE (workspace_id, member_id, day_of_week);

ALTER TABLE ONLY public.member_work_sessions
    ADD CONSTRAINT member_work_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.module_sprints
    ADD CONSTRAINT module_sprints_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT modules_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_feedback_images
    ADD CONSTRAINT project_feedback_images_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_feedback_settings
    ADD CONSTRAINT project_feedback_settings_pkey PRIMARY KEY (project_id);

ALTER TABLE ONLY public.project_feedback_settings
    ADD CONSTRAINT project_feedback_settings_token_key UNIQUE (token);

ALTER TABLE ONLY public.project_feedback_tasks
    ADD CONSTRAINT project_feedback_tasks_pkey PRIMARY KEY (feedback_id, task_id);

ALTER TABLE ONLY public.project_feedbacks
    ADD CONSTRAINT project_feedbacks_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_files
    ADD CONSTRAINT project_files_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_links
    ADD CONSTRAINT project_links_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_notes
    ADD CONSTRAINT project_notes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.project_users
    ADD CONSTRAINT project_users_pkey PRIMARY KEY (project_id, user_id);

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task_comment_mentions
    ADD CONSTRAINT task_comment_mentions_pkey PRIMARY KEY (comment_id, user_id);

ALTER TABLE ONLY public.task_comment_reads
    ADD CONSTRAINT task_comment_reads_pkey PRIMARY KEY (task_id, user_id);

ALTER TABLE ONLY public.task_comments
    ADD CONSTRAINT task_comments_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task_deadline_reminders
    ADD CONSTRAINT task_deadline_reminders_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task_deadline_reminders
    ADD CONSTRAINT task_deadline_reminders_task_id_end_date_key UNIQUE (task_id, end_date);

ALTER TABLE ONLY public.task_fault_holders
    ADD CONSTRAINT task_fault_holders_pkey PRIMARY KEY (task_id, user_id);

ALTER TABLE ONLY public.task_label_tasks
    ADD CONSTRAINT task_label_tasks_pkey PRIMARY KEY (task_id, label_id);

ALTER TABLE ONLY public.task_labels
    ADD CONSTRAINT task_labels_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.task_labels
    ADD CONSTRAINT task_labels_project_id_name_key UNIQUE (project_id, name);

ALTER TABLE ONLY public.task_workers
    ADD CONSTRAINT task_workers_pkey PRIMARY KEY (task_id, user_id);

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_alias_key UNIQUE (alias);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_firebase_uid_key UNIQUE (firebase_uid);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.whats_new_release_notes
    ADD CONSTRAINT whats_new_release_notes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.whats_new_releases
    ADD CONSTRAINT whats_new_releases_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.work_schedule_days
    ADD CONSTRAINT work_schedule_days_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.work_schedule_days
    ADD CONSTRAINT work_schedule_days_workspace_id_member_id_schedule_date_key UNIQUE (workspace_id, member_id, schedule_date);

ALTER TABLE ONLY public.work_schedule_sessions
    ADD CONSTRAINT work_schedule_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.workspace_attendance_settings
    ADD CONSTRAINT workspace_attendance_settings_pkey PRIMARY KEY (workspace_id);

ALTER TABLE ONLY public.workspace_file_settings
    ADD CONSTRAINT workspace_file_settings_pkey PRIMARY KEY (workspace_id);

ALTER TABLE ONLY public.workspace_files
    ADD CONSTRAINT workspace_files_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.workspace_invitations
    ADD CONSTRAINT workspace_invitations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.workspace_invitations
    ADD CONSTRAINT workspace_invitations_workspace_id_invitee_id_key UNIQUE (workspace_id, invitee_id);

ALTER TABLE ONLY public.workspace_kpi_settings
    ADD CONSTRAINT workspace_kpi_settings_pkey PRIMARY KEY (workspace_id);

ALTER TABLE ONLY public.workspace_links
    ADD CONSTRAINT workspace_links_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.workspace_notes
    ADD CONSTRAINT workspace_notes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.workspace_users
    ADD CONSTRAINT workspace_users_pkey PRIMARY KEY (workspace_id, user_id);

ALTER TABLE ONLY public.workspace_work_days
    ADD CONSTRAINT workspace_work_days_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.workspace_work_days
    ADD CONSTRAINT workspace_work_days_workspace_id_day_of_week_key UNIQUE (workspace_id, day_of_week);

ALTER TABLE ONLY public.workspace_work_sessions
    ADD CONSTRAINT workspace_work_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspaces_pkey PRIMARY KEY (id);

CREATE INDEX idx_attendance_correction_requests_member ON public.attendance_correction_requests USING btree (workspace_id, member_id);

CREATE INDEX idx_attendance_correction_requests_status ON public.attendance_correction_requests USING btree (workspace_id, status);

CREATE INDEX idx_attendance_records_workspace_member_time ON public.attendance_records USING btree (workspace_id, member_id, recorded_at);

CREATE INDEX idx_daily_reports_ws_date ON public.daily_reports USING btree (workspace_id, report_date);

CREATE INDEX idx_entity_attachments_entity ON public.entity_attachments USING btree (entity_type, entity_id, created_at DESC);

CREATE INDEX idx_entity_update_histories_entity ON public.entity_update_histories USING btree (entity_type, entity_id, created_at DESC);

CREATE INDEX idx_file_folders_ws_parent ON public.file_folders USING btree (workspace_id, parent_id) WHERE (deleted_at IS NULL);

CREATE INDEX idx_file_versions_file ON public.file_versions USING btree (file_id, created_at DESC);

CREATE INDEX idx_files_ws_folder ON public.files USING btree (workspace_id, folder_id) WHERE (deleted_at IS NULL);

CREATE INDEX idx_meeting_minutes_project_date ON public.meeting_minutes USING btree (project_id, meeting_date DESC);

CREATE INDEX idx_meeting_minutes_workspace_date ON public.meeting_minutes USING btree (workspace_id, meeting_date DESC);

CREATE INDEX idx_member_work_days_ws_member ON public.member_work_days USING btree (workspace_id, member_id);

CREATE INDEX idx_member_work_sessions_day_id ON public.member_work_sessions USING btree (day_id);

CREATE INDEX idx_module_sprints_module_start ON public.module_sprints USING btree (module_id, start_date);

CREATE INDEX idx_modules_project_id ON public.modules USING btree (project_id);

CREATE INDEX idx_notifications_workspace_user_created_at ON public.notifications USING btree (workspace_id, user_id, created_at DESC);

CREATE INDEX idx_project_feedback_images_feedback_id ON public.project_feedback_images USING btree (feedback_id);

CREATE INDEX idx_project_feedback_tasks_task_id ON public.project_feedback_tasks USING btree (task_id);

CREATE INDEX idx_project_feedbacks_project_id ON public.project_feedbacks USING btree (project_id);

CREATE INDEX idx_project_files_project_id ON public.project_files USING btree (project_id);

CREATE INDEX idx_project_links_project_id ON public.project_links USING btree (project_id);

CREATE INDEX idx_project_notes_project_id ON public.project_notes USING btree (project_id);

CREATE INDEX idx_projects_workspace_id ON public.projects USING btree (workspace_id);

CREATE INDEX idx_task_comment_mentions_task_user ON public.task_comment_mentions USING btree (task_id, user_id);

CREATE INDEX idx_task_comments_task_created_at ON public.task_comments USING btree (task_id, created_at);

CREATE INDEX idx_task_fault_holders_user_id ON public.task_fault_holders USING btree (user_id);

CREATE INDEX idx_task_labels_project_id ON public.task_labels USING btree (project_id);

CREATE INDEX idx_task_workers_user_id ON public.task_workers USING btree (user_id);

CREATE INDEX idx_tasks_created_by_id ON public.tasks USING btree (created_by_id);

CREATE INDEX idx_tasks_fault_holder_id ON public.tasks USING btree (fault_holder_id);

CREATE INDEX idx_tasks_module_id ON public.tasks USING btree (module_id);

CREATE INDEX idx_tasks_project_id ON public.tasks USING btree (project_id);

CREATE INDEX idx_whats_new_release_notes_release_id ON public.whats_new_release_notes USING btree (release_id, "position");

CREATE INDEX idx_whats_new_releases_published_at ON public.whats_new_releases USING btree (published_at DESC);

CREATE INDEX idx_work_schedule_days_status ON public.work_schedule_days USING btree (workspace_id, status);

CREATE INDEX idx_work_schedule_days_workspace_member_date ON public.work_schedule_days USING btree (workspace_id, member_id, schedule_date);

CREATE INDEX idx_work_schedule_sessions_day_id ON public.work_schedule_sessions USING btree (day_id);

CREATE INDEX idx_work_schedule_sessions_end_at ON public.work_schedule_sessions USING btree (end_at);

CREATE INDEX idx_work_schedule_sessions_start_at ON public.work_schedule_sessions USING btree (start_at);

CREATE INDEX idx_workspace_files_workspace_id ON public.workspace_files USING btree (workspace_id);

CREATE INDEX idx_workspace_links_workspace_id ON public.workspace_links USING btree (workspace_id);

CREATE INDEX idx_workspace_notes_workspace_id ON public.workspace_notes USING btree (workspace_id);

CREATE INDEX idx_workspaces_created_by_id ON public.workspaces USING btree (created_by_id);

CREATE UNIQUE INDEX uq_file_folders_module ON public.file_folders USING btree (module_id) WHERE (((kind)::text = 'module'::text) AND (deleted_at IS NULL));

CREATE UNIQUE INDEX uq_file_folders_project ON public.file_folders USING btree (project_id) WHERE (((kind)::text = 'project'::text) AND (deleted_at IS NULL));

CREATE UNIQUE INDEX uq_module_sprints_one_active ON public.module_sprints USING btree (module_id) WHERE (status = 'active'::public.module_sprint_status);

CREATE TRIGGER update_attendance_correction_requests_updated_at BEFORE UPDATE ON public.attendance_correction_requests FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_attendance_records_updated_at BEFORE UPDATE ON public.attendance_records FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_daily_reports_updated_at BEFORE UPDATE ON public.daily_reports FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_file_folders_updated_at BEFORE UPDATE ON public.file_folders FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_files_updated_at BEFORE UPDATE ON public.files FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_holidays_updated_at BEFORE UPDATE ON public.holidays FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_meeting_minutes_updated_at BEFORE UPDATE ON public.meeting_minutes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_member_work_days_updated_at BEFORE UPDATE ON public.member_work_days FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_member_work_sessions_updated_at BEFORE UPDATE ON public.member_work_sessions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_module_sprints_updated_at BEFORE UPDATE ON public.module_sprints FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_modules_updated_at BEFORE UPDATE ON public.modules FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_notifications_updated_at BEFORE UPDATE ON public.notifications FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_project_feedback_settings_updated_at BEFORE UPDATE ON public.project_feedback_settings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_project_feedbacks_updated_at BEFORE UPDATE ON public.project_feedbacks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_project_files_updated_at BEFORE UPDATE ON public.project_files FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_project_links_updated_at BEFORE UPDATE ON public.project_links FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_project_notes_updated_at BEFORE UPDATE ON public.project_notes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON public.projects FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_task_comments_updated_at BEFORE UPDATE ON public.task_comments FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_task_labels_updated_at BEFORE UPDATE ON public.task_labels FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON public.tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_whats_new_release_notes_updated_at BEFORE UPDATE ON public.whats_new_release_notes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_whats_new_releases_updated_at BEFORE UPDATE ON public.whats_new_releases FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_work_schedule_days_updated_at BEFORE UPDATE ON public.work_schedule_days FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_work_schedule_sessions_updated_at BEFORE UPDATE ON public.work_schedule_sessions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspace_attendance_settings_updated_at BEFORE UPDATE ON public.workspace_attendance_settings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspace_file_settings_updated_at BEFORE UPDATE ON public.workspace_file_settings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspace_files_updated_at BEFORE UPDATE ON public.workspace_files FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspace_kpi_settings_updated_at BEFORE UPDATE ON public.workspace_kpi_settings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspace_links_updated_at BEFORE UPDATE ON public.workspace_links FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspace_notes_updated_at BEFORE UPDATE ON public.workspace_notes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspace_work_days_updated_at BEFORE UPDATE ON public.workspace_work_days FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspace_work_sessions_updated_at BEFORE UPDATE ON public.workspace_work_sessions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_workspaces_updated_at BEFORE UPDATE ON public.workspaces FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

ALTER TABLE ONLY public.attendance_correction_requests
    ADD CONSTRAINT attendance_correction_requests_decided_by_fkey FOREIGN KEY (decided_by) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.attendance_correction_requests
    ADD CONSTRAINT attendance_correction_requests_member_id_fkey FOREIGN KEY (member_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.attendance_correction_requests
    ADD CONSTRAINT attendance_correction_requests_result_check_in_record_id_fkey FOREIGN KEY (result_check_in_record_id) REFERENCES public.attendance_records(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.attendance_correction_requests
    ADD CONSTRAINT attendance_correction_requests_result_check_out_record_id_fkey FOREIGN KEY (result_check_out_record_id) REFERENCES public.attendance_records(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.attendance_correction_requests
    ADD CONSTRAINT attendance_correction_requests_target_check_in_record_id_fkey FOREIGN KEY (target_check_in_record_id) REFERENCES public.attendance_records(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.attendance_correction_requests
    ADD CONSTRAINT attendance_correction_requests_target_check_out_record_id_fkey FOREIGN KEY (target_check_out_record_id) REFERENCES public.attendance_records(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.attendance_correction_requests
    ADD CONSTRAINT attendance_correction_requests_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.attendance_records
    ADD CONSTRAINT attendance_records_member_id_fkey FOREIGN KEY (member_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.attendance_records
    ADD CONSTRAINT attendance_records_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.attendance_reminders
    ADD CONSTRAINT attendance_reminders_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.work_schedule_sessions(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.customer_feedback_beneficiaries
    ADD CONSTRAINT customer_feedback_beneficiaries_feedback_id_fkey FOREIGN KEY (feedback_id) REFERENCES public.customer_feedbacks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.customer_feedback_beneficiaries
    ADD CONSTRAINT customer_feedback_beneficiaries_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.customer_feedbacks
    ADD CONSTRAINT customer_feedbacks_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.customer_feedbacks
    ADD CONSTRAINT customer_feedbacks_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.daily_report_reminders
    ADD CONSTRAINT daily_report_reminders_member_id_fkey FOREIGN KEY (member_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.daily_report_reminders
    ADD CONSTRAINT daily_report_reminders_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.daily_reports
    ADD CONSTRAINT daily_reports_member_id_fkey FOREIGN KEY (member_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.daily_reports
    ADD CONSTRAINT daily_reports_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.employee_monthly_attitudes
    ADD CONSTRAINT employee_monthly_attitudes_member_id_fkey FOREIGN KEY (member_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.employee_monthly_attitudes
    ADD CONSTRAINT employee_monthly_attitudes_updated_by_id_fkey FOREIGN KEY (updated_by_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.employee_monthly_attitudes
    ADD CONSTRAINT employee_monthly_attitudes_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.entity_attachments
    ADD CONSTRAINT entity_attachments_uploader_id_fkey FOREIGN KEY (uploader_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.entity_update_histories
    ADD CONSTRAINT entity_update_histories_changed_by_id_fkey FOREIGN KEY (changed_by_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.file_folders
    ADD CONSTRAINT file_folders_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

ALTER TABLE ONLY public.file_folders
    ADD CONSTRAINT file_folders_module_id_fkey FOREIGN KEY (module_id) REFERENCES public.modules(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.file_folders
    ADD CONSTRAINT file_folders_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.file_folders(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.file_folders
    ADD CONSTRAINT file_folders_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.file_folders
    ADD CONSTRAINT file_folders_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.file_versions
    ADD CONSTRAINT file_versions_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

ALTER TABLE ONLY public.file_versions
    ADD CONSTRAINT file_versions_file_id_fkey FOREIGN KEY (file_id) REFERENCES public.files(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_folder_id_fkey FOREIGN KEY (folder_id) REFERENCES public.file_folders(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.initiatives
    ADD CONSTRAINT initiatives_proposer_id_fkey FOREIGN KEY (proposer_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.initiatives
    ADD CONSTRAINT initiatives_ref_member_id_fkey FOREIGN KEY (ref_member_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.initiatives
    ADD CONSTRAINT initiatives_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.late_logs
    ADD CONSTRAINT late_logs_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.late_logs
    ADD CONSTRAINT late_logs_late_user_id_fkey FOREIGN KEY (late_user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.late_logs
    ADD CONSTRAINT late_logs_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.meeting_minutes
    ADD CONSTRAINT meeting_minutes_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.meeting_minutes
    ADD CONSTRAINT meeting_minutes_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.meeting_minutes
    ADD CONSTRAINT meeting_minutes_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.member_work_days
    ADD CONSTRAINT member_work_days_member_id_fkey FOREIGN KEY (member_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.member_work_days
    ADD CONSTRAINT member_work_days_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.member_work_sessions
    ADD CONSTRAINT member_work_sessions_day_id_fkey FOREIGN KEY (day_id) REFERENCES public.member_work_days(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.module_sprints
    ADD CONSTRAINT module_sprints_module_id_fkey FOREIGN KEY (module_id) REFERENCES public.modules(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT modules_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_feedback_images
    ADD CONSTRAINT project_feedback_images_feedback_id_fkey FOREIGN KEY (feedback_id) REFERENCES public.project_feedbacks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_feedback_settings
    ADD CONSTRAINT project_feedback_settings_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_feedback_tasks
    ADD CONSTRAINT project_feedback_tasks_feedback_id_fkey FOREIGN KEY (feedback_id) REFERENCES public.project_feedbacks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_feedback_tasks
    ADD CONSTRAINT project_feedback_tasks_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_feedbacks
    ADD CONSTRAINT project_feedbacks_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_feedbacks
    ADD CONSTRAINT project_feedbacks_responded_by_id_fkey FOREIGN KEY (responded_by_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.project_files
    ADD CONSTRAINT project_files_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.project_files
    ADD CONSTRAINT project_files_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_links
    ADD CONSTRAINT project_links_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.project_links
    ADD CONSTRAINT project_links_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_notes
    ADD CONSTRAINT project_notes_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.project_notes
    ADD CONSTRAINT project_notes_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_users
    ADD CONSTRAINT project_users_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.project_users
    ADD CONSTRAINT project_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_comment_mentions
    ADD CONSTRAINT task_comment_mentions_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES public.task_comments(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_comment_mentions
    ADD CONSTRAINT task_comment_mentions_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_comment_mentions
    ADD CONSTRAINT task_comment_mentions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_comment_reads
    ADD CONSTRAINT task_comment_reads_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_comment_reads
    ADD CONSTRAINT task_comment_reads_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_comments
    ADD CONSTRAINT task_comments_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.task_comments
    ADD CONSTRAINT task_comments_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_deadline_reminders
    ADD CONSTRAINT task_deadline_reminders_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_fault_holders
    ADD CONSTRAINT task_fault_holders_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_fault_holders
    ADD CONSTRAINT task_fault_holders_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_label_tasks
    ADD CONSTRAINT task_label_tasks_label_id_fkey FOREIGN KEY (label_id) REFERENCES public.task_labels(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_label_tasks
    ADD CONSTRAINT task_label_tasks_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_labels
    ADD CONSTRAINT task_labels_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.task_labels
    ADD CONSTRAINT task_labels_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_workers
    ADD CONSTRAINT task_workers_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.task_workers
    ADD CONSTRAINT task_workers_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_fault_holder_id_fkey FOREIGN KEY (fault_holder_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_module_id_fkey FOREIGN KEY (module_id) REFERENCES public.modules(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.tasks(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.whats_new_release_notes
    ADD CONSTRAINT whats_new_release_notes_release_id_fkey FOREIGN KEY (release_id) REFERENCES public.whats_new_releases(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.work_schedule_days
    ADD CONSTRAINT work_schedule_days_member_id_fkey FOREIGN KEY (member_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.work_schedule_days
    ADD CONSTRAINT work_schedule_days_registered_by_fkey FOREIGN KEY (registered_by) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.work_schedule_days
    ADD CONSTRAINT work_schedule_days_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.work_schedule_sessions
    ADD CONSTRAINT work_schedule_sessions_day_id_fkey FOREIGN KEY (day_id) REFERENCES public.work_schedule_days(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_attendance_settings
    ADD CONSTRAINT workspace_attendance_settings_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_file_settings
    ADD CONSTRAINT workspace_file_settings_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_files
    ADD CONSTRAINT workspace_files_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.workspace_files
    ADD CONSTRAINT workspace_files_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_invitations
    ADD CONSTRAINT workspace_invitations_invitee_id_fkey FOREIGN KEY (invitee_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_invitations
    ADD CONSTRAINT workspace_invitations_inviter_id_fkey FOREIGN KEY (inviter_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_invitations
    ADD CONSTRAINT workspace_invitations_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_kpi_settings
    ADD CONSTRAINT workspace_kpi_settings_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_links
    ADD CONSTRAINT workspace_links_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.workspace_links
    ADD CONSTRAINT workspace_links_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_notes
    ADD CONSTRAINT workspace_notes_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.workspace_notes
    ADD CONSTRAINT workspace_notes_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_users
    ADD CONSTRAINT workspace_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_users
    ADD CONSTRAINT workspace_users_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_work_days
    ADD CONSTRAINT workspace_work_days_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspace_work_sessions
    ADD CONSTRAINT workspace_work_sessions_day_id_fkey FOREIGN KEY (day_id) REFERENCES public.workspace_work_days(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspaces_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES public.users(id);

INSERT INTO public.holidays VALUES ('5440ede6-3bb6-4a9e-8c22-85f4c36c8191', 'Tết Dương lịch', 'New Year''s Day', 'vietnam', 'solar', 1, 1, NULL, true, 'Ngày đầu tiên của năm dương lịch', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('f987e084-7412-41e9-8f1f-15291e35e57e', 'Ngày Giải phóng miền Nam', 'Reunification Day', 'vietnam', 'solar', 30, 4, NULL, true, 'Ngày giải phóng hoàn toàn miền Nam, thống nhất đất nước', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('2eaac09b-4584-486c-bf55-4727aead0653', 'Ngày Quốc tế Lao động', 'International Workers'' Day', 'vietnam', 'solar', 1, 5, NULL, true, 'Ngày kỷ niệm phong trào công nhân quốc tế', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('dbd9f7ac-cd6f-484b-8d5c-895e07613dba', 'Ngày Quốc khánh', 'National Day', 'vietnam', 'solar', 2, 9, NULL, true, 'Ngày tuyên ngôn độc lập khai sinh nước VNDCCH', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('68770290-757e-45f4-9003-aa9baa7782dd', 'Ngày Quốc khánh (Nghỉ thêm)', 'National Day (Additional Day)', 'vietnam', 'solar', 3, 9, NULL, true, 'Ngày nghỉ thêm của dịp Lễ Quốc khánh', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('cbee7f50-3a01-4469-be3f-6c7a3b606742', 'Ngày Quốc tế Phụ nữ', 'International Women''s Day', 'vietnam', 'solar', 8, 3, NULL, false, 'Ngày tôn vinh phụ nữ thế giới và Việt Nam', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('77b4ba5b-ce72-4025-ba04-7896bd5c8b79', 'Ngày Giải phóng Thủ đô', 'Capital Liberation Day', 'vietnam', 'solar', 10, 10, NULL, false, 'Ngày giải phóng Thủ đô Hà Nội', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('cdc390d6-e85b-49cf-91ee-4c6a2a73a14f', 'Ngày Phụ nữ Việt Nam', 'Vietnamese Women''s Day', 'vietnam', 'solar', 20, 10, NULL, false, 'Ngày kỷ niệm thành lập Hội Liên hiệp Phụ nữ Việt Nam', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('3e6a539f-22e7-42b9-843d-68ff04cd15ba', 'Ngày Nhà giáo Việt Nam', 'Vietnamese Teachers'' Day', 'vietnam', 'solar', 20, 11, NULL, false, 'Ngày tôn vinh các nhà giáo tại Việt Nam', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('88f86dda-52d9-401d-a592-f3b42a8b5fd5', 'Ngày Thành lập Quân đội Nhân dân Việt Nam', 'People''s Army of Vietnam Foundation Day', 'vietnam', 'solar', 22, 12, NULL, false, 'Ngày thành lập QĐND Việt Nam', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('4c8c2738-a11e-41c6-8296-cabd76775ef5', 'Lễ Giáng sinh', 'Christmas Day', 'world', 'solar', 25, 12, NULL, false, 'Ngày kỷ niệm sự ra đời của Chúa Giêsu', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('f1240a6a-00b8-4db5-b8a1-ec46f9d1c235', 'Ngày Cá tháng Tư', 'April Fools'' Day', 'world', 'solar', 1, 4, NULL, false, 'Ngày của những lời nói dối vô hại', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('542c8ac6-8503-40f8-980d-efea4d904dc1', 'Ngày Valentine', 'Valentine''s Day', 'world', 'solar', 14, 2, NULL, false, 'Ngày lễ tình nhân', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('e38d6726-6d8e-4415-ae8e-49d24d2110d2', 'Ngày Quốc tế Thiếu nhi', 'International Children''s Day', 'world', 'solar', 1, 6, NULL, false, 'Ngày bảo vệ và chăm sóc trẻ em', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('f3595fb3-468d-4352-8339-88605c1f532e', 'Ngày Quốc tế Hạnh phúc', 'International Day of Happiness', 'world', 'solar', 20, 3, NULL, false, 'Ngày tôn vinh hạnh phúc trên toàn cầu', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('5215e017-c23e-49ce-ba96-013489b60d9a', 'Ngày Halloween', 'Halloween', 'world', 'solar', 31, 10, NULL, false, 'Lễ hội hóa trang phương Tây', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('5f1dfbed-563c-45f1-9795-768ae8648d76', 'Ngày Quốc tế Nam giới', 'International Men''s Day', 'world', 'solar', 19, 11, NULL, false, 'Ngày tôn vinh nam giới', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('4c51dde4-b4fd-46dd-b910-0f35e9d030d0', 'Ngày Trái Đất', 'Earth Day', 'world', 'solar', 22, 4, NULL, false, 'Ngày bảo vệ môi trường Trái Đất', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('71e5a874-ff60-403b-bbda-2e699ba168c9', 'Ngày Quốc tế Người cao tuổi', 'International Day of Older Persons', 'world', 'solar', 1, 10, NULL, false, 'Ngày chăm sóc và bảo vệ người cao tuổi', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('e003b853-11e8-4e46-9afc-4770df922a46', 'Tết Nguyên Đán (30 Tết)', 'Lunar New Year Eve 2025', 'vietnam', 'lunar', 28, 1, 2025, true, 'Đêm giao thừa âm lịch 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('853c5585-8fb5-46d6-9437-276a6c7f49e9', 'Tết Nguyên Đán (Mùng 1)', 'Lunar New Year Day 1 2025', 'vietnam', 'lunar', 29, 1, 2025, true, 'Ngày mùng 1 Tết Nguyên Đán 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('60931924-ccb9-421b-bd13-3542d8ada02d', 'Tết Nguyên Đán (Mùng 2)', 'Lunar New Year Day 2 2025', 'vietnam', 'lunar', 30, 1, 2025, true, 'Ngày mùng 2 Tết Nguyên Đán 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('e9a9748f-4a57-44a3-a9d3-1e71586e6cb1', 'Tết Nguyên Đán (Mùng 3)', 'Lunar New Year Day 3 2025', 'vietnam', 'lunar', 31, 1, 2025, true, 'Ngày mùng 3 Tết Nguyên Đán 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('4c219dda-cafc-41fa-96a9-a7b06a61604b', 'Giỗ Tổ Hùng Vương', 'Hung Kings Commemoration Day 2025', 'vietnam', 'lunar', 7, 4, 2025, true, 'Ngày giỗ Tổ Hùng Vương 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('7831f81d-b04a-4bd3-96e3-26c29d703d11', 'Tết Đoan ngọ', 'Dragon Boat Festival 2025', 'vietnam', 'lunar', 31, 5, 2025, false, 'Ngày diệt sâu bọ 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('c9f235fd-27e0-4c56-9285-03f04ce7851e', 'Lễ Vu lan', 'Ghost Festival 2025', 'vietnam', 'lunar', 6, 9, 2025, false, 'Lễ Vu lan báo hiếu 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('2b8c54e3-53fc-402f-b7f8-9d9fe0c79354', 'Tết Trung thu', 'Mid-Autumn Festival 2025', 'vietnam', 'lunar', 6, 10, 2025, false, 'Tết Trung thu 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('b8e83111-5dc1-40ac-b5fa-49d3cf97d258', 'Ông Công Ông Táo', 'Kitchen Gods'' Day 2025', 'vietnam', 'lunar', 22, 1, 2025, false, 'Ngày đưa tiễn ông Táo về trời 2025', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('066adffa-da25-4411-884d-a5079e86637f', 'Tết Nguyên Đán (30 Tết)', 'Lunar New Year Eve 2026', 'vietnam', 'lunar', 16, 2, 2026, true, 'Đêm giao thừa âm lịch 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('196fa708-2461-4bf9-8b39-7338b71f9f43', 'Tết Nguyên Đán (Mùng 1)', 'Lunar New Year Day 1 2026', 'vietnam', 'lunar', 17, 2, 2026, true, 'Ngày mùng 1 Tết Nguyên Đán 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('6f0a13dd-474e-4396-9436-2af575db5bb3', 'Tết Nguyên Đán (Mùng 2)', 'Lunar New Year Day 2 2026', 'vietnam', 'lunar', 18, 2, 2026, true, 'Ngày mùng 2 Tết Nguyên Đán 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('86424982-0126-4f86-82ad-d513643ec152', 'Tết Nguyên Đán (Mùng 3)', 'Lunar New Year Day 3 2026', 'vietnam', 'lunar', 19, 2, 2026, true, 'Ngày mùng 3 Tết Nguyên Đán 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('4b333f15-250d-41c3-864d-c644da1fb37a', 'Giỗ Tổ Hùng Vương', 'Hung Kings Commemoration Day 2026', 'vietnam', 'lunar', 26, 4, 2026, true, 'Ngày giỗ Tổ Hùng Vương 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('28385053-81fc-4014-ae34-ed3401825848', 'Tết Đoan ngọ', 'Dragon Boat Festival 2026', 'vietnam', 'lunar', 19, 6, 2026, false, 'Ngày diệt sâu bọ 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('b34bbe63-7b94-4565-9dfa-f8bcbcee6a7f', 'Lễ Vu lan', 'Ghost Festival 2026', 'vietnam', 'lunar', 27, 8, 2026, false, 'Lễ Vu lan báo hiếu 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('a16e3006-9ad4-4431-b979-967bcd4be69e', 'Tết Trung thu', 'Mid-Autumn Festival 2026', 'vietnam', 'lunar', 25, 9, 2026, false, 'Tết Trung thu 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('45af145d-825e-4cc4-813a-c066b7125ef7', 'Ông Công Ông Táo', 'Kitchen Gods'' Day 2026', 'vietnam', 'lunar', 10, 1, 2026, false, 'Ngày đưa tiễn ông Táo về trời 2026', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('25c8d804-eb97-429c-8b5e-acd2e8e72bdd', 'Tết Nguyên Đán (30 Tết)', 'Lunar New Year Eve 2027', 'vietnam', 'lunar', 5, 2, 2027, true, 'Đêm giao thừa âm lịch 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('89c47e45-66b6-454d-90b1-ddd7d61d24b7', 'Tết Nguyên Đán (Mùng 1)', 'Lunar New Year Day 1 2027', 'vietnam', 'lunar', 6, 2, 2027, true, 'Ngày mùng 1 Tết Nguyên Đán 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('fcf89763-fb01-47da-a5b8-d3845180d710', 'Tết Nguyên Đán (Mùng 2)', 'Lunar New Year Day 2 2027', 'vietnam', 'lunar', 7, 2, 2027, true, 'Ngày mùng 2 Tết Nguyên Đán 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('91897101-0a4f-4759-b9f5-fb4a14de4920', 'Tết Nguyên Đán (Mùng 3)', 'Lunar New Year Day 3 2027', 'vietnam', 'lunar', 8, 2, 2027, true, 'Ngày mùng 3 Tết Nguyên Đán 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('f92c6011-d610-42a8-b0a5-c1f0eaf21b05', 'Giỗ Tổ Hùng Vương', 'Hung Kings Commemoration Day 2027', 'vietnam', 'lunar', 15, 4, 2027, true, 'Ngày giỗ Tổ Hùng Vương 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('65347dfe-938f-4382-af8d-ebfb5d6f941c', 'Tết Đoan ngọ', 'Dragon Boat Festival 2027', 'vietnam', 'lunar', 9, 6, 2027, false, 'Ngày diệt sâu bọ 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('2d8b7ff4-67a5-4bd2-a0ac-b79941f1c5ec', 'Lễ Vu lan', 'Ghost Festival 2027', 'vietnam', 'lunar', 16, 8, 2027, false, 'Lễ Vu lan báo hiếu 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('b34044ea-e48d-42da-836d-3e5fd7711bbd', 'Tết Trung thu', 'Mid-Autumn Festival 2027', 'vietnam', 'lunar', 15, 9, 2027, false, 'Tết Trung thu 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('ed774720-2f46-4c16-b135-30574de74d17', 'Ông Công Ông Táo', 'Kitchen Gods'' Day 2027', 'vietnam', 'lunar', 30, 1, 2027, false, 'Ngày đưa tiễn ông Táo về trời 2027', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('62b44794-9fdf-4681-a82b-23bc1a5c92ad', 'Tết Nguyên Đán (30 Tết)', 'Lunar New Year Eve 2028', 'vietnam', 'lunar', 25, 1, 2028, true, 'Đêm giao thừa âm lịch 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('ff7ddd41-4a20-4658-8b25-458c467d54c0', 'Tết Nguyên Đán (Mùng 1)', 'Lunar New Year Day 1 2028', 'vietnam', 'lunar', 26, 1, 2028, true, 'Ngày mùng 1 Tết Nguyên Đán 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('079360f2-198a-42e0-b7b4-9f4006fa9e61', 'Tết Nguyên Đán (Mùng 2)', 'Lunar New Year Day 2 2028', 'vietnam', 'lunar', 27, 1, 2028, true, 'Ngày mùng 2 Tết Nguyên Đán 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('00423e20-c085-4be9-b205-5807cefd6d3d', 'Tết Nguyên Đán (Mùng 3)', 'Lunar New Year Day 3 2028', 'vietnam', 'lunar', 28, 1, 2028, true, 'Ngày mùng 3 Tết Nguyên Đán 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('9bffb47e-9c75-437d-af64-c960a84c14ed', 'Giỗ Tổ Hùng Vương', 'Hung Kings Commemoration Day 2028', 'vietnam', 'lunar', 4, 4, 2028, true, 'Ngày giỗ Tổ Hùng Vương 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('bd310d60-afcb-4d59-9f7c-e9cbec524b9c', 'Tết Đoan ngọ', 'Dragon Boat Festival 2028', 'vietnam', 'lunar', 28, 5, 2028, false, 'Ngày diệt sâu bọ 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('44283303-4adb-4949-89d4-ebc38adb5e7c', 'Lễ Vu lan', 'Ghost Festival 2028', 'vietnam', 'lunar', 3, 9, 2028, false, 'Lễ Vu lan báo hiếu 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('08c155ff-b1ef-49df-a9d4-7538e1b85186', 'Tết Trung thu', 'Mid-Autumn Festival 2028', 'vietnam', 'lunar', 3, 10, 2028, false, 'Tết Trung thu 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');
INSERT INTO public.holidays VALUES ('73896cd1-fa2b-4572-ad11-552107ac90ca', 'Ông Công Ông Táo', 'Kitchen Gods'' Day 2028', 'vietnam', 'lunar', 20, 1, 2028, false, 'Ngày đưa tiễn ông Táo về trời 2028', '2026-11-10 14:38:31.913034+07', '2026-11-10 14:38:31.913034+07');

-- +goose Down
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
