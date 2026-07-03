-- +goose Up
-- smartscan baseline schema.
-- Generated from the GORM models via cmd/schemadump (PostgreSQL 18, uuidv7()).
-- PostgreSQL 18+ is required for the native uuidv7() default.

--
-- PostgreSQL database dump
--

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it

--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS '';

--
-- Name: activity_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.activity_logs (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_id uuid,
    tenant_id uuid,
    action_type character varying(100),
    entity_type character varying(100),
    entity_id uuid,
    old_values jsonb,
    new_values jsonb,
    ip_address inet,
    user_agent text,
    created_at timestamp with time zone
);

--
-- Name: app_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_settings (
    id uuid DEFAULT uuidv7() NOT NULL,
    setting_key character varying(100) NOT NULL,
    setting_value jsonb NOT NULL,
    updated_by uuid,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

--
-- Name: certification_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.certification_types (
    id uuid DEFAULT uuidv7() NOT NULL,
    country_code character varying(2),
    code character varying(50) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    logo_url text,
    website_url text,
    is_active boolean DEFAULT true,
    display_order bigint DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: cities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cities (
    id bigint NOT NULL,
    province_id bigint NOT NULL,
    country_code character varying(2) NOT NULL,
    name character varying(100) NOT NULL,
    postal_code_prefix character varying(10),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: cities_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: cities_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cities_id_seq OWNED BY public.cities.id;

--
-- Name: counterfeit_detections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.counterfeit_detections (
    id uuid DEFAULT uuidv7() NOT NULL,
    qr_code_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    detection_reason text,
    interaction_ids jsonb,
    total_interactions_count bigint,
    first_interaction_at timestamp with time zone,
    last_interaction_at timestamp with time zone,
    status character varying(20) DEFAULT 'active'::character varying,
    resolved_by uuid,
    resolved_at timestamp with time zone,
    created_at timestamp with time zone
);

--
-- Name: counterfeit_reports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.counterfeit_reports (
    id uuid DEFAULT uuidv7() NOT NULL,
    qr_code_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    counterfeit_detection_id uuid,
    description text,
    photos jsonb,
    store_name character varying(255),
    province character varying(100),
    city character varying(100),
    ip_address inet,
    user_agent text,
    geolocation jsonb,
    created_at timestamp with time zone
);

--
-- Name: countries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.countries (
    code character varying(2) NOT NULL,
    name character varying(100) NOT NULL,
    phone_code character varying(5),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: geofence_violations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.geofence_violations (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    batch_id uuid NOT NULL,
    qr_code_id uuid,
    product_id uuid,
    interaction_id uuid,
    scan_latitude numeric(10,7) NOT NULL,
    scan_longitude numeric(10,7) NOT NULL,
    distance_from_center_km numeric(8,2) NOT NULL,
    distance_from_edge_km numeric(8,2) NOT NULL,
    gps_accuracy_meters numeric(8,2),
    severity character varying(20) NOT NULL,
    created_at timestamp with time zone
);

--
-- Name: geofence_zone_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.geofence_zone_templates (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    template_name character varying(255) NOT NULL,
    latitude numeric(10,7) NOT NULL,
    longitude numeric(10,7) NOT NULL,
    radius_km numeric(6,1) NOT NULL,
    label character varying(255),
    usage_count bigint DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: interactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.interactions (
    id uuid DEFAULT uuidv7() NOT NULL,
    qr_code_id uuid,
    product_id uuid,
    tenant_id uuid NOT NULL,
    interaction_category character varying(50),
    interaction_subcategory character varying(50),
    interaction_status character varying(20),
    scanned_by uuid,
    ip_address inet,
    user_agent text,
    geolocation jsonb,
    additional_data jsonb,
    validation_template_id uuid,
    created_at timestamp with time zone
);

--
-- Name: inventory_movements; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inventory_movements (
    id uuid DEFAULT uuidv7() NOT NULL,
    location_id uuid NOT NULL,
    qr_code_id uuid NOT NULL,
    movement_type character varying(20),
    scanned_by uuid,
    scan_geolocation jsonb,
    scanned_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    type character varying(50) NOT NULL,
    title character varying(255) NOT NULL,
    body text,
    link text,
    data jsonb,
    read_at timestamp with time zone,
    created_at timestamp with time zone
);

--
-- Name: page_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.page_templates (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    template_type character varying(50) NOT NULL,
    template_name character varying(255) NOT NULL,
    html_content text NOT NULL,
    css_content text,
    js_content text,
    custom_fields jsonb,
    background_config jsonb,
    is_active boolean DEFAULT true,
    created_by uuid,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

--
-- Name: product_certifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_certifications (
    id uuid DEFAULT uuidv7() NOT NULL,
    product_id uuid NOT NULL,
    certification_type_id uuid NOT NULL,
    registration_number character varying(255) NOT NULL,
    sort_order bigint DEFAULT 0,
    created_at timestamp with time zone
);

--
-- Name: product_images; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_images (
    id uuid DEFAULT uuidv7() NOT NULL,
    product_id uuid NOT NULL,
    image_url text NOT NULL,
    caption character varying(255),
    is_main boolean DEFAULT false,
    sort_order bigint DEFAULT 0,
    file_size bigint,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

--
-- Name: product_social_account_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_social_account_links (
    id uuid DEFAULT uuidv7() NOT NULL,
    product_id uuid NOT NULL,
    social_account_id uuid NOT NULL,
    sort_order bigint DEFAULT 0,
    created_at timestamp with time zone
);

--
-- Name: product_social_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_social_links (
    id uuid DEFAULT uuidv7() NOT NULL,
    product_id uuid NOT NULL,
    platform_id uuid NOT NULL,
    handle_or_url character varying(500) NOT NULL,
    created_at timestamp with time zone
);

--
-- Name: products; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.products (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    product_name character varying(255) NOT NULL,
    product_code character varying(100),
    description text,
    status character varying(20) DEFAULT 'active'::character varying,
    display_config jsonb DEFAULT '{"batch_code": false, "brand_name": true, "expiry_date": false, "product_code": false, "product_name": true, "production_date": false, "show_verification_count": true}'::jsonb,
    warranty_fields_config jsonb DEFAULT '{"custom_fields": [], "optional_fields": {"invoice_number": false, "purchase_receipt": false}}'::jsonb,
    landing_appearance_config jsonb DEFAULT '{"card_blur": 0, "preset_id": null, "card_opacity": 90, "overlay_color": "#000000", "background_type": "none", "overlay_opacity": 30, "custom_background_url": null}'::jsonb,
    template_overrides jsonb,
    warranty_template_overrides jsonb,
    default_validation_template_id uuid,
    default_warranty_template_id uuid,
    warranty_enabled boolean DEFAULT false,
    warranty_months bigint DEFAULT 12,
    max_warranty_registration_days bigint,
    website_url character varying(500),
    website_caption character varying(100),
    videos jsonb DEFAULT '[]'::jsonb,
    counterfeit_scan_max bigint,
    loyalty_points_override bigint,
    created_by uuid,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: provinces; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.provinces (
    id bigint NOT NULL,
    country_code character varying(2) NOT NULL,
    name character varying(100) NOT NULL,
    code character varying(10),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: provinces_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.provinces_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: provinces_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.provinces_id_seq OWNED BY public.provinces.id;

--
-- Name: qc_scans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.qc_scans (
    id uuid DEFAULT uuidv7() NOT NULL,
    location_id uuid,
    qr_code_id uuid NOT NULL,
    qc_status character varying(20),
    scanned_by uuid,
    scan_geolocation jsonb,
    is_correction boolean DEFAULT false,
    corrects_scan_id uuid,
    correction_reason text,
    scanned_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

--
-- Name: qr_batches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.qr_batches (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    product_id uuid NOT NULL,
    batch_name character varying(255) NOT NULL,
    batch_code character varying(100),
    qr_count bigint NOT NULL,
    status character varying(20) DEFAULT 'completed'::character varying,
    prefix character varying(50),
    suffix character varying(50),
    production_date date,
    expiry_date date,
    logo_url text,
    csv_file_url text,
    need_validation boolean DEFAULT false,
    validation_template_id uuid,
    warranty_template_id uuid,
    geofence_enabled boolean DEFAULT false,
    geofence_latitude numeric(10,7),
    geofence_longitude numeric(10,7),
    geofence_radius_km numeric(6,1),
    geofence_label character varying(255),
    counterfeit_scan_max bigint,
    created_by uuid,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: qr_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.qr_codes (
    id uuid DEFAULT uuidv7() NOT NULL,
    batch_id uuid NOT NULL,
    qr_uuid uuid DEFAULT uuidv7() NOT NULL,
    qr_code character varying(255) NOT NULL,
    qr_image_url text,
    status character varying(20) DEFAULT 'active'::character varying,
    counterfeit_status character varying(20) DEFAULT 'valid'::character varying,
    is_compressed boolean DEFAULT false,
    compressed_data bytea,
    compressed_at timestamp with time zone,
    loyalty_claimed_by uuid,
    loyalty_claimed_at timestamp with time zone,
    counterfeit_scan_max bigint,
    created_at timestamp with time zone
);

--
-- Name: qr_generation_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.qr_generation_queue (
    id uuid DEFAULT uuidv7() NOT NULL,
    batch_id uuid NOT NULL,
    total_qr_count bigint NOT NULL,
    generated_count bigint DEFAULT 0,
    status character varying(20) DEFAULT 'queued'::character varying,
    worker_id character varying(100),
    error_message text,
    created_at timestamp with time zone,
    started_at timestamp with time zone,
    completed_at timestamp with time zone
);

--
-- Name: social_media_platforms; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.social_media_platforms (
    id uuid DEFAULT uuidv7() NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    icon character varying(50),
    base_url text,
    deep_link_pattern text,
    placeholder_text character varying(255),
    validation_type character varying(20) DEFAULT 'text'::character varying,
    is_active boolean DEFAULT true,
    display_order bigint DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: tenant_locations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_locations (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    location_name character varying(255) NOT NULL,
    location_type character varying(50) DEFAULT 'warehouse'::character varying NOT NULL,
    address text,
    city character varying(100),
    province character varying(100),
    postal_code character varying(10),
    phone_number character varying(20),
    geolocation jsonb,
    allowed_radius integer,
    status character varying(20) DEFAULT 'active'::character varying,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: tenant_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_settings (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    setting_key character varying(100) NOT NULL,
    setting_value jsonb NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

--
-- Name: tenant_social_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_social_accounts (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    platform_id uuid NOT NULL,
    account_handle character varying(255) NOT NULL,
    account_url character varying(500),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

--
-- Name: tenant_staff; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_staff (
    id uuid DEFAULT uuidv7() NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL,
    full_name character varying(255) NOT NULL,
    phone_number character varying(20),
    address text,
    "position" character varying(100),
    role character varying(50) NOT NULL,
    is_primary_admin boolean DEFAULT false,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

--
-- Name: tenants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenants (
    id uuid DEFAULT uuidv7() NOT NULL,
    company_name character varying(255) NOT NULL,
    company_address text,
    country character varying(100),
    province character varying(100),
    city character varying(100),
    country_code character varying(2),
    province_id bigint,
    city_id bigint,
    postal_code character varying(10),
    business_field character varying(100),
    phone_number character varying(20),
    company_email character varying(255),
    default_validation_template_id uuid,
    default_warranty_template_id uuid,
    slug character varying(100) NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: theme_presets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.theme_presets (
    id uuid DEFAULT uuidv7() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    preset_type character varying(20) NOT NULL,
    background_url text NOT NULL,
    thumbnail_url text,
    overlay_color character varying(7) DEFAULT '#000000'::character varying,
    overlay_opacity bigint DEFAULT 30,
    card_opacity bigint DEFAULT 90,
    card_blur bigint DEFAULT 0,
    is_active boolean DEFAULT true,
    display_order bigint DEFAULT 0,
    created_by uuid,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT uuidv7() NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    user_type character varying(50) NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying,
    must_change_password boolean DEFAULT false,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: warranty_activations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.warranty_activations (
    id uuid DEFAULT uuidv7() NOT NULL,
    qr_code_id uuid NOT NULL,
    customer_name character varying(255),
    customer_email character varying(255),
    customer_phone character varying(20),
    purchase_date date,
    purchase_store character varying(255),
    address text,
    country_code character varying(2),
    province_id bigint,
    city_id bigint,
    activation_data jsonb,
    activated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    warranty_expiry_date date,
    ip_address inet,
    geolocation jsonb,
    expiry_reminder_sent_at timestamp with time zone,
    duplicate_attempt_count bigint DEFAULT 0 NOT NULL,
    last_duplicate_attempt_at timestamp with time zone
);

--
-- Name: cities id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cities ALTER COLUMN id SET DEFAULT nextval('public.cities_id_seq'::regclass);

--
-- Name: provinces id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provinces ALTER COLUMN id SET DEFAULT nextval('public.provinces_id_seq'::regclass);

--
-- Name: activity_logs activity_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity_logs
    ADD CONSTRAINT activity_logs_pkey PRIMARY KEY (id);

--
-- Name: app_settings app_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_settings
    ADD CONSTRAINT app_settings_pkey PRIMARY KEY (id);

--
-- Name: certification_types certification_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.certification_types
    ADD CONSTRAINT certification_types_pkey PRIMARY KEY (id);

--
-- Name: cities cities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cities
    ADD CONSTRAINT cities_pkey PRIMARY KEY (id);

--
-- Name: counterfeit_detections counterfeit_detections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counterfeit_detections
    ADD CONSTRAINT counterfeit_detections_pkey PRIMARY KEY (id);

--
-- Name: counterfeit_reports counterfeit_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counterfeit_reports
    ADD CONSTRAINT counterfeit_reports_pkey PRIMARY KEY (id);

--
-- Name: countries countries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.countries
    ADD CONSTRAINT countries_pkey PRIMARY KEY (code);

--
-- Name: geofence_violations geofence_violations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geofence_violations
    ADD CONSTRAINT geofence_violations_pkey PRIMARY KEY (id);

--
-- Name: geofence_zone_templates geofence_zone_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geofence_zone_templates
    ADD CONSTRAINT geofence_zone_templates_pkey PRIMARY KEY (id);

--
-- Name: interactions interactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interactions
    ADD CONSTRAINT interactions_pkey PRIMARY KEY (id);

--
-- Name: inventory_movements inventory_movements_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_movements
    ADD CONSTRAINT inventory_movements_pkey PRIMARY KEY (id);

--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);

--
-- Name: page_templates page_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_templates
    ADD CONSTRAINT page_templates_pkey PRIMARY KEY (id);

--
-- Name: product_certifications product_certifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_certifications
    ADD CONSTRAINT product_certifications_pkey PRIMARY KEY (id);

--
-- Name: product_images product_images_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_images
    ADD CONSTRAINT product_images_pkey PRIMARY KEY (id);

--
-- Name: product_social_account_links product_social_account_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_social_account_links
    ADD CONSTRAINT product_social_account_links_pkey PRIMARY KEY (id);

--
-- Name: product_social_links product_social_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_social_links
    ADD CONSTRAINT product_social_links_pkey PRIMARY KEY (id);

--
-- Name: products products_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_pkey PRIMARY KEY (id);

--
-- Name: provinces provinces_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provinces
    ADD CONSTRAINT provinces_pkey PRIMARY KEY (id);

--
-- Name: qc_scans qc_scans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qc_scans
    ADD CONSTRAINT qc_scans_pkey PRIMARY KEY (id);

--
-- Name: qr_batches qr_batches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_batches
    ADD CONSTRAINT qr_batches_pkey PRIMARY KEY (id);

--
-- Name: qr_codes qr_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_codes
    ADD CONSTRAINT qr_codes_pkey PRIMARY KEY (id);

--
-- Name: qr_generation_queue qr_generation_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_generation_queue
    ADD CONSTRAINT qr_generation_queue_pkey PRIMARY KEY (id);

--
-- Name: social_media_platforms social_media_platforms_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.social_media_platforms
    ADD CONSTRAINT social_media_platforms_pkey PRIMARY KEY (id);

--
-- Name: tenant_locations tenant_locations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_locations
    ADD CONSTRAINT tenant_locations_pkey PRIMARY KEY (id);

--
-- Name: tenant_settings tenant_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_settings
    ADD CONSTRAINT tenant_settings_pkey PRIMARY KEY (id);

--
-- Name: tenant_social_accounts tenant_social_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_social_accounts
    ADD CONSTRAINT tenant_social_accounts_pkey PRIMARY KEY (id);

--
-- Name: tenant_staff tenant_staff_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_staff
    ADD CONSTRAINT tenant_staff_pkey PRIMARY KEY (id);

--
-- Name: tenants tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);

--
-- Name: theme_presets theme_presets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.theme_presets
    ADD CONSTRAINT theme_presets_pkey PRIMARY KEY (id);

--
-- Name: certification_types uni_certification_types_code; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.certification_types
    ADD CONSTRAINT uni_certification_types_code UNIQUE (code);

--
-- Name: social_media_platforms uni_social_media_platforms_code; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.social_media_platforms
    ADD CONSTRAINT uni_social_media_platforms_code UNIQUE (code);

--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

--
-- Name: warranty_activations warranty_activations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.warranty_activations
    ADD CONSTRAINT warranty_activations_pkey PRIMARY KEY (id);

--
-- Name: idx_app_settings_setting_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_app_settings_setting_key ON public.app_settings USING btree (setting_key);

--
-- Name: idx_certification_types_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_certification_types_deleted_at ON public.certification_types USING btree (deleted_at);

--
-- Name: idx_cities_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cities_deleted_at ON public.cities USING btree (deleted_at);

--
-- Name: idx_countries_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_countries_deleted_at ON public.countries USING btree (deleted_at);

--
-- Name: idx_geofence_zone_templates_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_geofence_zone_templates_deleted_at ON public.geofence_zone_templates USING btree (deleted_at);

--
-- Name: idx_interactions_product_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_interactions_product_id ON public.interactions USING btree (product_id);

--
-- Name: idx_notifications_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_tenant_id ON public.notifications USING btree (tenant_id);

--
-- Name: idx_products_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_products_deleted_at ON public.products USING btree (deleted_at);

--
-- Name: idx_provinces_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_provinces_deleted_at ON public.provinces USING btree (deleted_at);

--
-- Name: idx_qr_batches_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_qr_batches_deleted_at ON public.qr_batches USING btree (deleted_at);

--
-- Name: idx_qr_codes_qr_code; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_qr_codes_qr_code ON public.qr_codes USING btree (qr_code);

--
-- Name: idx_qr_codes_qr_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_qr_codes_qr_uuid ON public.qr_codes USING btree (qr_uuid);

--
-- Name: idx_qr_generation_queue_batch_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_qr_generation_queue_batch_id ON public.qr_generation_queue USING btree (batch_id);

--
-- Name: idx_social_media_platforms_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_social_media_platforms_deleted_at ON public.social_media_platforms USING btree (deleted_at);

--
-- Name: idx_tenant_locations_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenant_locations_deleted_at ON public.tenant_locations USING btree (deleted_at);

--
-- Name: idx_tenants_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_deleted_at ON public.tenants USING btree (deleted_at);

--
-- Name: idx_tenants_slug; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_tenants_slug ON public.tenants USING btree (slug);

--
-- Name: idx_theme_presets_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_theme_presets_deleted_at ON public.theme_presets USING btree (deleted_at);

--
-- Name: idx_users_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);

--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);

--
-- Name: activity_logs fk_activity_logs_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity_logs
    ADD CONSTRAINT fk_activity_logs_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: activity_logs fk_activity_logs_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity_logs
    ADD CONSTRAINT fk_activity_logs_user FOREIGN KEY (user_id) REFERENCES public.users(id);

--
-- Name: certification_types fk_certification_types_country; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.certification_types
    ADD CONSTRAINT fk_certification_types_country FOREIGN KEY (country_code) REFERENCES public.countries(code);

--
-- Name: cities fk_cities_country; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cities
    ADD CONSTRAINT fk_cities_country FOREIGN KEY (country_code) REFERENCES public.countries(code);

--
-- Name: cities fk_cities_province; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cities
    ADD CONSTRAINT fk_cities_province FOREIGN KEY (province_id) REFERENCES public.provinces(id);

--
-- Name: counterfeit_detections fk_counterfeit_detections_qr_code; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counterfeit_detections
    ADD CONSTRAINT fk_counterfeit_detections_qr_code FOREIGN KEY (qr_code_id) REFERENCES public.qr_codes(id);

--
-- Name: counterfeit_detections fk_counterfeit_detections_resolved_by_staff; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counterfeit_detections
    ADD CONSTRAINT fk_counterfeit_detections_resolved_by_staff FOREIGN KEY (resolved_by) REFERENCES public.tenant_staff(id);

--
-- Name: counterfeit_detections fk_counterfeit_detections_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counterfeit_detections
    ADD CONSTRAINT fk_counterfeit_detections_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: counterfeit_reports fk_counterfeit_reports_counterfeit_detection; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counterfeit_reports
    ADD CONSTRAINT fk_counterfeit_reports_counterfeit_detection FOREIGN KEY (counterfeit_detection_id) REFERENCES public.counterfeit_detections(id);

--
-- Name: counterfeit_reports fk_counterfeit_reports_qr_code; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counterfeit_reports
    ADD CONSTRAINT fk_counterfeit_reports_qr_code FOREIGN KEY (qr_code_id) REFERENCES public.qr_codes(id);

--
-- Name: counterfeit_reports fk_counterfeit_reports_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counterfeit_reports
    ADD CONSTRAINT fk_counterfeit_reports_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: geofence_violations fk_geofence_violations_batch; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geofence_violations
    ADD CONSTRAINT fk_geofence_violations_batch FOREIGN KEY (batch_id) REFERENCES public.qr_batches(id);

--
-- Name: geofence_violations fk_geofence_violations_interaction; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geofence_violations
    ADD CONSTRAINT fk_geofence_violations_interaction FOREIGN KEY (interaction_id) REFERENCES public.interactions(id);

--
-- Name: geofence_violations fk_geofence_violations_product; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geofence_violations
    ADD CONSTRAINT fk_geofence_violations_product FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: geofence_violations fk_geofence_violations_qr_code; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geofence_violations
    ADD CONSTRAINT fk_geofence_violations_qr_code FOREIGN KEY (qr_code_id) REFERENCES public.qr_codes(id);

--
-- Name: geofence_violations fk_geofence_violations_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geofence_violations
    ADD CONSTRAINT fk_geofence_violations_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: geofence_zone_templates fk_geofence_zone_templates_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.geofence_zone_templates
    ADD CONSTRAINT fk_geofence_zone_templates_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: interactions fk_interactions_product; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interactions
    ADD CONSTRAINT fk_interactions_product FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: interactions fk_interactions_qr_code; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interactions
    ADD CONSTRAINT fk_interactions_qr_code FOREIGN KEY (qr_code_id) REFERENCES public.qr_codes(id);

--
-- Name: interactions fk_interactions_scanned_by_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interactions
    ADD CONSTRAINT fk_interactions_scanned_by_user FOREIGN KEY (scanned_by) REFERENCES public.users(id);

--
-- Name: interactions fk_interactions_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interactions
    ADD CONSTRAINT fk_interactions_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: interactions fk_interactions_validation_template; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interactions
    ADD CONSTRAINT fk_interactions_validation_template FOREIGN KEY (validation_template_id) REFERENCES public.page_templates(id);

--
-- Name: inventory_movements fk_inventory_movements_location; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_movements
    ADD CONSTRAINT fk_inventory_movements_location FOREIGN KEY (location_id) REFERENCES public.tenant_locations(id);

--
-- Name: inventory_movements fk_inventory_movements_qr_code; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_movements
    ADD CONSTRAINT fk_inventory_movements_qr_code FOREIGN KEY (qr_code_id) REFERENCES public.qr_codes(id);

--
-- Name: inventory_movements fk_inventory_movements_scanned_by_staff; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_movements
    ADD CONSTRAINT fk_inventory_movements_scanned_by_staff FOREIGN KEY (scanned_by) REFERENCES public.tenant_staff(id);

--
-- Name: page_templates fk_page_templates_created_by_staff; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_templates
    ADD CONSTRAINT fk_page_templates_created_by_staff FOREIGN KEY (created_by) REFERENCES public.tenant_staff(id);

--
-- Name: page_templates fk_page_templates_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_templates
    ADD CONSTRAINT fk_page_templates_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: product_certifications fk_product_certifications_certification_type; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_certifications
    ADD CONSTRAINT fk_product_certifications_certification_type FOREIGN KEY (certification_type_id) REFERENCES public.certification_types(id);

--
-- Name: product_certifications fk_product_certifications_product; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_certifications
    ADD CONSTRAINT fk_product_certifications_product FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: product_social_account_links fk_product_social_account_links_social_account; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_social_account_links
    ADD CONSTRAINT fk_product_social_account_links_social_account FOREIGN KEY (social_account_id) REFERENCES public.tenant_social_accounts(id);

--
-- Name: product_social_links fk_product_social_links_platform; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_social_links
    ADD CONSTRAINT fk_product_social_links_platform FOREIGN KEY (platform_id) REFERENCES public.social_media_platforms(id);

--
-- Name: product_social_links fk_product_social_links_product; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_social_links
    ADD CONSTRAINT fk_product_social_links_product FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: products fk_products_created_by_staff; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT fk_products_created_by_staff FOREIGN KEY (created_by) REFERENCES public.tenant_staff(id);

--
-- Name: products fk_products_default_validation_template; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT fk_products_default_validation_template FOREIGN KEY (default_validation_template_id) REFERENCES public.page_templates(id);

--
-- Name: products fk_products_default_warranty_template; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT fk_products_default_warranty_template FOREIGN KEY (default_warranty_template_id) REFERENCES public.page_templates(id);

--
-- Name: product_images fk_products_images; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_images
    ADD CONSTRAINT fk_products_images FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: qr_batches fk_products_qr_batches; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_batches
    ADD CONSTRAINT fk_products_qr_batches FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: product_social_account_links fk_products_social_account_links; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_social_account_links
    ADD CONSTRAINT fk_products_social_account_links FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: provinces fk_provinces_country; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provinces
    ADD CONSTRAINT fk_provinces_country FOREIGN KEY (country_code) REFERENCES public.countries(code);

--
-- Name: qc_scans fk_qc_scans_corrects_scan; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qc_scans
    ADD CONSTRAINT fk_qc_scans_corrects_scan FOREIGN KEY (corrects_scan_id) REFERENCES public.qc_scans(id);

--
-- Name: qc_scans fk_qc_scans_location; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qc_scans
    ADD CONSTRAINT fk_qc_scans_location FOREIGN KEY (location_id) REFERENCES public.tenant_locations(id);

--
-- Name: qc_scans fk_qc_scans_qr_code; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qc_scans
    ADD CONSTRAINT fk_qc_scans_qr_code FOREIGN KEY (qr_code_id) REFERENCES public.qr_codes(id);

--
-- Name: qc_scans fk_qc_scans_scanned_by_staff; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qc_scans
    ADD CONSTRAINT fk_qc_scans_scanned_by_staff FOREIGN KEY (scanned_by) REFERENCES public.tenant_staff(id);

--
-- Name: qr_batches fk_qr_batches_created_by_staff; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_batches
    ADD CONSTRAINT fk_qr_batches_created_by_staff FOREIGN KEY (created_by) REFERENCES public.tenant_staff(id);

--
-- Name: qr_codes fk_qr_batches_qr_codes; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_codes
    ADD CONSTRAINT fk_qr_batches_qr_codes FOREIGN KEY (batch_id) REFERENCES public.qr_batches(id);

--
-- Name: qr_batches fk_qr_batches_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_batches
    ADD CONSTRAINT fk_qr_batches_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: qr_batches fk_qr_batches_validation_template; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_batches
    ADD CONSTRAINT fk_qr_batches_validation_template FOREIGN KEY (validation_template_id) REFERENCES public.page_templates(id);

--
-- Name: qr_batches fk_qr_batches_warranty_template; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_batches
    ADD CONSTRAINT fk_qr_batches_warranty_template FOREIGN KEY (warranty_template_id) REFERENCES public.page_templates(id);

--
-- Name: qr_generation_queue fk_qr_generation_queue_batch; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.qr_generation_queue
    ADD CONSTRAINT fk_qr_generation_queue_batch FOREIGN KEY (batch_id) REFERENCES public.qr_batches(id);

--
-- Name: tenant_settings fk_tenant_settings_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_settings
    ADD CONSTRAINT fk_tenant_settings_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: tenant_social_accounts fk_tenant_social_accounts_platform; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_social_accounts
    ADD CONSTRAINT fk_tenant_social_accounts_platform FOREIGN KEY (platform_id) REFERENCES public.social_media_platforms(id);

--
-- Name: tenant_social_accounts fk_tenant_social_accounts_tenant; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_social_accounts
    ADD CONSTRAINT fk_tenant_social_accounts_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: tenants fk_tenants_city_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT fk_tenants_city_ref FOREIGN KEY (city_id) REFERENCES public.cities(id);

--
-- Name: tenants fk_tenants_country_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT fk_tenants_country_ref FOREIGN KEY (country_code) REFERENCES public.countries(code);

--
-- Name: tenants fk_tenants_default_validation_template; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT fk_tenants_default_validation_template FOREIGN KEY (default_validation_template_id) REFERENCES public.page_templates(id);

--
-- Name: tenants fk_tenants_default_warranty_template; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT fk_tenants_default_warranty_template FOREIGN KEY (default_warranty_template_id) REFERENCES public.page_templates(id);

--
-- Name: tenant_locations fk_tenants_locations; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_locations
    ADD CONSTRAINT fk_tenants_locations FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: products fk_tenants_products; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT fk_tenants_products FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: tenants fk_tenants_province_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT fk_tenants_province_ref FOREIGN KEY (province_id) REFERENCES public.provinces(id);

--
-- Name: tenant_staff fk_tenants_staff; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_staff
    ADD CONSTRAINT fk_tenants_staff FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

--
-- Name: tenant_staff fk_users_tenant_staff; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_staff
    ADD CONSTRAINT fk_users_tenant_staff FOREIGN KEY (user_id) REFERENCES public.users(id);

--
-- Name: warranty_activations fk_warranty_activations_city; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.warranty_activations
    ADD CONSTRAINT fk_warranty_activations_city FOREIGN KEY (city_id) REFERENCES public.cities(id);

--
-- Name: warranty_activations fk_warranty_activations_country; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.warranty_activations
    ADD CONSTRAINT fk_warranty_activations_country FOREIGN KEY (country_code) REFERENCES public.countries(code);

--
-- Name: warranty_activations fk_warranty_activations_province; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.warranty_activations
    ADD CONSTRAINT fk_warranty_activations_province FOREIGN KEY (province_id) REFERENCES public.provinces(id);

--
-- Name: warranty_activations fk_warranty_activations_qr_code; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.warranty_activations
    ADD CONSTRAINT fk_warranty_activations_qr_code FOREIGN KEY (qr_code_id) REFERENCES public.qr_codes(id);

--
-- PostgreSQL database dump complete
--


-- Composite uniqueness the application relies on (ON CONFLICT upserts)
ALTER TABLE public.tenant_settings
    ADD CONSTRAINT uniq_tenant_settings_tenant_key UNIQUE (tenant_id, setting_key);

-- One warranty activation per QR code (duplicate attempts are tracked, not inserted)
CREATE UNIQUE INDEX IF NOT EXISTS uniq_warranty_activations_qr_code
    ON public.warranty_activations (qr_code_id);

-- Active batch names are unique per product (soft-deleted rows excluded)
CREATE UNIQUE INDEX IF NOT EXISTS uniq_qr_batch_name_active
    ON public.qr_batches (product_id, batch_name) WHERE deleted_at IS NULL;

-- +goose Down
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
