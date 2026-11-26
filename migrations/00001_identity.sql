-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL CONSTRAINT users_email_unique UNIQUE,
    password_hash text,
    email_verified_at timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'active' CONSTRAINT users_status_check CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_lower_check CHECK (email = lower(email))
);

CREATE TABLE user_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    provider text NOT NULL CONSTRAINT user_identities_provider_check CHECK (provider = 'google'),
    provider_subject text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_identities_provider_subject_unique UNIQUE (provider, provider_subject),
    CONSTRAINT user_identities_user_provider_unique UNIQUE (user_id, provider)
);

CREATE TABLE auth_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id),
    email text NOT NULL,
    purpose text NOT NULL CONSTRAINT auth_challenges_purpose_check CHECK (purpose IN ('register', 'login', 'google', 'reset')),
    pending_token_hash bytea NOT NULL CONSTRAINT auth_challenges_token_unique UNIQUE,
    pending_password_hash text,
    provider_subject text,
    otp_digest bytea,
    attempts integer NOT NULL DEFAULT 0 CONSTRAINT auth_challenges_attempts_check CHECK (attempts >= 0),
    resend_count integer NOT NULL DEFAULT 0 CONSTRAINT auth_challenges_resend_check CHECK (resend_count >= 0),
    sent_at timestamptz,
    expires_at timestamptz NOT NULL,
    family_expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    invalidated_at timestamptz,
    CONSTRAINT auth_challenges_token_length_check CHECK (octet_length(pending_token_hash) = 32),
    CONSTRAINT auth_challenges_digest_length_check CHECK (otp_digest IS NULL OR octet_length(otp_digest) = 32)
);
CREATE INDEX auth_challenges_expires_idx ON auth_challenges(expires_at);
CREATE INDEX auth_challenges_email_purpose_idx ON auth_challenges(email, purpose);

CREATE TABLE oauth_states (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    state_hash bytea NOT NULL CONSTRAINT oauth_states_hash_unique UNIQUE,
    browser_token_hash bytea NOT NULL,
    nonce text NOT NULL,
    pkce_verifier text NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oauth_states_hash_length_check CHECK (octet_length(state_hash) = 32),
    CONSTRAINT oauth_states_browser_length_check CHECK (octet_length(browser_token_hash) = 32)
);
CREATE INDEX oauth_states_expires_idx ON oauth_states(expires_at);

CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    token_hash bytea NOT NULL CONSTRAINT sessions_token_unique UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CONSTRAINT sessions_token_length_check CHECK (octet_length(token_hash) = 32)
);
CREATE INDEX sessions_user_idx ON sessions(user_id);
CREATE INDEX sessions_expires_idx ON sessions(expires_at);

CREATE TABLE email_blocklist (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind text NOT NULL CONSTRAINT email_blocklist_kind_check CHECK (kind IN ('email', 'domain')),
    value text NOT NULL,
    reason text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT email_blocklist_kind_value_unique UNIQUE (kind, value),
    CONSTRAINT email_blocklist_lower_check CHECK (value = lower(value))
);

CREATE TABLE disposable_email_domains (
    domain text PRIMARY KEY,
    source_revision text NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT disposable_domain_lower_check CHECK (domain = lower(domain))
);

CREATE TABLE auth_rate_limits (
    scope text NOT NULL,
    key_hash bytea NOT NULL,
    window_start timestamptz NOT NULL,
    count integer NOT NULL DEFAULT 0 CONSTRAINT auth_rate_count_check CHECK (count >= 0),
    PRIMARY KEY (scope, key_hash, window_start),
    CONSTRAINT auth_rate_key_length_check CHECK (octet_length(key_hash) = 32)
);
CREATE INDEX auth_rate_window_idx ON auth_rate_limits(window_start);

CREATE TABLE auth_audit_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id),
    event text NOT NULL,
    outcome text NOT NULL,
    request_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT auth_audit_details_object_check CHECK (jsonb_typeof(details) = 'object')
);
CREATE INDEX auth_audit_created_idx ON auth_audit_logs(created_at);
