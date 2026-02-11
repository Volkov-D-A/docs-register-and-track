-- Пользователи
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    login VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Роли пользователей
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (
        role IN ('admin', 'clerk', 'executor')
    ),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, role)
);

-- Индексы
CREATE INDEX idx_users_login ON users (login);

CREATE INDEX idx_user_roles_user_id ON user_roles (user_id);

-- Начальный администратор (пароль: admin123)
INSERT INTO
    users (
        login,
        password_hash,
        full_name
    )
VALUES (
        'admin',
        '$2a$10$QWk7gI0Jh.F7CrdXImFzI.meruiFwSWHNBpzhKCAVex4QgAPaFCm6',
        'Администратор'
    );

INSERT INTO
    user_roles (user_id, role)
SELECT id, 'admin'
FROM users
WHERE
    login = 'admin';