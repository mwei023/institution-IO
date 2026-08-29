-- Adds a fees:manage permission for fee structure and invoice-generation
-- operations (create fee structure, add components, generate invoices for
-- a class) — distinct from fees:receipt, which only covers recording
-- payments against invoices that already exist. Granted to super_admin
-- only, matching migration 011's pattern for classes:write.
--
-- Granted to every super_admin role row (one per institution), not a
-- hardcoded role_id, for the same reason migration 011 did this: a
-- literal role_id would silently miss every institution but the one it
-- was copied from.

INSERT INTO permissions (id, resource, action, description)
VALUES (gen_random_uuid(), 'fees', 'manage', 'Create fee structures, add fee components, and generate invoices')
ON CONFLICT (resource, action) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.slug = 'super_admin'
  AND p.resource = 'fees'
  AND p.action = 'manage'
ON CONFLICT (role_id, permission_id) DO NOTHING;