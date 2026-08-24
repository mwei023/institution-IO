DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'classes' AND action = 'write'
);

DELETE FROM permissions WHERE resource = 'classes' AND action = 'write';
