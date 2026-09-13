CREATE TABLE user_substitutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    principal_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    substitute_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    starts_at DATE,
    ends_at DATE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CHECK (principal_user_id <> substitute_user_id),
    CHECK (starts_at IS NULL OR ends_at IS NULL OR starts_at <= ends_at),
    UNIQUE (principal_user_id)
);

CREATE INDEX idx_user_substitutions_principal ON user_substitutions (principal_user_id);
CREATE INDEX idx_user_substitutions_substitute ON user_substitutions (substitute_user_id);
CREATE INDEX idx_user_substitutions_active ON user_substitutions (substitute_user_id, is_active, starts_at, ends_at);
