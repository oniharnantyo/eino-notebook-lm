-- Delete initial users by email
DELETE FROM users WHERE email IN (
    'admin@einonotebook.com',
    'demo@einonotebook.com',
    'test@einonotebook.com'
);
