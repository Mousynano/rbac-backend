-- +goose Up
CREATE TABLE diagrams (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id),
    name text NOT NULL,
    blueprint jsonb NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT diagrams_name_length_check CHECK (char_length(name) BETWEEN 1 AND 120),
    CONSTRAINT diagrams_blueprint_object_check CHECK (jsonb_typeof(blueprint) = 'object'),
    CONSTRAINT diagrams_version_check CHECK (version >= 1)
);
CREATE INDEX diagrams_owner_updated_idx ON diagrams(owner_id, updated_at DESC, id DESC);
