-- Down for 000012: remove fees:manage grant (keep permission row removal safe).
DELETE FROM role_permissions rp USING permissions p, roles r
WHERE rp.permission_id = p.id AND rp.role_id = r.id
  AND p.resource = 'fees' AND p.action = 'manage' AND r.slug = 'super_admin';
DELETE FROM permissions WHERE resource = 'fees' AND action = 'manage';
