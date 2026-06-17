# User Roles & Permissions Matrix
## Institution OS
**Version**: 1.0.0

---

## Permission Key

| Symbol | Meaning |
|--------|---------|
| ✅ | Full access |
| 📖 | Read only |
| 🔒 | Own data only |
| ✗ | No access |
| ⚙️ | Configurable per institution |

---

## Core Permission Matrix

| Permission | SysAdmin | Super Admin | Dept Admin | Teacher | Student | Parent | Accountant | Nurse | Librarian | Storekeeper | Driver | Security | Board |
|-----------|----------|-------------|------------|---------|---------|--------|------------|-------|-----------|-------------|--------|----------|-------|
| **STUDENTS** | | | | | | | | | | | | | |
| students:read | ✗ | ✅ | ✅ (dept) | 📖 (own class) | 🔒 | 🔒 (child) | 📖 | 📖 | 📖 | ✗ | ✗ | ✗ | ✗ |
| students:write | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| students:delete | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| students:export | ✗ | ✅ | ✅ (dept) | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **FEES** | | | | | | | | | | | | | |
| fees:read | ✗ | ✅ | ✅ (dept) | ✗ | 🔒 | 🔒 (child) | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | 📖 |
| fees:write | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| fees:receipt | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| fees:waiver | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ⚙️ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **LEDGER** | | | | | | | | | | | | | |
| ledger:read | ✗ | ✅ | ✅ (dept) | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | 📖 |
| ledger:write | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| ledger:approve | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **PAYROLL** | | | | | | | | | | | | | |
| payroll:read | ✗ | ✅ | ✅ (dept) | 🔒 | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| payroll:write | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| payroll:approve | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **STAFF** | | | | | | | | | | | | | |
| staff:read | ✗ | ✅ | ✅ (dept) | 🔒 | ✗ | ✗ | 📖 | ✗ | ✗ | ✗ | ✗ | ✗ | 📖 |
| staff:write | ✗ | ✅ | ✅ (dept) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **ATTENDANCE** | | | | | | | | | | | | | |
| attendance:read | ✗ | ✅ | ✅ (dept) | ✅ (own class) | 🔒 | 🔒 (child) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| attendance:write | ✗ | ✅ | ✅ (dept) | ✅ (own class) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| attendance:approve | ✗ | ✅ | ✅ (dept) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **GRADES** | | | | | | | | | | | | | |
| grades:read | ✗ | ✅ | ✅ (dept) | ✅ (own subject) | 🔒 | 🔒 (child) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| grades:write | ✗ | ✅ | ✅ (dept) | ✅ (own subject) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| grades:publish | ✗ | ✅ | ✅ (dept) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **REPORTS** | | | | | | | | | | | | | |
| reports:read | ✅ | ✅ | ✅ (dept) | ✅ (own) | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | 📖 |
| reports:export | ✅ | ✅ | ✅ (dept) | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **COMMUNICATIONS** | | | | | | | | | | | | | |
| announcements:read | ✗ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| announcements:write | ✗ | ✅ | ✅ (dept) | ✅ (class only) | ✗ | ✗ | ✅ | ✅ | ✗ | ✗ | ✗ | ✅ | ✗ |
| **DOCUMENTS** | | | | | | | | | | | | | |
| documents:read | ✗ | ✅ | ✅ (dept) | ✅ (own students) | 🔒 | 🔒 (child) | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ |
| documents:write | ✗ | ✅ | ✅ (dept) | ✅ (own students) | ✗ | ✗ | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ |
| documents:delete | ✗ | ✅ | ✅ (dept) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **SETTINGS** | | | | | | | | | | | | | |
| settings:read | ✅ | ✅ | 📖 | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| settings:write | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **USERS** | | | | | | | | | | | | | |
| users:read | ✅ | ✅ | ✅ (dept) | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| users:write | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| users:delete | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| **AUDIT** | | | | | | | | | | | | | |
| audit_logs:read | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |

---

## Scope Constraints (Enforced in Service Layer)

### Teacher Scope
- `students:read` — only students enrolled in classes assigned to this teacher
- `attendance:write` — only for their assigned classes on the current day
- `grades:write` — only for subjects they are the assigned teacher for
- `announcements:write` — only to their own classes (audience = `class:{their_class_id}`)

### Department Admin Scope
- All permissions with `(dept)` notation apply only within their department
- A Finance Dept Admin cannot access HR records
- An Academic HOD cannot post to the general ledger

### Parent Scope
- All `(child)` permissions apply only to students linked via `student_parent_links`
- One parent account can be linked to multiple children
- Cannot see other parents' children

### Student Scope
- `🔒` permissions are restricted to the student's own record only
- Students can view their own: grades, attendance, fee balance, timetable, report cards
- Students cannot see other students' data

---

## Special Access Rules

### Temporary External Access (Ministry Auditor, KNEC Officer)
Generated by Super Admin as a time-limited token:
- Scope defined at creation (e.g., `enrollment_statistics`, `exam_results_only`)
- Default TTL: 7 days
- No permanent account created
- All access logged in audit trail with token ID
- Cannot be extended without Super Admin action

### MFA Enforcement by Role (Configurable per institution)
| Role | Default MFA | Recommended |
|------|------------|-------------|
| Super Admin | Optional | Enforce |
| Accountant | Optional | Enforce |
| Department Admin | Optional | Recommend |
| Teacher | Optional | Optional |
| Parent | Optional | Optional |
| Student | Optional | Not recommended |
| All others | Optional | Optional |

---

## API Route → Permission Mapping

| Method | Route | Required Permission |
|--------|-------|-------------------|
| GET | /students | `students:read` |
| POST | /admissions | `students:write` |
| POST | /admissions/{id}/review | `students:write` |
| POST | /admissions/{id}/enroll | `students:write` |
| GET | /finance/students/{id}/account | `fees:read` |
| POST | /finance/pay/manual | `fees:receipt` |
| GET | /finance/summary | `fees:read` |
| GET | /academics/classes | `attendance:read` |
| POST | /academics/attendance | `attendance:write` |
| GET | /academics/grades/entries | `grades:read` |
| POST | /academics/grades/entries | `grades:write` |
| POST | /communications/announcements | `announcements:write` |
| GET | /communications/announcements | `announcements:read` |
| GET | /locker/{type}/{id} | `documents:read` |
| POST | /locker/{type}/{id} | `documents:write` |
| DELETE | /locker/documents/{id} | `documents:delete` |
| GET | /audit-logs | `audit_logs:read` |
| GET | /settings | `settings:read` |
| PUT | /settings | `settings:write` |
