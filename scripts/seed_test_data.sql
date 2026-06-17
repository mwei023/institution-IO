-- =============================================================
-- Institution OS — Seed Data for Upendo Secondary School
-- Institution ID: 11111111-1111-1111-1111-111111111111
-- Run after all migrations: psql $DATABASE_URL -f seed_test_data.sql
-- =============================================================

BEGIN;

-- ================================================================
-- 1. INSTITUTION
-- ================================================================
INSERT INTO institutions (id, name, short_name, type, registration_number,
    address, county, sub_county, phone, email, school_slug, is_active)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Upendo Secondary School',
    'Upendo',
    'secondary',
    'TSC/SEC/2005/0042',
    'Ngong Road, Karen',
    'Nairobi',
    'Langata',
    '+254722100200',
    'info@upendo.ac.ke',
    'upendo',
    true
)
ON CONFLICT (id) DO NOTHING;

-- ================================================================
-- 2. ROLES (system roles for the institution)
-- ================================================================
INSERT INTO roles (id, institution_id, name, slug, is_system) VALUES
    ('r0000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'Super Admin',      'super_admin',      true),
    ('r0000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'Department Admin',  'dept_admin',       true),
    ('r0000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'Teacher',           'teacher',          true),
    ('r0000001-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'Student',           'student',          true),
    ('r0000001-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'Parent',            'parent',           true),
    ('r0000001-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'Accountant',        'accountant',       true),
    ('r0000001-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', 'Nurse',             'nurse',            true),
    ('r0000001-0000-0000-0000-000000000008', '11111111-1111-1111-1111-111111111111', 'Librarian',         'librarian',        true),
    ('r0000001-0000-0000-0000-000000000009', '11111111-1111-1111-1111-111111111111', 'Storekeeper',       'storekeeper',      true),
    ('r0000001-0000-0000-0000-000000000010', '11111111-1111-1111-1111-111111111111', 'Driver',            'driver',           true),
    ('r0000001-0000-0000-0000-000000000011', '11111111-1111-1111-1111-111111111111', 'Security Officer',  'security_officer', true),
    ('r0000001-0000-0000-0000-000000000012', '11111111-1111-1111-1111-111111111111', 'Board Member',      'board_member',     true)
ON CONFLICT (institution_id, slug) DO NOTHING;

-- ================================================================
-- 3. PERMISSIONS
-- ================================================================
INSERT INTO permissions (id, resource, action, description) VALUES
    -- Students
    (gen_random_uuid(), 'students',     'read',    'View student records'),
    (gen_random_uuid(), 'students',     'write',   'Create/edit student records'),
    (gen_random_uuid(), 'students',     'delete',  'Archive student records'),
    (gen_random_uuid(), 'students',     'export',  'Export student data'),
    -- Fees
    (gen_random_uuid(), 'fees',         'read',    'View fee accounts'),
    (gen_random_uuid(), 'fees',         'write',   'Manage fee structures'),
    (gen_random_uuid(), 'fees',         'receipt', 'Record fee payments'),
    (gen_random_uuid(), 'fees',         'waiver',  'Grant fee waivers'),
    -- Ledger
    (gen_random_uuid(), 'ledger',       'read',    'View general ledger'),
    (gen_random_uuid(), 'ledger',       'write',   'Post journal entries'),
    (gen_random_uuid(), 'ledger',       'approve', 'Approve journal entries'),
    -- Payroll
    (gen_random_uuid(), 'payroll',      'read',    'View payroll'),
    (gen_random_uuid(), 'payroll',      'write',   'Process payroll'),
    (gen_random_uuid(), 'payroll',      'approve', 'Approve payroll run'),
    -- Staff
    (gen_random_uuid(), 'staff',        'read',    'View staff records'),
    (gen_random_uuid(), 'staff',        'write',   'Create/edit staff records'),
    -- Attendance
    (gen_random_uuid(), 'attendance',   'read',    'View attendance records'),
    (gen_random_uuid(), 'attendance',   'write',   'Record attendance'),
    (gen_random_uuid(), 'attendance',   'approve', 'Approve attendance corrections'),
    -- Grades
    (gen_random_uuid(), 'grades',       'read',    'View grades'),
    (gen_random_uuid(), 'grades',       'write',   'Enter grades'),
    (gen_random_uuid(), 'grades',       'publish', 'Publish report cards'),
    -- Reports
    (gen_random_uuid(), 'reports',      'read',    'View reports'),
    (gen_random_uuid(), 'reports',      'export',  'Export reports'),
    -- Settings
    (gen_random_uuid(), 'settings',     'read',    'View system settings'),
    (gen_random_uuid(), 'settings',     'write',   'Modify system settings'),
    -- Users
    (gen_random_uuid(), 'users',        'read',    'View user accounts'),
    (gen_random_uuid(), 'users',        'write',   'Create/edit user accounts'),
    (gen_random_uuid(), 'users',        'delete',  'Deactivate user accounts'),
    -- Announcements
    (gen_random_uuid(), 'announcements','read',    'View announcements'),
    (gen_random_uuid(), 'announcements','write',   'Create announcements'),
    -- Documents
    (gen_random_uuid(), 'documents',    'read',    'View documents'),
    (gen_random_uuid(), 'documents',    'write',   'Upload documents'),
    -- Audit
    (gen_random_uuid(), 'audit_logs',   'read',    'View audit logs')
ON CONFLICT (resource, action) DO NOTHING;

-- ================================================================
-- 4. ROLE → PERMISSION ASSIGNMENTS
-- Super Admin gets everything
-- ================================================================
INSERT INTO role_permissions (role_id, permission_id)
SELECT 'r0000001-0000-0000-0000-000000000001', id FROM permissions
ON CONFLICT DO NOTHING;

-- Accountant gets fee + ledger + payroll + reports
INSERT INTO role_permissions (role_id, permission_id)
SELECT 'r0000001-0000-0000-0000-000000000006', id
FROM permissions
WHERE (resource IN ('fees','ledger','payroll','reports','students') AND action IN ('read','write','receipt','approve','export'))
   OR (resource = 'audit_logs' AND action = 'read')
ON CONFLICT DO NOTHING;

-- Teacher gets attendance (write), grades (write), students (read), announcements (read), documents (read/write)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 'r0000001-0000-0000-0000-000000000003', id
FROM permissions
WHERE (resource = 'attendance' AND action IN ('read','write'))
   OR (resource = 'grades' AND action IN ('read','write'))
   OR (resource = 'students' AND action = 'read')
   OR (resource = 'announcements' AND action IN ('read','write'))
   OR (resource = 'documents' AND action IN ('read','write'))
ON CONFLICT DO NOTHING;

-- Student gets read-only on their own data
INSERT INTO role_permissions (role_id, permission_id)
SELECT 'r0000001-0000-0000-0000-000000000004', id
FROM permissions
WHERE (resource IN ('grades','attendance','fees','announcements','documents') AND action = 'read')
ON CONFLICT DO NOTHING;

-- Parent gets fee read, announcements read, grades read
INSERT INTO role_permissions (role_id, permission_id)
SELECT 'r0000001-0000-0000-0000-000000000005', id
FROM permissions
WHERE (resource IN ('fees','grades','attendance','announcements','documents') AND action = 'read')
ON CONFLICT DO NOTHING;

-- ================================================================
-- 5. ACADEMIC YEAR & TERMS
-- ================================================================
INSERT INTO academic_years (id, institution_id, name, start_date, end_date, is_current) VALUES
    ('ay000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '2024', '2024-01-08', '2024-11-29', true)
ON CONFLICT (institution_id, name) DO NOTHING;

INSERT INTO terms (id, institution_id, academic_year_id, name, term_number, start_date, end_date, is_current) VALUES
    ('tm000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Term 1', 1, '2024-01-08', '2024-04-05', false),
    ('tm000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Term 2', 2, '2024-05-06', '2024-08-02', false),
    ('tm000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Term 3', 3, '2024-09-02', '2024-11-29', true)
ON CONFLICT (institution_id, academic_year_id, term_number) DO NOTHING;

-- ================================================================
-- 6. DEPARTMENTS
-- ================================================================
INSERT INTO departments (id, institution_id, name, code) VALUES
    ('dp000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'Sciences',    'SCI'),
    ('dp000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'Humanities',  'HUM'),
    ('dp000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'Mathematics', 'MAT'),
    ('dp000001-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'Languages',   'LAN'),
    ('dp000001-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'Technical',   'TEC'),
    ('dp000001-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'Finance',     'FIN'),
    ('dp000001-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', 'HR',          'HR')
ON CONFLICT (institution_id, code) DO NOTHING;

-- ================================================================
-- 7. CLASSES (3 streams per form, 2024)
-- ================================================================
INSERT INTO classes (id, institution_id, academic_year_id, name, stream, level) VALUES
    ('cl000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 1', 'East',  1),
    ('cl000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 1', 'West',  1),
    ('cl000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 1', 'North', 1),
    ('cl000001-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 2', 'East',  2),
    ('cl000001-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 2', 'West',  2),
    ('cl000001-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 2', 'North', 2),
    ('cl000001-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 3', 'East',  3),
    ('cl000001-0000-0000-0000-000000000008', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 3', 'West',  3),
    ('cl000001-0000-0000-0000-000000000009', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 3', 'North', 3),
    ('cl000001-0000-0000-0000-000000000010', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 4', 'East',  4),
    ('cl000001-0000-0000-0000-000000000011', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 4', 'West',  4),
    ('cl000001-0000-0000-0000-000000000012', '11111111-1111-1111-1111-111111111111', 'ay000001-0000-0000-0000-000000000001', 'Form 4', 'North', 4)
ON CONFLICT DO NOTHING;

-- ================================================================
-- 8. SUBJECTS
-- ================================================================
INSERT INTO subjects (id, institution_id, department_id, code, name, knec_code, is_compulsory, credit_hours_pw) VALUES
    ('su000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000004', 'ENG', 'English Language',      '101', true,  5),
    ('su000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000004', 'KIS', 'Kiswahili',             '102', true,  5),
    ('su000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000003', 'MAT', 'Mathematics',           '121', true,  6),
    ('su000001-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000001', 'BIO', 'Biology',               '231', false, 4),
    ('su000001-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000001', 'PHY', 'Physics',               '232', false, 4),
    ('su000001-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000001', 'CHE', 'Chemistry',             '233', false, 4),
    ('su000001-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000002', 'HIS', 'History & Government',  '311', false, 3),
    ('su000001-0000-0000-0000-000000000008', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000002', 'GEO', 'Geography',             '312', false, 3),
    ('su000001-0000-0000-0000-000000000009', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000002', 'CRE', 'C.R.E',                 '313', false, 3),
    ('su000001-0000-0000-0000-000000000010', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000005', 'BST', 'Business Studies',      '565', false, 3),
    ('su000001-0000-0000-0000-000000000011', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000005', 'AGR', 'Agriculture',           '443', false, 3),
    ('su000001-0000-0000-0000-000000000012', '11111111-1111-1111-1111-111111111111', 'dp000001-0000-0000-0000-000000000005', 'CMP', 'Computer Studies',      '451', false, 3)
ON CONFLICT (institution_id, code) DO NOTHING;

-- ================================================================
-- 9. STAFF USERS (Principal, Teachers, Accountant, Nurse)
-- All passwords = "Admin1234!" (bcrypt hash below)
-- ================================================================
-- bcrypt hash for "Admin1234!" with cost 12:
-- $2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.

INSERT INTO users (id, institution_id, role_id, email, phone, first_name, last_name,
    gender, password_hash, is_active, must_change_password) VALUES
    -- Principal (Super Admin)
    ('u0000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000001',
     'principal@upendo.ac.ke', '+254722100201', 'Margaret', 'Wanjiku',
     'F', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    -- Deputy Principal (Dept Admin)
    ('u0000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000002',
     'deputy@upendo.ac.ke', '+254722100202', 'Peter', 'Kamau',
     'M', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    -- Accountant
    ('u0000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000006',
     'finance@upendo.ac.ke', '+254722100203', 'Grace', 'Atieno',
     'F', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    -- Nurse
    ('u0000001-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000007',
     'nurse@upendo.ac.ke', '+254722100204', 'Susan', 'Muthoni',
     'F', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    -- Teacher 1 — English
    ('u0000001-0000-0000-0000-000000000010', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000003',
     'teacher.odhiambo@upendo.ac.ke', '+254722100210', 'James', 'Odhiambo',
     'M', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    -- Teacher 2 — Mathematics
    ('u0000001-0000-0000-0000-000000000011', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000003',
     'teacher.njoroge@upendo.ac.ke', '+254722100211', 'Alice', 'Njoroge',
     'F', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    -- Teacher 3 — Biology
    ('u0000001-0000-0000-0000-000000000012', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000003',
     'teacher.mugo@upendo.ac.ke', '+254722100212', 'David', 'Mugo',
     'M', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    -- Teacher 4 — Physics
    ('u0000001-0000-0000-0000-000000000013', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000003',
     'teacher.kariuki@upendo.ac.ke', '+254722100213', 'Samuel', 'Kariuki',
     'M', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    -- Teacher 5 — Chemistry & Class Teacher Form 1 East
    ('u0000001-0000-0000-0000-000000000014', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000003',
     'teacher.waweru@upendo.ac.ke', '+254722100214', 'Anne', 'Waweru',
     'F', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false)
ON CONFLICT (institution_id, email) DO NOTHING;

-- Assign class teachers
UPDATE classes SET class_teacher_id = 'u0000001-0000-0000-0000-000000000014'
WHERE id = 'cl000001-0000-0000-0000-000000000001';
UPDATE classes SET class_teacher_id = 'u0000001-0000-0000-0000-000000000010'
WHERE id = 'cl000001-0000-0000-0000-000000000004';
UPDATE classes SET class_teacher_id = 'u0000001-0000-0000-0000-000000000011'
WHERE id = 'cl000001-0000-0000-0000-000000000007';
UPDATE classes SET class_teacher_id = 'u0000001-0000-0000-0000-000000000012'
WHERE id = 'cl000001-0000-0000-0000-000000000010';

-- ================================================================
-- 10. STUDENTS (20 demo students across forms)
-- ================================================================
INSERT INTO students (id, institution_id, admission_number, first_name, last_name,
    date_of_birth, gender, nationality, status, admission_date, current_class_id, is_boarder) VALUES
    -- Form 1 East (5 students)
    ('st000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '2024/F1E/0001', 'Amara',    'Odhiambo',  '2009-03-15', 'F', 'Kenyan', 'active', '2024-01-08', 'cl000001-0000-0000-0000-000000000001', false),
    ('st000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', '2024/F1E/0002', 'Brian',    'Kipchoge',  '2009-07-22', 'M', 'Kenyan', 'active', '2024-01-08', 'cl000001-0000-0000-0000-000000000001', true),
    ('st000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', '2024/F1E/0003', 'Cynthia',  'Muthoni',   '2009-11-08', 'F', 'Kenyan', 'active', '2024-01-08', 'cl000001-0000-0000-0000-000000000001', true),
    ('st000001-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', '2024/F1E/0004', 'Daniel',   'Otieno',    '2009-01-30', 'M', 'Kenyan', 'active', '2024-01-08', 'cl000001-0000-0000-0000-000000000001', false),
    ('st000001-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', '2024/F1E/0005', 'Esther',   'Wanjiru',   '2009-05-12', 'F', 'Kenyan', 'active', '2024-01-08', 'cl000001-0000-0000-0000-000000000001', false),
    -- Form 2 East (5 students)
    ('st000001-0000-0000-0000-000000000011', '11111111-1111-1111-1111-111111111111', '2023/F1E/0001', 'Felix',    'Kamau',     '2008-02-18', 'M', 'Kenyan', 'active', '2023-01-09', 'cl000001-0000-0000-0000-000000000004', true),
    ('st000001-0000-0000-0000-000000000012', '11111111-1111-1111-1111-111111111111', '2023/F1E/0002', 'Gloria',   'Achieng',   '2008-09-25', 'F', 'Kenyan', 'active', '2023-01-09', 'cl000001-0000-0000-0000-000000000004', false),
    ('st000001-0000-0000-0000-000000000013', '11111111-1111-1111-1111-111111111111', '2023/F1E/0003', 'Henry',    'Njuguna',   '2008-06-14', 'M', 'Kenyan', 'active', '2023-01-09', 'cl000001-0000-0000-0000-000000000004', true),
    ('st000001-0000-0000-0000-000000000014', '11111111-1111-1111-1111-111111111111', '2023/F1E/0004', 'Irene',    'Cheptoo',   '2008-12-03', 'F', 'Kenyan', 'active', '2023-01-09', 'cl000001-0000-0000-0000-000000000004', false),
    ('st000001-0000-0000-0000-000000000015', '11111111-1111-1111-1111-111111111111', '2023/F1E/0005', 'Joseph',   'Mutua',     '2008-04-27', 'M', 'Kenyan', 'active', '2023-01-09', 'cl000001-0000-0000-0000-000000000004', false),
    -- Form 3 East (5 students)
    ('st000001-0000-0000-0000-000000000021', '11111111-1111-1111-1111-111111111111', '2022/F1E/0001', 'Kioni',    'Waweru',    '2007-08-10', 'F', 'Kenyan', 'active', '2022-01-10', 'cl000001-0000-0000-0000-000000000007', true),
    ('st000001-0000-0000-0000-000000000022', '11111111-1111-1111-1111-111111111111', '2022/F1E/0002', 'Laban',    'Kirui',     '2007-03-19', 'M', 'Kenyan', 'active', '2022-01-10', 'cl000001-0000-0000-0000-000000000007', false),
    ('st000001-0000-0000-0000-000000000023', '11111111-1111-1111-1111-111111111111', '2022/F1E/0003', 'Mary',     'Nyambura',  '2007-11-05', 'F', 'Kenyan', 'active', '2022-01-10', 'cl000001-0000-0000-0000-000000000007', true),
    ('st000001-0000-0000-0000-000000000024', '11111111-1111-1111-1111-111111111111', '2022/F1E/0004', 'Nathan',   'Simiyu',    '2007-07-28', 'M', 'Kenyan', 'active', '2022-01-10', 'cl000001-0000-0000-0000-000000000007', false),
    ('st000001-0000-0000-0000-000000000025', '11111111-1111-1111-1111-111111111111', '2022/F1E/0005', 'Olivia',   'Anyango',   '2007-01-16', 'F', 'Kenyan', 'active', '2022-01-10', 'cl000001-0000-0000-0000-000000000007', false),
    -- Form 4 East (5 students — KCSE candidates)
    ('st000001-0000-0000-0000-000000000031', '11111111-1111-1111-1111-111111111111', '2021/F1E/0001', 'Patrick',  'Mwangi',    '2006-05-22', 'M', 'Kenyan', 'active', '2021-01-11', 'cl000001-0000-0000-0000-000000000010', true),
    ('st000001-0000-0000-0000-000000000032', '11111111-1111-1111-1111-111111111111', '2021/F1E/0002', 'Queen',    'Otieno',    '2006-09-08', 'F', 'Kenyan', 'active', '2021-01-11', 'cl000001-0000-0000-0000-000000000010', false),
    ('st000001-0000-0000-0000-000000000033', '11111111-1111-1111-1111-111111111111', '2021/F1E/0003', 'Robert',   'Njenga',    '2006-02-14', 'M', 'Kenyan', 'active', '2021-01-11', 'cl000001-0000-0000-0000-000000000010', true),
    ('st000001-0000-0000-0000-000000000034', '11111111-1111-1111-1111-111111111111', '2021/F1E/0004', 'Sharon',   'Kerubo',    '2006-06-30', 'F', 'Kenyan', 'active', '2021-01-11', 'cl000001-0000-0000-0000-000000000010', false),
    ('st000001-0000-0000-0000-000000000035', '11111111-1111-1111-1111-111111111111', '2021/F1E/0005', 'Thomas',   'Rotich',    '2006-12-11', 'M', 'Kenyan', 'active', '2021-01-11', 'cl000001-0000-0000-0000-000000000010', true)
ON CONFLICT (institution_id, admission_number) DO NOTHING;

-- ================================================================
-- 11. PARENT USERS & LINKS (one parent per student demo)
-- ================================================================
INSERT INTO users (id, institution_id, role_id, email, phone, first_name, last_name,
    gender, password_hash, is_active, must_change_password) VALUES
    ('u0000002-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000005',
     'parent.amara@gmail.com', '+254711200001', 'John', 'Odhiambo',
     'M', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    ('u0000002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000005',
     'parent.brian@gmail.com', '+254711200002', 'Kipchoge', 'Senior',
     'M', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false),
    ('u0000002-0000-0000-0000-000000000011', '11111111-1111-1111-1111-111111111111',
     'r0000001-0000-0000-0000-000000000005',
     'parent.kioni@gmail.com', '+254711200011', 'Sarah', 'Waweru',
     'F', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewKyNiAYMyzJ4Uu.', true, false)
ON CONFLICT (institution_id, email) DO NOTHING;

INSERT INTO parent_guardians (id, institution_id, user_id, first_name, last_name,
    relationship, phone, email, is_primary_contact, is_fee_payer) VALUES
    ('pg000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111',
     'u0000002-0000-0000-0000-000000000001', 'John', 'Odhiambo',
     'father', '+254711200001', 'parent.amara@gmail.com', true, true),
    ('pg000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111',
     'u0000002-0000-0000-0000-000000000002', 'Kipchoge', 'Senior',
     'father', '+254711200002', 'parent.brian@gmail.com', true, true),
    ('pg000001-0000-0000-0000-000000000011', '11111111-1111-1111-1111-111111111111',
     'u0000002-0000-0000-0000-000000000011', 'Sarah', 'Waweru',
     'mother', '+254711200011', 'parent.kioni@gmail.com', true, true)
ON CONFLICT DO NOTHING;

INSERT INTO student_parent_links (id, institution_id, student_id, parent_id) VALUES
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'st000001-0000-0000-0000-000000000001', 'pg000001-0000-0000-0000-000000000001'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'st000001-0000-0000-0000-000000000002', 'pg000001-0000-0000-0000-000000000002'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'st000001-0000-0000-0000-000000000021', 'pg000001-0000-0000-0000-000000000011')
ON CONFLICT DO NOTHING;

-- ================================================================
-- 12. FEE STRUCTURE & INVOICES (Term 3 2024)
-- ================================================================
INSERT INTO fee_structures (id, institution_id, academic_year_id, name, is_active) VALUES
    ('fs000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111',
     'ay000001-0000-0000-0000-000000000001', 'Day Scholar 2024', true),
    ('fs000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111',
     'ay000001-0000-0000-0000-000000000001', 'Boarder 2024', true)
ON CONFLICT DO NOTHING;

INSERT INTO fee_components (id, institution_id, fee_structure_id, name, amount, is_mandatory) VALUES
    -- Day Scholar
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'fs000001-0000-0000-0000-000000000001', 'Tuition',           15000, true),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'fs000001-0000-0000-0000-000000000001', 'Activity Fee',       2000, true),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'fs000001-0000-0000-0000-000000000001', 'Lunch',              2500, false),
    -- Boarder
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'fs000001-0000-0000-0000-000000000002', 'Tuition',           15000, true),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'fs000001-0000-0000-0000-000000000002', 'Boarding',          18000, true),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'fs000001-0000-0000-0000-000000000002', 'Activity Fee',       2000, true),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'fs000001-0000-0000-0000-000000000002', 'Caution Money',      5000, false)
ON CONFLICT DO NOTHING;

-- Sample invoices for a few students (Term 3)
INSERT INTO fee_invoices (id, institution_id, student_id, term_id, invoice_number,
    total_amount, paid_amount, status, due_date) VALUES
    -- Amara (day) — fully paid
    ('fi000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111',
     'st000001-0000-0000-0000-000000000001', 'tm000001-0000-0000-0000-000000000003',
     'INV/2024/000001', 19500, 19500, 'paid', '2024-09-13'),
    -- Brian (boarder) — partial
    ('fi000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111',
     'st000001-0000-0000-0000-000000000002', 'tm000001-0000-0000-0000-000000000003',
     'INV/2024/000002', 40000, 20000, 'partial', '2024-09-13'),
    -- Kioni (boarder) — unpaid
    ('fi000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111',
     'st000001-0000-0000-0000-000000000021', 'tm000001-0000-0000-0000-000000000003',
     'INV/2024/000003', 40000, 0, 'unpaid', '2024-09-13')
ON CONFLICT (institution_id, invoice_number) DO NOTHING;

-- Sample payments
INSERT INTO fee_payments (id, institution_id, student_id, invoice_id, receipt_number,
    amount, payment_method, mpesa_reference, received_at) VALUES
    ('fp000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111',
     'st000001-0000-0000-0000-000000000001', 'fi000001-0000-0000-0000-000000000001',
     'RCP/2024/000001', 19500, 'mpesa', 'QKA6YTRNP1', NOW() - INTERVAL '5 days'),
    ('fp000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111',
     'st000001-0000-0000-0000-000000000002', 'fi000001-0000-0000-0000-000000000002',
     'RCP/2024/000002', 20000, 'mpesa', 'QKB7ZUSOQ2', NOW() - INTERVAL '3 days')
ON CONFLICT (institution_id, receipt_number) DO NOTHING;

-- ================================================================
-- 13. TIMELINE EVENTS for demo students
-- ================================================================
INSERT INTO student_timeline_events (institution_id, student_id, event_type, title, description, event_date)
SELECT '11111111-1111-1111-1111-111111111111', id,
    'enrolled', 'Enrolled at Upendo Secondary School',
    'Enrolled in Form 1 East for academic year 2024', admission_date
FROM students
WHERE institution_id = '11111111-1111-1111-1111-111111111111'
ON CONFLICT DO NOTHING;

-- ================================================================
-- 14. CHART OF ACCOUNTS (basic)
-- ================================================================
INSERT INTO chart_of_accounts (id, institution_id, code, name, type, sub_type) VALUES
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '1100', 'Cash at Hand',           'asset',   'current_asset'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '1110', 'M-Pesa Wallet',           'asset',   'current_asset'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '1120', 'Equity Bank Account',     'asset',   'current_asset'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '1200', 'Fees Receivable',         'asset',   'current_asset'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '1500', 'Fixed Assets',            'asset',   'fixed_asset'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '2100', 'Accounts Payable',        'liability','current_liability'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '2200', 'PAYE Payable',            'liability','current_liability'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '2210', 'NSSF Payable',            'liability','current_liability'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '2220', 'NHIF/SHIF Payable',       'liability','current_liability'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '3000', 'School Fund',             'equity',  null),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '4100', 'Tuition Fee Income',      'income',  'fee_income'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '4200', 'Boarding Fee Income',     'income',  'fee_income'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '4300', 'Government Capitation',   'income',  'grant_income'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '5100', 'Salaries - Teaching',     'expense', 'staff_cost'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '5110', 'Salaries - Non-Teaching', 'expense', 'staff_cost'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '5200', 'Teaching Materials',      'expense', 'operating'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '5300', 'Utilities',               'expense', 'operating'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '5400', 'Maintenance & Repairs',   'expense', 'operating'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '5500', 'Food & Catering',         'expense', 'operating')
ON CONFLICT (institution_id, code) DO NOTHING;

-- ================================================================
-- 15. DEMO ANNOUNCEMENTS
-- ================================================================
INSERT INTO announcements (institution_id, title, body, type, audience,
    is_published, published_at, created_by) VALUES
    ('11111111-1111-1111-1111-111111111111',
     'Term 3 2024 Fee Deadline Reminder',
     'Dear Parents and Guardians, kindly note that Term 3 2024 school fees are due by 13th September 2024. '
     'Students with outstanding balances risk being sent home. Please pay via M-Pesa Paybill 400200, '
     'Account Number = your child''s admission number.',
     'fee_notice', 'parents', true, NOW() - INTERVAL '2 days',
     'u0000001-0000-0000-0000-000000000003'),
    ('11111111-1111-1111-1111-111111111111',
     'Opening of Term 3 2024',
     'We welcome back all students for Term 3 2024. Reporting date is Monday 2nd September 2024. '
     'Boarders should report by 4:00 PM. Please come with all required school items.',
     'general', 'all', true, NOW() - INTERVAL '30 days',
     'u0000001-0000-0000-0000-000000000001'),
    ('11111111-1111-1111-1111-111111111111',
     'Form 4 Mock Examination Schedule',
     'Form 4 Mock examinations will be held from 7th to 18th October 2024. Timetables have been posted '
     'on the school notice board. Students are advised to prepare adequately.',
     'academic', 'students', true, NOW() - INTERVAL '7 days',
     'u0000001-0000-0000-0000-000000000002')
ON CONFLICT DO NOTHING;

COMMIT;
