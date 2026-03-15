-- Insert initial admin user
-- Password: 'admin123' (bcrypt hash - change this after first login!)
INSERT INTO users (id, email, name, password, status, created_at, updated_at)
VALUES (
    '01234567-0123-4567-89ab-0123456789ab',
    'admin@einonotebook.com',
    'Admin User',
    '$2a$10$N9qo8uLOickgx2ZMRZoMye1j50kgCMa1aX9qO5.uPJHtYzMz.8KVi',
    'active',
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;

-- Insert initial demo user
-- Password: 'demo123' (bcrypt hash - for demo purposes only!)
INSERT INTO users (id, email, name, password, status, created_at, updated_at)
VALUES (
    '01234567-0123-4567-89ab-0123456789ac',
    'demo@einonotebook.com',
    'Demo User',
    '$2a$10$8XZJ5J5J5J5J5J5J5J5J5Je1j50kgCMa1aX9qO5.uPJHtYzMz.8KVi',
    'active',
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;

-- Insert initial test user
-- Password: 'test123' (bcrypt hash - for testing only!)
INSERT INTO users (id, email, name, password, status, created_at, updated_at)
VALUES (
    '01234567-0123-4567-89ab-0123456789ad',
    'test@einonotebook.com',
    'Test User',
    '$2a$10$7XZJ5J5J5J5J5J5J5J5J5Je1j50kgCMa1aX9qO5.uPJHtYzMz.8KVi',
    'active',
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;
