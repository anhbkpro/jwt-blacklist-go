CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on username for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Insert default users (admin and user)
INSERT INTO users (username, email, password, role)
VALUES
    ('admin', 'admin@example.com', '$argon2id$v=19$m=65536,t=3,p=2$mwTVNvIy4EBaphLMv6Iozg$HrAc8MQ/g1HX6eryWcFc75h7vknOqADznwS6zA04REw', 'admin'),
    ('user', 'user@example.com', '$argon2id$v=19$m=65536,t=3,p=2$P00D1MRXhY+tSrYMCDe0rg$JkUThHcvsIxD1RW+5zGqCfvzbtK2+RQ5iV6jyH/OcjI', 'user')
ON CONFLICT (username) DO NOTHING;
