-- Adds a proper classes:write permission for class-management operations
-- (create class, assign/reassign class teacher) — previously these
-- operations didn't exist at all; nothing was ever reusing attendance:write
-- for them. Granted to super_admin only, matching the existing admin-tier
-- pattern used by grades:publish and attendance:approve (teacher does NOT
-- get this — teachers don't create classes or assign class teachers).
--
-- Granted to every super_admin role row, not a hardcoded role_id — this
-- institution has more than one super_admin role (one per institution,
-- copied with new IDs the way Complete Academy's roles were seeded in
-- Session 5), so a single INSERT with a literal role_id would silently
-- miss every institution but the one it was copied from.

INSERT INTO permissions (id, resource, action, description)
VALUES (gen_random_uuid(), 'classes', 'write', 'Create classes and assign/reassign class teachers')
ON CONFLICT (resource, action) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.slug = 'super_admin'
  AND p.resource = 'classes'
  AND p.action = 'write'
ON CONFLICT (role_id, permission_id) DO NOTHING;