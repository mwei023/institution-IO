# Business Requirements Specification
## Institution OS — School Management Platform
**Version**: 1.0.0  
**Date**: 2025-01-01  
**Status**: Approved for Development  
**Audience**: Engineering, Product, QA, Implementation Teams

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Scope & Objectives](#2-scope--objectives)
3. [Stakeholders & User Roles](#3-stakeholders--user-roles)
4. [Functional Requirements by Domain](#4-functional-requirements-by-domain)
   - 4.1 Identity & Access Management
   - 4.2 Student Lifecycle
   - 4.3 Parent Engagement
   - 4.4 Academics
   - 4.5 Examinations
   - 4.6 Human Resources
   - 4.7 Payroll
   - 4.8 Finance & Accounting
   - 4.9 Procurement
   - 4.10 Inventory Management
   - 4.11 Asset Management
   - 4.12 Boarding
   - 4.13 Clinic & Health
   - 4.14 Transport
   - 4.15 Library
   - 4.16 Security & Gate Management
   - 4.17 Visitor Management
   - 4.18 Governance
   - 4.19 Communication Hub
   - 4.20 Notification Center
   - 4.21 Analytics & Reporting
   - 4.22 Public Portal
5. [Supporting Platform Services](#5-supporting-platform-services)
6. [Non-Functional Requirements](#6-non-functional-requirements)
7. [Integration Requirements](#7-integration-requirements)
8. [Compliance & Regulatory Requirements](#8-compliance--regulatory-requirements)
9. [Implementation Phases](#9-implementation-phases)
10. [Pilot School Definition](#10-pilot-school-definition)
11. [Glossary](#11-glossary)

---

## 1. Executive Summary

Institution OS is a comprehensive, production-grade school management platform designed to serve Kenyan educational institutions. It replaces fragmented spreadsheets, paper registers, and disconnected tools with a single source of truth for every student, staff member, asset, and transaction.

### Problem Statement

Kenyan schools currently operate with:
- Student records split across paper registers, Excel files, and WhatsApp groups
- Fee collection managed through handwritten receipts with no real-time reconciliation
- Teacher attendance and lesson tracking done manually with no analytics visibility
- Parent communication dependent on informal WhatsApp groups with no audit trails
- Ministry reporting done retrospectively from incomplete data
- Boarding, clinic, transport, and library operating as completely isolated silos

### Solution

Institution OS unifies all school operations into one platform with:
- **One student profile** — admission through graduation, every interaction recorded
- **One financial ledger** — every fee, every payment, every expenditure traceable
- **One communication layer** — announcements, alerts, and conversations in one place
- **One analytics engine** — principals and HODs get actionable intelligence, not raw data
- **Role-appropriate interfaces** — a teacher's view differs fundamentally from an accountant's

### Design Philosophy

- **Simplicity over complexity** — each screen answers one user goal
- **Workflow first** — users think in tasks, not modules
- **Trust first** — every action audited, every document traceable
- **Mobile-first for parents, desktop-powerful for admins**
- **Configurable, not custom** — intelligent defaults with a workflow builder

---

## 2. Scope & Objectives

### In Scope (Phase 1 — MVP)

| Domain | Coverage |
|--------|----------|
| Identity & Access | Full RBAC, JWT auth, MFA placeholder |
| Student Lifecycle | Admission → enrollment → active profile |
| Parent Portal | Dashboard, fee payment (M-Pesa), attendance alerts |
| Academics | Classes, subjects, timetables, attendance, gradebook |
| Finance | Fee structures, receipting, basic general ledger |
| Communication | Announcements, SMS stub, in-app messaging |
| Digital Locker | Document upload, viewing, basic access controls |
| Reporting | CSV exports, basic dashboards |

### In Scope (Future Phases — Schema Ready Now)

Boarding, Clinic, Transport, Library, HR, Payroll, Procurement, Inventory, Asset Management, Security, Visitor Management, Governance, Public Portal, Advanced Analytics.

### Out of Scope

- AI/ML predictive models (Phase 3+)
- Blockchain-based certificates
- Dedicated native iOS/Android applications (PWA only in Phase 1)
- Multi-currency support (KES only)
- Multi-language support (English only in Phase 1)

### Primary Objectives

1. Digitize the complete student lifecycle from lead to alumni
2. Provide real-time financial visibility to school leadership
3. Replace informal WhatsApp parent communication with a structured channel
4. Enable data-driven academic decisions (attendance trends, grade distributions)
5. Ensure complete audit trails for every action across all modules
6. Comply with Kenya's Data Protection Act, 2019

---

## 3. Stakeholders & User Roles

### 3.1 Role Overview

Institution OS has 15 distinct roles. Each role maps to a specific job function and carries a tailored permission set.

| # | Role | Primary Location | Primary Concern |
|---|------|-----------------|-----------------|
| 1 | SysAdmin | Platform-wide | System health, tenant management |
| 2 | School Super Admin | School-wide | Strategic oversight, compliance |
| 3 | Department Admin | Department | HOD, Finance Manager, HR Manager |
| 4 | Teacher | Classroom | Lesson delivery, student progress |
| 5 | Student | Learner | Academic progress, timetable |
| 6 | Parent/Guardian | External | Child welfare, fees, communication |
| 7 | Nurse/Matron | Clinic | Student health, emergencies |
| 8 | Accountant | Finance Office | Ledger, payroll, statutory returns |
| 9 | Librarian | Library | Catalogue, borrowing, digital resources |
| 10 | Storekeeper | Store/Inventory | Stock management, asset tracking |
| 11 | Driver | Transport | Routes, vehicle logs, manifests |
| 12 | Cook | Cafeteria | Meal planning, dietary tracking |
| 13 | Janitor | Campus | Task logs, maintenance requests |
| 14 | Security Officer | Gate | Movement logs, visitor management |
| 15 | Board Member | Governance | Budget approval, strategic plans |

### 3.2 Role Descriptions

#### Role 1: SysAdmin
The SysAdmin is an Anthropic/vendor-side operator who manages the multi-tenant platform infrastructure.

**Responsibilities:**
- Onboard new institutions (create tenants, assign Super Admins)
- Monitor system health, uptime, and performance
- Manage backups, restore procedures, and disaster recovery
- Review audit logs for security incidents
- Manage subscriptions and billing (future)

**Does NOT have access to:**
- Individual student or staff personal data within institutions
- Financial transactions within institutions
- Day-to-day school operations

#### Role 2: School Super Admin (Principal / Registrar)
The most powerful role within an institution. Typically the Principal or Registrar.

**Responsibilities:**
- Configure school settings (name, logo, term dates, fee structures)
- Manage all user accounts within the institution
- Approve admissions and enrollment
- Oversee KNEC examination registration
- Access all dashboards and reports
- Manage the discipline register
- Configure communication channels (SMS provider, WhatsApp Business API)
- Set up the workflow builder (approval chains)

#### Role 3: Department Admin (HOD / Finance Manager / HR Manager)
A mid-level administrative role scoped to a specific department.

**Responsibilities (Academic HOD):**
- Manage teachers in their department
- Review lesson plans and attendance
- Generate subject-level analytics
- Manage department-specific resources

**Responsibilities (Finance Manager / Accountant):**
- Full access to fee management, ledger, receipts, payroll
- Bank reconciliation and financial reporting
- Procurement approvals up to a defined threshold

**Responsibilities (HR Manager):**
- Manage staff records, contracts, and leave
- Oversee recruitment pipeline
- Generate HR reports

#### Role 4: Teacher
The most numerous role. Interacts with the system daily for core academic tasks.

**Responsibilities:**
- Record daily attendance for their classes
- Create and mark assignments
- Enter grades into the gradebook
- Create lesson plans (weekly/termly)
- Send messages to parents of their students
- View their own timetable
- Access student profiles (academic data only)

**Does NOT have access to:**
- Financial data (except their own payslip)
- HR records of other staff
- Discipline records not involving their students

#### Role 5: Student
A self-service role for active enrolled students (secondary and above).

**Responsibilities:**
- View personal timetable
- View grades and report cards
- Submit assignments (if digital submission is enabled)
- Check fee balance (read-only)
- View library borrowings
- Access announcements

#### Role 6: Parent / Guardian
An external-facing role. One parent account can be linked to multiple children.

**Responsibilities:**
- View child's attendance status ("Is my child in school today?")
- Pay school fees via M-Pesa
- View fee balance and payment history
- Receive and acknowledge announcements
- View child's report cards
- Communicate with the class teacher
- Grant/revoke consent (medical, trips, photography)
- Track child's bus location (if transport module enabled)

#### Role 7: Nurse / Matron
Manages student health within the school clinic.

**Responsibilities:**
- Record student clinic visits (symptoms, diagnosis, treatment)
- Manage prescription and dispensary records
- View student allergy and medical history
- Manage isolation room occupancy
- Generate emergency alerts (critical cases)
- Refer students to external facilities
- Record medical inventory

#### Role 8: Accountant
Full access to the financial subsystem.

**Responsibilities:**
- Manage fee structures (amounts, categories, installment schedules)
- Generate fee invoices (termly/per student)
- Record manual payments (cash, cheque, bank transfer)
- Reconcile M-Pesa payments
- Post journal entries to the general ledger
- Prepare cashbook and bank reconciliation statements
- Process payroll and generate payslips
- File PAYE, NSSF, NHIF/SHIF returns
- Manage supplier accounts and payment runs
- Generate financial reports (P&L, balance sheet, budget vs actual)

#### Role 9: Librarian
Manages the school library catalogue and borrowing.

**Responsibilities:**
- Add, edit, and retire book records
- Process book checkouts and returns
- Manage overdue books and fines
- Add digital resources (e-books, links)
- Generate library usage reports

#### Role 10: Storekeeper
Manages physical inventory and assets.

**Responsibilities:**
- Receive goods from suppliers (GRN process)
- Issue items to departments (stores requisition)
- Maintain stock levels and reorder alerts
- Tag and track fixed assets
- Conduct periodic stock takes
- Report discrepancies

#### Role 11: Driver
A limited operational role for the transport module.

**Responsibilities:**
- View assigned route and student manifest
- Start/end trip and record mileage
- Report vehicle defects
- Mark students as boarded/alighted
- Receive trip instructions from transport admin

#### Role 12: Cook
Manages cafeteria operations.

**Responsibilities:**
- View the weekly meal plan
- Submit stock requisitions to the storekeeper
- Record meals served per day
- View student dietary restrictions and allergies
- Report equipment issues

#### Role 13: Janitor
Operational support for campus maintenance.

**Responsibilities:**
- View assigned maintenance tasks
- Update task status (in-progress, completed)
- Submit maintenance requests
- Record cleaning logs

#### Role 14: Security Officer
Manages physical access and incidents at the school gate.

**Responsibilities:**
- Log student gate entries/exits
- Check-in and check-out visitors
- Record incidents (theft, fights, damage)
- Manage lost and found items
- Initiate emergency broadcasts
- View visitor pre-registration

#### Role 15: Board Member
A governance-only role with no operational write access.

**Responsibilities:**
- View strategic plans, meeting minutes, and policies
- Approve budget submissions (electronic approval workflow)
- View executive-level financial summaries
- Receive governance announcements

---

## 4. Functional Requirements by Domain

### 4.1 Identity & Access Management

#### 4.1.1 Overview
The IAM module controls who can enter Institution OS, what they can see, and what actions they can perform. Every other module depends on IAM.

#### 4.1.2 Authentication

**Login Flow:**
1. User enters email/phone + password
2. System validates credentials (bcrypt hashed)
3. On success: issue JWT access token (15-minute expiry) + refresh token (7-day expiry, stored in httpOnly cookie)
4. On MFA enabled: redirect to TOTP entry screen before issuing tokens
5. Failed login: increment attempt counter; lock account after 5 consecutive failures (30-minute lockout)
6. All login events (success and failure) written to `audit_logs`

**Password Requirements:**
- Minimum 8 characters
- At least one uppercase, one lowercase, one digit
- Passwords stored as bcrypt hash (cost factor 12)
- Password reset via email OTP (6-digit, 15-minute expiry)
- Users cannot reuse their last 5 passwords

**Session Management:**
- Access token: JWT, 15-minute TTL, contains `user_id`, `institution_id`, `role`, `permissions[]`
- Refresh token: opaque random string, stored in Redis with `institution_id:user_id` key
- Logout: blacklist access token in Redis until expiry; delete refresh token
- Concurrent sessions: maximum 3 active sessions per user (configurable per institution)

**Multi-Factor Authentication (MFA):**
- Phase 1: TOTP (Google Authenticator / Authy compatible) — optional, admin can enforce per role
- Future: SMS OTP as fallback

#### 4.1.3 Authorization — RBAC Model

**Entities:**
- `Role` — named group of permissions (e.g., "Teacher")
- `Permission` — a specific action on a resource (e.g., `attendance:write`)
- `RolePermission` — mapping of permissions to roles
- `UserRole` — assignment of a role to a user within an institution

**Permission Format:** `{resource}:{action}`

| Resource | Actions |
|----------|---------|
| `students` | `read`, `write`, `delete`, `export` |
| `attendance` | `read`, `write`, `approve` |
| `grades` | `read`, `write`, `publish` |
| `fees` | `read`, `write`, `receipt`, `waiver` |
| `ledger` | `read`, `write`, `approve` |
| `payroll` | `read`, `write`, `approve` |
| `staff` | `read`, `write`, `delete` |
| `reports` | `read`, `export` |
| `settings` | `read`, `write` |
| `users` | `read`, `write`, `delete` |
| `announcements` | `read`, `write`, `delete` |
| `documents` | `read`, `write`, `delete` |
| `audit_logs` | `read` |

**Scope Constraints:**
- Teachers can only `read` students enrolled in their classes
- Department Admins can only `write` resources within their department
- Parents can only read data for their linked children
- Students can only read their own data

#### 4.1.4 User Management

**User Profile Fields:**
- `id` (UUID)
- `institution_id` (UUID)
- `email` (unique within institution)
- `phone` (E.164 format, e.g., +254712345678)
- `first_name`, `last_name`, `other_names`
- `gender` (M/F/Other)
- `date_of_birth`
- `national_id` / passport number
- `profile_photo_url`
- `role` (FK to roles table)
- `is_active` (boolean)
- `mfa_enabled` (boolean)
- `mfa_secret` (encrypted)
- `last_login_at`
- `password_changed_at`
- `created_at`, `updated_at`, `deleted_at`
- `created_by`, `updated_by`

**User Lifecycle:**
1. Super Admin creates user account (or bulk import via CSV)
2. System sends welcome email with temporary password
3. User logs in, forced to change password on first login
4. Super Admin assigns role(s) and department
5. User becomes active
6. Deactivation: `is_active = false` — user cannot log in but data retained
7. Deletion: soft delete (`deleted_at` set) — data retained, user invisible in UI

---

### 4.2 Student Lifecycle

#### 4.2.1 Overview
The student lifecycle spans from first contact (lead/prospect) through admission, enrollment, active academic journey, and finally graduation or exit, with alumni tracking thereafter.

#### 4.2.2 Lead / Prospect Management

**Purpose:** Capture interest from prospective students before formal application.

**Data Captured:**
- Prospect name, date of birth, gender
- Parent/guardian name, phone, email
- Intended class/grade
- Source (walk-in, referral, website, open day)
- Notes

**Workflow:**
1. Admissions officer creates prospect record
2. System schedules follow-up reminder
3. Officer converts prospect to formal applicant OR marks as lost

#### 4.2.3 Admission

**Application Form Fields:**
- Student: full name, date of birth, gender, birth certificate number, nationality
- Previous school: name, last class attended, leaving certificate number
- Parent 1 (required): name, relationship, national ID, phone, email, occupation, employer
- Parent 2 (optional): same fields
- Emergency contact (must be different from parents): name, relationship, phone
- Medical: known conditions, allergies, current medications, doctor name and phone, blood group
- Special needs: learning support required (yes/no), details
- Boarding: day scholar or boarder
- Transport: bus required (yes/no), preferred route
- Previous disciplinary issues: declaration
- Documents required: birth certificate, previous school leaving certificate/result slip, passport photos (4), parent national ID copy

**Admission Workflow (configurable via Workflow Engine):**
1. Admissions officer submits application → status: `submitted`
2. Principal reviews → approves or rejects → status: `approved` / `rejected`
3. On approval: fee invoice generated (admission fee, registration fee)
4. Parent pays admission fee → status: `awaiting_enrollment`
5. Admissions officer completes enrollment → status: `enrolled`
6. Student profile created with system-generated admission number

**Admission Number Format:** `{YEAR}/{CLASS_CODE}/{SEQUENCE}` e.g., `2024/F1/0042`

#### 4.2.4 Enrollment

**On Enrollment, System Creates:**
- Student profile (see 4.2.5)
- Parent/guardian user accounts (if email provided, else phone-based)
- Class assignment
- Fee ledger account
- Digital locker folder
- Transport manifest entry (if bus selected)
- Boarding room assignment (if boarder)

**Class Assignment Rules:**
- Manual assignment by admissions officer
- OR automatic stream placement based on available capacity

#### 4.2.5 Student Profile

**Profile Sections:**

*Personal Information:*
- Admission number (system-generated, immutable)
- Full name, date of birth, gender, nationality
- Birth certificate number
- Blood group, height, weight (updated annually)
- Photo (updated annually)
- Home address

*Academic Information:*
- Current class and stream
- Admission date
- KCPE index number (for Form 1) / previous result slip
- KNEC registration number (when applicable)

*Parent/Guardian Information:*
- Up to 4 linked parent/guardian accounts
- Primary contact designation
- Fee responsibility (who gets the fee invoice)

*Fee Account:*
- Current balance (arrears or credit)
- Payment history summary
- Active fee structure

*Academic History:*
- Term-by-term attendance percentage
- Grade summary per subject per term
- Exam results (internal and KCSE/KCPE)

*Discipline Record:*
- Incidents, sanctions, suspensions, counseling sessions

*Medical Record:*
- Known conditions, allergies, medications
- Clinic visit history
- Vaccination records

*Documents (Digital Locker):*
- Birth certificate
- KCPE certificate
- National ID (when obtained)
- Leave permission forms
- Medical consent forms

*Activity Log (Student Journey Timeline):*
Chronological timeline of every significant event: enrolled, class promotion, fee payment, attendance absence, discipline incident, clinic visit, library borrowing, assignment submitted, exam registered, graduated.

#### 4.2.6 Class Promotion

**End-of-Year Process:**
1. Super Admin initiates "Year-End" procedure
2. System validates all students have final grades for the year
3. Admin reviews and confirms promotion/retention decisions
4. System moves all promoted students to the next class
5. Retained students remain in current class
6. Graduating students moved to `alumni` status

#### 4.2.7 Withdrawal / Exit

**Exit Types:**
- `Transfer out`: moving to another school
- `Withdrawal`: family decision
- `Expulsion`: disciplinary
- `Deferral`: temporary leave
- `Deceased`

**Exit Process:**
1. Admin initiates exit, selects type and date
2. System checks outstanding fees — blocks exit if balance > 0 (configurable)
3. System generates leaving certificate
4. All services (transport, boarding, library) de-registered
5. Student status set to `inactive`
6. Parent portal access deactivated

#### 4.2.8 Alumni

**Alumni Profile retains:**
- Full academic history
- Examination results and certificates
- Contact information (updated by alumni)
- Employment/university information (self-reported)

**Alumni can:**
- Request official transcripts via public portal
- Register for alumni network events
- Update contact and career information

---

### 4.3 Parent Engagement

#### 4.3.1 Overview
Parents interact with Institution OS primarily through mobile web (PWA). The interface must be simple, fast, and focused on the three things parents care most about: Is my child safe? What do they owe? How are they performing?

#### 4.3.2 Parent Dashboard

**Primary Cards:**
1. **Attendance Status**: "Kioni is in school today" / "Kioni was absent — 3rd absence this term"
2. **Fee Balance**: KES 12,500 due by 15 March 2024
3. **Latest Results**: Form 2 Term 1 — Position 8/42
4. **Unread Announcements**: 2 new

**Quick Actions:**
- Pay Fees (M-Pesa)
- Message Class Teacher
- Download Report Card
- View Timetable

#### 4.3.3 Fee Payment

**M-Pesa Integration (Daraja API):**
1. Parent taps "Pay Fees"
2. Selects child, sees outstanding balance, enters amount
3. Confirms phone number for STK push
4. M-Pesa STK push sent to parent's phone
5. Parent enters M-Pesa PIN on their phone
6. System receives Daraja callback
7. Payment recorded in student fee account
8. Receipt generated and accessible in parent portal
9. SMS confirmation sent to parent
10. Parent portal balance updates in real-time

**Payment History:**
- Date, amount, M-Pesa transaction reference, receipt number
- PDF receipt downloadable

#### 4.3.4 Communication

- Parent receives all school announcements tagged for parents
- Parent can message their child's class teacher (not directly to all teachers)
- Teacher response visible in parent portal
- All messages audited

#### 4.3.5 Consent Management

**Consent Types:**
- Medical treatment consent (e.g., permission to administer first aid, minor procedures)
- School trip consent
- Photography/media consent
- Data processing consent (GDPR-aligned)

**Consent Flow:**
1. Admin creates consent request, sends to parents
2. Parent reviews details in portal, clicks Approve or Decline
3. Consent recorded with timestamp, IP address, parent identity
4. Nurse and trip coordinators can view consent status per student

---

### 4.4 Academics

#### 4.4.1 Classes & Streams

**Class Structure:**
- Class: e.g., "Form 1", "Grade 6", "Class 8"
- Stream: e.g., "Form 1 East", "Form 1 West"
- Class capacity: configurable (default 45)

**Class Entity:**
- `id`, `institution_id`
- `name` (e.g., "Form 1")
- `stream` (e.g., "East")
- `academic_year`
- `class_teacher_id` (FK to staff)
- `capacity`
- `current_enrollment` (computed)

#### 4.4.2 Subjects & Curriculum

**Subject Fields:**
- `code` (e.g., "ENG", "MAT", "BIO")
- `name` (e.g., "English Language")
- `department_id`
- `is_compulsory` (boolean)
- `knec_code` (for examination registration)
- `credit_hours_per_week`

**Subject-Class Assignment:**
- Many subjects can be assigned to many classes
- Each assignment has an `assigned_teacher_id`

#### 4.4.3 Timetables

**Timetable Generation:**
- Manual entry: admins build timetable period by period
- Periods: configurable (e.g., 8 periods per day × 5 days)
- Period entity: `day_of_week`, `period_number`, `start_time`, `end_time`, `subject_id`, `class_id`, `teacher_id`, `room`

**Timetable Constraints (future — Phase 2):**
- Teacher cannot be in two places simultaneously
- Room cannot host two classes simultaneously
- Subject minimum weekly frequency respected

**Published Timetable:**
- Once published, visible to teachers, students, and parents
- Changes require new version with change note

#### 4.4.4 Lesson Planning

**Lesson Plan Fields:**
- `teacher_id`, `subject_id`, `class_id`
- `term`, `week_number`
- `topic`, `subtopic`
- `learning_objectives` (text)
- `teaching_aids` (text)
- `activities` (text)
- `assessment_method` (text)
- `notes`
- `status`: draft / submitted / approved

**Workflow:**
1. Teacher creates weekly lesson plan
2. Submits to HOD for review
3. HOD approves or returns with comments
4. Approved plans archived for inspection readiness

#### 4.4.5 Attendance

**Attendance Taking:**
- Teacher opens "Take Attendance" for their current period
- System shows class register (all enrolled students)
- Teacher marks each student: `present`, `absent`, `late`, `excused`
- Submission locks that session (cannot retroactively change without admin override + audit log)

**Attendance Rules:**
- Only the assigned class teacher or subject teacher can take attendance for that session
- Admin can override attendance with mandatory reason and audit trail
- Late arrivals (marked `late`) after a configurable grace period count as half-day absences for reporting

**Absenteeism Alerts:**
- If a student is absent without prior notification: SMS alert to parents within 30 minutes of first period
- If a student reaches 3 consecutive absences: email alert to class teacher and Super Admin
- If attendance drops below 75% (configurable threshold): parent flag in student profile

**Attendance Records:**
- Daily summary per student per class
- Monthly and termly attendance percentage
- Teacher attendance (recorded by HOD or deputy)

#### 4.4.6 Assignments

**Assignment Entity:**
- `title`, `description` (rich text)
- `subject_id`, `class_id`, `teacher_id`
- `due_date`
- `max_marks`
- `submission_type`: physical / digital / both
- `attachments` (files in digital locker)
- `status`: draft / published / closed

**Submission Tracking:**
- Mark each student: `submitted` / `not_submitted` / `excused`
- Enter marks (0 to max_marks)
- Optional feedback comment per student
- Late submissions flagged automatically

#### 4.4.7 Gradebook

**Grade Entry:**
- Teacher opens gradebook for their subject + class + term
- Enter marks for each assessment component (CAT 1, CAT 2, Midterm, End-term)
- Component weights configurable per subject per class
- System computes weighted total automatically

**Grade Scale (configurable per institution):**
| Mark Range | Grade | Points |
|------------|-------|--------|
| 80–100 | A | 12 |
| 75–79 | A- | 11 |
| 70–74 | B+ | 10 |
| 65–69 | B | 9 |
| 60–64 | B- | 8 |
| 55–59 | C+ | 7 |
| 50–54 | C | 6 |
| 45–49 | C- | 5 |
| 40–44 | D+ | 4 |
| 35–39 | D | 3 |
| 30–34 | D- | 2 |
| 0–29 | E | 1 |

#### 4.4.8 Report Cards

**Report Card Generation:**
1. End of term: Super Admin initiates report generation
2. System compiles grades for all subjects for all students
3. Computes aggregate marks, grade points, class position, stream position
4. Generates teacher comments (teacher fills per student, max 150 characters)
5. Class teacher adds overall comment
6. Principal signs off (electronic approval)
7. Report cards published to parent portal
8. PDF versions generated for printing

**Report Card Contents:**
- Student name, admission number, class, term, year
- Subject-by-subject marks, grades, and teacher remark
- Aggregate grade points and position in class
- Attendance summary (days present / total days)
- Discipline summary (incidents this term)
- Class teacher comment
- Principal comment
- Next term opening date

---

### 4.5 Examinations

#### 4.5.1 Internal Examinations

**Exam Schedule Entity:**
- `name` (e.g., "Form 4 Mock 2024")
- `type`: CAT / Midterm / End-term / Mock / Pre-Mock
- `term`, `academic_year`
- `start_date`, `end_date`
- Exam papers per subject per class

**Exam Results Entry:**
- Exam coordinator uploads results per subject (CSV or manual entry)
- System validates: marks within range, all students covered
- Results locked after confirmation by HOD
- Statistical analysis auto-generated (mean, median, std dev, pass rate)

#### 4.5.2 KNEC Registration

**KCSE Registration (Form 4):**
1. Admissions officer opens KNEC registration module
2. System lists all Form 4 students
3. Officer verifies student data (name spelling, DOB, gender vs KNEC requirements)
4. Assigns subjects per student (compulsory + optional clusters)
5. Validates no student exceeds maximum subject count
6. Generates KNEC registration form (PDF) per student
7. Generates centre summary (total candidates per subject)
8. Exports data in KNEC-required CSV format
9. Records submission date and acknowledgement number

**KCPE Registration (Class 8) — same flow adapted for primary.**

#### 4.5.3 Examination Results (External)

- Upload KCSE/KCPE results from KNEC (CSV import)
- System maps to student records by index number
- Results visible on student profile and in alumni module
- Statistical analysis: school mean grade, grade distribution

---

### 4.6 Human Resources

#### 4.6.1 Staff Records

**Staff Profile Fields:**
- TSC number (for teachers), Staff ID
- Full name, DOB, gender, nationality
- National ID / passport
- Contact: phone, email, home address
- Next of kin (name, relationship, phone)
- Academic qualifications (degree, institution, year)
- Professional certifications (teaching certificate, etc.)
- Employment type: permanent / contract / part-time
- Department, designation/job title
- Reporting line (supervisor)
- Date of employment, probation end date
- Bank details (for payroll — encrypted)
- NSSF number, NHIF/SHIF number, KRA PIN
- Profile photo
- Digital Locker: certificates, contract, ID copy

#### 4.6.2 Recruitment

**Vacancy → Hire Workflow:**
1. HOD submits vacancy request → approved by Principal
2. HR posts vacancy (internal notice or public portal)
3. Applications received → shortlisted by HR
4. Interview scheduled → scores recorded per panel member
5. Background check initiated (police clearance certificate required)
6. Offer letter generated and sent
7. Candidate accepts → staff record created
8. Documents requested (certificates, ID, photos)
9. Contract generated and signed (electronic signature Phase 2)
10. Staff added to payroll

#### 4.6.3 Leave Management

**Leave Types:**
- Annual leave (configurable days per year per grade)
- Sick leave (with medical certificate requirement for >2 consecutive days)
- Maternity leave (3 months per Kenya Employment Act)
- Paternity leave (2 weeks)
- Compassionate leave (3 days per bereavement)
- Study leave
- Unpaid leave

**Leave Request Flow:**
1. Staff applies for leave (dates, type, reason)
2. Department Admin reviews → approves or declines
3. If approved: timetable gap flagged, cover teacher assigned
4. Leave deducted from leave balance
5. Leave calendar updated (visible to HOD)

**Leave Balance:**
- Accrual method configurable (monthly or per full year credited upfront)
- Carry-over policy: configurable (max days to carry forward)
- Leave encashment: configurable (cash out unused leave at year end)

#### 4.6.4 Performance Management

**Annual Appraisal Cycle:**
1. Performance targets set at start of year (teacher and HOD agree)
2. Mid-year review: self-assessment + HOD review
3. End-year review: self-assessment + HOD review + classroom observation scores
4. Performance score computed (1–5 scale)
5. Score linked to promotion eligibility and increment schedule

**Performance Dimensions (Teachers):**
- Lesson delivery quality (classroom observation)
- Student academic outcomes (% pass rate improvement)
- Attendance and punctuality
- Participation in extra-curricular
- Curriculum delivery completion rate
- Professional development undertaken

#### 4.6.5 Staff Training & Development

- Training record per staff member
- CPD hours tracked (continuing professional development)
- Training calendar (workshops, seminars)
- Training completion certificates stored in digital locker

#### 4.6.6 Retirement & Exit

**Exit Process:**
1. HR initiates exit record (type: resignation / retirement / dismissal / redundancy)
2. System generates clearance checklist (library books, assets, leave balance settlement)
3. Final payroll run triggered (includes terminal benefits calculation)
4. Experience letter / service certificate generated
5. System access deactivated
6. Staff record archived (not deleted — required for statutory audits)

---

### 4.7 Payroll

#### 4.7.1 Payroll Structure

**Earnings:**
- Basic salary
- House allowance
- Transport allowance
- Hardship allowance
- Acting allowance
- Any custom allowance

**Statutory Deductions (Kenya):**
- PAYE (progressive tax per KRA bands — updated each financial year)
- NSSF (Tier I: 6% of pensionable pay up to KES 6,000; Tier II: variable)
- NHIF/SHIF (income-based premium per government schedule)
- Affordable Housing Levy (1.5% of gross salary — effective 2024)
- NITA Levy (KES 50 per month for employers with >5 employees)

**Voluntary Deductions:**
- SACCO loan repayments
- Staff welfare contribution
- Union dues
- Salary advance recovery
- Any custom deduction

#### 4.7.2 Payroll Run Process

1. HR confirms active staff list for the month
2. Payroll admin verifies salary changes (new hires, promotions, increments)
3. Leave without pay deductions applied
4. Allowances and arrears added
5. System computes all statutory deductions per KRA formula
6. Payroll preview generated — admin reviews exceptions
7. Super Admin / Finance Manager approves payroll run
8. Payslips generated (PDF per employee, accessible in self-service portal)
9. Bank payment file generated (CSV format for bank bulk transfer)
10. PAYE return generated (KRA format)
11. NSSF contribution schedule generated
12. NHIF/SHIF schedule generated
13. All payments posted to general ledger

#### 4.7.3 Payslip

**Payslip Contents:**
- Employer name, employee name, staff ID, TSC number
- Pay period (month/year)
- All earnings itemized
- All deductions itemized
- Gross pay, total deductions, net pay
- Bank account last 4 digits
- Cumulative YTD figures
- Employer contribution to NSSF/NHIF

---

### 4.8 Finance & Accounting

#### 4.8.1 Chart of Accounts

**Account Categories:**
- `1000–1999`: Assets (Cash, M-Pesa Wallet, Bank Accounts, Receivables, Prepayments, Fixed Assets)
- `2000–2999`: Liabilities (Payables, Accruals, Tax payable, Loans)
- `3000–3999`: Equity / Fund Balance
- `4000–4999`: Income (Tuition fees, Boarding fees, Capitation grants, Other income)
- `5000–5999`: Operating Expenses (Salaries, Teaching materials, Utilities, Maintenance)
- `6000–6999`: Capital Expenditure

**Account Entity:**
- `code`, `name`, `type` (Asset/Liability/Equity/Income/Expense)
- `parent_id` (for sub-accounts)
- `is_active`

#### 4.8.2 Fee Management

**Fee Structure:**
- Configurable per class per term per academic year
- Components: Tuition, Boarding, Transport, Lunch, Uniform Deposit, Caution Money, Activities
- Each component: amount, mandatory/optional, due date

**Fee Invoice Generation:**
- At enrollment: one-time admission and registration fees
- Each term start: termly fee invoice auto-generated per student per active fee structure
- Invoice includes: all mandatory components + any optional components the student has opted into

**Fee Accounts:**
- Each student has a running fee account
- Debit: invoice generated
- Credit: payment received
- Balance = total invoiced − total paid

**Fee Waivers / Scholarships:**
- Admin can create a waiver / scholarship for specific fee components
- Waivers must be approved by Super Admin
- Waiver recorded in audit log with reason

#### 4.8.3 Receipting

**Receipt Generated On:**
- M-Pesa payment (automated on Daraja callback)
- Manual payment (cash/cheque/bank transfer — entered by accountant)

**Receipt Entity:**
- `receipt_number` (sequential, institution-scoped)
- `student_id`, `invoice_id`
- `amount_paid`, `payment_method`
- `m_pesa_reference` (if applicable)
- `bank_reference` (if applicable)
- `received_by` (user ID)
- `received_at`
- PDF receipt auto-generated, accessible to parent

#### 4.8.4 M-Pesa Daraja Integration

**Flows Supported:**
- STK Push (Lipa Na M-Pesa): parent-initiated payment
- C2B Paybill: parent pays via M-Pesa menu using paybill number + student admission number as account reference
- B2C: school-initiated refunds (Phase 2)

**Security:**
- Daraja API credentials stored encrypted
- Callback endpoint authenticated via Daraja IP whitelist + shared secret
- All Daraja events logged in `mpesa_transactions` table
- Idempotency: duplicate `MpesaReceiptNumber` silently ignored

#### 4.8.5 General Ledger

**Double-Entry Accounting:**
- Every financial event creates at least one debit and one credit journal entry
- Journal entry entity: `date`, `description`, `reference`, `lines[]` (account, debit, credit, narration)
- Trial balance, income statement, balance sheet generated from ledger

**Automated Postings:**
- Fee invoice: Debit Fees Receivable, Credit Tuition Fee Income
- Fee payment: Debit Cash/Bank, Credit Fees Receivable
- Payroll: Debit Salaries Expense, Credit Bank + PAYE Payable + NSSF Payable + etc.
- Procurement: Debit expense account, Credit Accounts Payable

#### 4.8.6 Bank Reconciliation

- Import bank statement (CSV or manual entry)
- System auto-matches entries by amount + date ± tolerance
- Unmatched items highlighted for manual review
- Reconciliation report: statement balance, adjusted book balance, differences

#### 4.8.7 Budgeting

**Budget Cycle:**
- Finance manager creates annual budget per account / department
- Budget submitted to Principal → submitted to Board for approval
- Once approved, system tracks actual vs budget per account
- Variance alerts when actual spend exceeds budget by configurable threshold (e.g., >10%)

#### 4.8.8 Government Capitation

- Record capitation grants received (primary: FDSE, secondary: national grants)
- Allocate to specific expenditure categories per government guidelines
- Generate capitation utilization report (Ministry format)

---

### 4.9 Procurement

**Purchase Request Flow:**
1. Any staff member with access submits purchase request (item, qty, estimated cost, justification)
2. Department Admin approves (up to KES threshold)
3. Finance Manager approves (above threshold)
4. Procurement officer raises LPO (Local Purchase Order) — sends to supplier
5. Supplier delivers → storekeeper captures GRN (Goods Received Note)
6. GRN matched to LPO (3-way match: PO + GRN + Supplier Invoice)
7. Finance approves payment → supplier invoice posted to accounts payable
8. Payment run processes supplier payment

**Entities:** Purchase Request, LPO, GRN, Supplier Invoice, Payment

---

### 4.10 Inventory Management

**Stock Item Entity:**
- `code`, `name`, `category` (stationery / cleaning / lab / kitchen / sports)
- `unit_of_measure` (pieces, reams, litres, kg)
- `reorder_level`, `reorder_quantity`
- `current_stock` (computed from movements)
- `unit_cost` (last purchase price)
- `storage_location`

**Stock Movements:**
- `GRN` (increase stock) — from procurement
- `Issue` (decrease stock) — from stores requisition
- `Adjustment` (± correction after stocktake — requires approval)
- `Transfer` (between stores/departments)

**Stocktake Process:**
1. Storekeeper initiates stocktake for a category
2. System shows expected stock
3. Storekeeper counts and enters actuals
4. System generates variance report
5. Finance Manager approves adjustments

---

### 4.11 Asset Management

**Asset Entity:**
- `asset_tag` (barcode/QR — printable label)
- `name`, `description`, `category`
- `serial_number`, `model`, `manufacturer`
- `purchase_date`, `purchase_cost`
- `supplier_id`
- `useful_life_years` (for depreciation)
- `depreciation_method` (straight-line / declining balance)
- `current_book_value` (computed)
- `condition`: excellent / good / fair / poor / condemned
- `assigned_to` (user or department)
- `location` (room/building)
- `warranty_expiry`
- `disposal_date`, `disposal_method`, `disposal_proceeds`

**Asset Lifecycle:**
1. Procurement triggers asset creation (high-value items auto-flagged)
2. Asset tagged and entered into register
3. Asset assigned to user / department
4. Maintenance scheduled (annual inspection, service intervals)
5. Condition updated after each inspection
6. Condemned asset: disposal approval required → proceeds posted to ledger

---

### 4.12 Boarding

#### 4.12.1 Overview
Boarding management covers dormitory administration, room assignments, night attendance, inspections, and boarder welfare.

#### 4.12.2 Dormitory Structure
- `Block` (e.g., "Kilimanjaro Block")
- `Dormitory` within block (e.g., "Dorm A")
- `Room` within dormitory (capacity, current occupants)
- `Bed` within room (labeled A, B, C... or numbered)

#### 4.12.3 Boarding Assignment
- Student assigned to bed on enrollment (or when boarding starts)
- Room capacity tracked in real-time
- Waitlist for full dormitories

#### 4.12.4 Night Attendance
- Matron / prefect takes night roll call
- Records: present / absent without leave / absent with leave
- Absence without leave triggers parent alert + security notification

#### 4.12.5 Dormitory Inspections
- Weekly inspection schedule
- Matron records cleanliness scores per room
- Individual student neatness scores
- Reports visible to Principal

#### 4.12.6 Lost Property
- Student reports lost item
- System creates lost property log
- Found item logged by security / matron
- Owner identified and item returned

---

### 4.13 Clinic & Health

#### 4.13.1 Student Clinic Visit

**Visit Record:**
- `student_id`, `visit_date`, `visit_time`
- `complaint` (free text)
- `vital_signs` (temperature, blood pressure, pulse — for secondary+)
- `diagnosis`
- `treatment_given`
- `medications_prescribed` (from stock)
- `nurse_id`
- `follow_up_required` (boolean), `follow_up_date`
- `referred_out` (boolean), `referral_facility`

#### 4.13.2 Medical Profile Per Student
- Blood group
- Known chronic conditions (asthma, diabetes, epilepsy, etc.)
- Allergies (food, medication, environmental)
- Current regular medications
- GP / family doctor name and phone
- Emergency medical authority (parent consent on file)

#### 4.13.3 Isolation Room
- `Bed` entity in clinic (similar to boarding)
- Student admitted to isolation: admission reason, date, expected release
- Parent notified on admission and discharge
- Meals arranged (flag for kitchen)

#### 4.13.4 Emergency Alerts
- Nurse can trigger "Medical Emergency" for a student
- Alert sent to: Principal, class teacher, parent(s), transport coordinator
- Alert includes student name, location, condition summary
- All emergency alerts logged for post-incident review

#### 4.13.5 Vaccination Records
- Record vaccination administered (date, vaccine, batch number, nurse)
- Track outstanding vaccinations per student (per national immunization schedule)

---

### 4.14 Transport

#### 4.14.1 Routes & Stops
- `Route` entity: name, description, estimated duration
- `Stop` entity: name, GPS coordinates, order_in_route, pickup_time, dropoff_time

#### 4.14.2 Vehicle Management
- `Vehicle`: registration plate, make, model, year, capacity
- `Driver`: linked staff record
- Insurance expiry, inspection certificate, PSV licence — with renewal alerts

#### 4.14.3 Student Manifests
- Each route has a morning manifest (AM) and evening manifest (PM)
- Students assigned to route + stop
- Driver views manifest on mobile — marks each student as boarded/missed

#### 4.14.4 Trip Logs
- Driver starts and ends each trip: system records departure time, arrival time, mileage
- Any incidents recorded (breakdown, accident, route deviation)

#### 4.14.5 Parent Visibility
- Parent can see their child's bus location (GPS integration — Phase 2)
- Parent receives push notification when bus is 10 minutes away from their stop (Phase 2)
- Parent receives alert if child was not on the bus (absent from manifest)

---

### 4.15 Library

#### 4.15.1 Book Catalogue
- `Book`: ISBN, title, author(s), publisher, year, edition, subject, call_number, condition
- `Copy`: copy_number, barcode, condition, status (available / borrowed / lost / retired)

#### 4.15.2 Borrowing
- Student / staff borrows copy: `Borrowing` record (copy_id, borrower_id, due_date, issued_by)
- Maximum concurrent borrowings configurable (default: 3 for students, 5 for teachers)
- Borrowing period configurable (default: 14 days)
- Renewal: extend due date once (if no holds)
- Return: condition noted, fine computed if overdue

#### 4.15.3 Fines
- Overdue fine rate: KES per day (configurable)
- Fine added to student fee account (or separate library account — configurable)
- Lost book: replacement cost charged

#### 4.15.4 Digital Resources
- URL links to digital resources, e-books, open-access journals
- Visible to students and teachers in library catalogue

---

### 4.16 Security & Gate Management

#### 4.16.1 Gate Logs
- Security officer logs every student entry/exit with timestamp
- Authorized exit: requires signed permission from class teacher / admin
- Unauthorized exit flagged → immediate alert to class teacher and parent

#### 4.16.2 Incident Reports
- Security officer files incident: type (theft, damage, assault, suspicious activity), description, parties involved, evidence attached
- Incident reviewed by Principal
- Incident may trigger discipline module entry

#### 4.16.3 Emergency Protocols
- Security can initiate: fire drill, lockdown, evacuation
- All registered users on campus receive emergency broadcast
- Roll call capability: class teachers account for students in real-time during emergency

---

### 4.17 Visitor Management

#### 4.17.1 Pre-Registration
- Parent / outside party registers visit in advance (via public portal)
- Provides: name, national ID, purpose, host (staff member), expected date/time
- Host receives notification to approve or decline

#### 4.17.2 Walk-in Visitors
- Security officer captures: name, national ID (photographed), purpose, host
- System sends host notification for approval
- Visitor badge number assigned and printed (future: physical badge printer integration)

#### 4.17.3 Visitor Log
- All visitors: check-in time, check-out time, purpose, host
- Visitors who overstay (no check-out by end of day) flagged
- Restricted individuals list: security alerted if blacklisted person attempts entry

---

### 4.18 Governance

#### 4.18.1 Board Management
- Board member directory (name, role on board, term start/end)
- Meeting scheduling: type (ordinary / special / AGM), date, venue, agenda

#### 4.18.2 Meeting Minutes
- Minutes drafted in system (rich text editor)
- Minutes circulated to board members for corrections (comment workflow)
- Minutes approved and signed (electronic) → immutable archive

#### 4.18.3 Policies & Procedures
- Document repository for school policies
- Version control: previous versions retained
- Review cycle: policy has a review date; system alerts when due
- Board approval workflow for new / amended policies

#### 4.18.4 Strategic Plans
- Upload strategic plan document
- Track strategic objectives and key performance indicators
- Progress reports submitted termly
- Board visible dashboard: objective → status → last updated

#### 4.18.5 Budget Approval
- Finance Manager submits annual budget document
- Board reviews and approves via electronic workflow
- Approved budget becomes the basis for variance reporting in finance module

---

### 4.19 Communication Hub

#### 4.19.1 Channels

**Announcement Broadcast:**
- Author: any admin / teacher (based on role)
- Audience: all school / specific class / specific role / specific parent group
- Type: general / academic / event / emergency / fee notice
- Delivery: in-app + push notification + optional SMS
- Read receipts tracked

**Groups:**
- Class groups (auto-created per class: teacher + parents)
- Department groups (teacher + HOD)
- Staff groups (all staff)
- Custom groups (admin-created)

**Direct Messaging:**
- Parent ↔ Class Teacher only
- Student ↔ Teacher only (secondary school)
- Staff ↔ Staff (any)
- All messages archived with sender, recipient, timestamp

#### 4.19.2 Document Sharing
- Attach documents from Digital Locker to messages or announcements
- External documents: upload on send

#### 4.19.3 Emergency Broadcast
- Highest priority message type
- Sent to ALL users instantly via all available channels (in-app, push, SMS, email)
- Initiated by: Principal, Security Officer (for campus incidents), Nurse (medical emergency)
- Acknowledgement required from: all teachers

---

### 4.20 Notification Center

**Notification Engine Architecture:**
- Every module publishes domain events (e.g., `student.absent`, `payment.received`, `assignment.due`)
- Notification service subscribes to events
- Notification preferences: per user, per event type, per channel
- Channels: in-app, push (PWA), SMS (via Africa's Talking or Twilio stub), email (SendGrid stub), WhatsApp Business API (stub)

**Notification Log:**
- Every notification: recipient, channel, content, sent_at, delivered_at, read_at
- Failed notifications: retry queue (3 attempts with exponential backoff)

**Critical Notifications (cannot be disabled):**
- Medical emergency
- Child absent from school
- Fee payment received
- Emergency broadcast

**Configurable Notifications:**
- Assignment due reminder
- Timetable change
- Report card published
- Fee invoice generated
- Library overdue

---

### 4.21 Analytics & Reporting

#### 4.21.1 Dashboards

**Principal Dashboard:**
- Students present today (% and number)
- Outstanding fee balance (aggregate)
- Incidents this week
- Students in clinic / isolation
- Upcoming events (next 7 days)
- Teacher attendance rate

**Teacher Dashboard:**
- Next period (subject, class, room)
- Attendance taken today (per class)
- Pending assignments to mark
- Unread parent messages

**Parent Dashboard:**
- Child attendance status
- Fee balance due
- Latest result summary
- Unread announcements

**Finance Dashboard:**
- Fee collection this term vs target
- Outstanding balance by class
- Expense vs budget (current month)
- Cash and bank balance

#### 4.21.2 Standard Reports

| Report | Audience | Frequency |
|--------|----------|-----------|
| Daily Attendance | Principal, HOD | Daily |
| Fee Collection Summary | Accountant, Principal | Weekly |
| Fee Balance by Class | Accountant | On-demand |
| Student Academic Performance | HOD, Principal | Per term |
| Teacher Lesson Completion | HOD | Monthly |
| Payroll Summary | Finance Manager | Monthly |
| Capitation Utilization | Principal, Board | Per term |
| Inventory Stock Report | Storekeeper | Monthly |
| Library Borrowing Statistics | Librarian | Monthly |
| Ministry Enrollment Statistics | Principal | Annual |

#### 4.21.3 Student Risk Engine (Phase 2)

**Risk Flags:**
- Attendance < 75%: "At Risk — Absenteeism"
- Grade decline > 15 points term-on-term: "Academic Decline"
- 3+ discipline incidents in a term: "Behaviour Concern"
- Fee balance > 50% of term fees: "Fee Defaulter"
- No parent message interaction in 60+ days: "Low Parental Engagement"

Risk score visible on student profile; class teacher and HOD alerted.

---

### 4.22 Public Portal

#### 4.22.1 School Website

**Pages:**
- Home (hero, mission, news, events, contact)
- About Us (history, vision, leadership team, accreditations)
- Academics (curriculum, subjects, results)
- Admissions (requirements, process, fees overview, online application form)
- Gallery (photos — consent-filtered)
- News & Events
- Contact Us (map, phone, email, contact form)

**Content Management:**
- Super Admin can update all public pages via built-in CMS
- News posts and events managed from admin panel, auto-publish to website

#### 4.22.2 Online Admissions

- Prospective parent completes online application form
- Uploads required documents
- Pays application fee via M-Pesa
- Admissions officer reviews in admin panel (converts to formal application in 4.2.3 flow)

#### 4.22.3 Results Lookup

- Former students can enter their admission number / index number
- System displays their KCSE/KCPE results (public — post official release)
- Certificate replacement request form (triggers admin workflow)

#### 4.22.4 External Access Portal

**Temporary Read-Only Access:**
- Super Admin generates time-limited access link (e.g., 7 days) with password
- Use case: Ministry auditor, KNEC officer, external examiner
- Access scope: configurable (e.g., enrollment statistics only, or exam results only)
- All access logged in audit trail
- No account creation required (token-based)

---

## 5. Supporting Platform Services

### 5.1 Workflow Engine

**Purpose:** Allow Super Admins to configure approval chains for any multi-step process without code changes.

**Configurable Workflows Include:**
- Admission approval chain
- Leave request approval
- Purchase request approval
- Policy update approval
- Student exit approval

**Workflow Builder:**
- Steps (who approves), conditions (e.g., "if amount > KES 50,000, add Finance Manager"), notifications at each step
- Timeout handling (if approver doesn't act in N days, escalate)
- Delegation (approver can delegate during leave)

### 5.2 Calendar Engine

**Global Institution Calendar:**
- Term dates, exam schedules, public holidays, school events
- Role-based views: teacher sees their timetable + school events; parent sees child's timetable + school events
- iCal export (for Google Calendar / Outlook sync)
- Event types: academic, exam, co-curricular, holiday, boarding, transport

### 5.3 Digital Locker

**Architecture:**
- Folder per entity (student, staff, institution)
- Files stored in object storage (S3-compatible, e.g., MinIO for on-premises)
- Metadata in PostgreSQL
- Access control: file-level read/write permissions tied to RBAC

**File Operations:**
- Upload (with virus scan placeholder)
- Download (direct URL with signed token, 1-hour expiry)
- Preview (PDF, images rendered in browser)
- Version history (new uploads create new version, previous retained)
- Delete (soft delete — admin can restore within 30 days)

**Document Types per Student:**
- Birth certificate, admission letter, fee receipts, report cards, medical consent forms, trip consent forms, KCSE certificate

### 5.4 Audit Engine

**Every significant action creates an audit record:**
- `table_name`, `record_id`
- `action` (INSERT / UPDATE / DELETE / VIEW)
- `old_values` (JSONB — for UPDATE/DELETE)
- `new_values` (JSONB — for INSERT/UPDATE)
- `changed_by` (user_id)
- `institution_id`
- `ip_address`, `user_agent`
- `created_at`

**Audit Log Properties:**
- Immutable: no UPDATE or DELETE on audit table ever (append-only)
- Retained: minimum 7 years (Kenya Public Archives requirement)
- Searchable: filter by user, action type, table, date range
- Exportable: PDF / CSV for external audit requests

### 5.5 Consent Registry

**Consent Record:**
- Student ID
- Consent type (medical / photography / trip / data_processing)
- Granted by: parent ID
- Granted at: timestamp
- IP address at time of consent
- Consent text shown (versioned — if policy changes, new consent required)
- Revoked at (if revoked)

### 5.6 Backup & Recovery

**Backup Policy:**
- Hourly: incremental PostgreSQL WAL archive to object storage
- Daily: full PostgreSQL dump — retained 30 days
- Weekly: full dump — retained 12 weeks
- Monthly: full dump — retained 24 months

**Restore Procedure:**
- Documented in `disaster_recovery.md`
- RTO: ≤ 4 hours (Recovery Time Objective)
- RPO: ≤ 1 hour (Recovery Point Objective)
- Monthly restore test to staging environment (automated via CI job)

---

## 6. Non-Functional Requirements

### 6.1 Performance

| Metric | Target |
|--------|--------|
| Page load time (P95) | < 2 seconds |
| API response time (P95) | < 200 ms (read), < 500 ms (write) |
| Dashboard data refresh | < 3 seconds |
| Report generation (standard) | < 10 seconds |
| Concurrent users (Phase 1) | 500 |
| Concurrent users (Phase 2) | 5,000 |

### 6.2 Availability

- Uptime SLA: 99.5% (Phase 1), 99.9% (Phase 2)
- Planned maintenance: Saturdays 11pm–2am EAT
- Zero-downtime deployments (Phase 2)

### 6.3 Scalability

- Horizontal scaling of backend (stateless Go binary) via load balancer
- Read replicas for PostgreSQL (Phase 2)
- Redis cluster for session and cache (Phase 2)
- `institution_id` on every table enables future sharding by institution

### 6.4 Security

- All data in transit: TLS 1.2+ (HTTPS enforced, HTTP redirects)
- All passwords: bcrypt cost 12
- PII fields at rest: AES-256 encryption for bank details, national IDs
- JWT signing: RS256 with 2048-bit RSA keys
- Rate limiting: 100 requests/minute per IP (unauthenticated), 1,000/minute per user (authenticated)
- SQL injection: parameterized queries only (no string concatenation)
- XSS: Content Security Policy headers; input sanitized before storage
- CSRF: SameSite=Strict cookies + CSRF token on state-changing requests

### 6.5 Reliability

- All external API calls (M-Pesa, SMS) wrapped with circuit breaker and retry
- Database transactions for all multi-step operations
- Queue-based processing for notifications (no synchronous SMS in request path)
- Graceful degradation: if SMS provider is down, in-app notification still delivered

---

## 7. Integration Requirements

| Integration | Type | Phase | Notes |
|-------------|------|-------|-------|
| M-Pesa Daraja API | REST (STK Push + C2B) | Phase 1 | Safaricom's payment API — requires registered paybill |
| Africa's Talking SMS | REST | Phase 1 (stub) | SMS for notifications |
| SendGrid Email | REST | Phase 1 (stub) | Transactional email |
| WhatsApp Business API | REST | Phase 2 | Requires Meta approval |
| KRA iTax | File export | Phase 2 | PAYE return submission |
| KNEC Portal | File export / API | Phase 2 | Exam registration data |
| Google Calendar | iCal | Phase 2 | Calendar sync |
| GPS Tracking (transport) | WebSocket | Phase 2 | Vehicle location streaming |
| Ministry of Education NEMIS | API | Phase 2 | National Education Management Information System |

---

## 8. Compliance & Regulatory Requirements

### 8.1 Kenya Data Protection Act, 2019

- All personal data processing registered with Office of Data Protection Commissioner (ODPC)
- Privacy notice displayed to users on registration
- Data subject rights implemented: access, correction, deletion (with legal retention exceptions), portability
- Data breach notification: report to ODPC within 72 hours
- Data Protection Officer (DPO) designation required for institutions > threshold

### 8.2 Children's Data Protection

- Students under 18 are data subjects requiring parental consent
- Explicit consent captured via Consent Registry (Section 5.5)
- Student data not used for advertising or profiling
- Photos of minors require parental consent flag = true before display

### 8.3 Financial Regulations

- Fee receipts compliant with KRA ETR requirements (electronic receipt — Phase 2)
- PAYE remittance by 9th of following month (statutory deadline)
- NSSF remittance by last day of month
- NHIF/SHIF remittance by 9th of following month
- Annual financial statements per Kenya Schools and Colleges Act

### 8.4 Education Sector Compliance

- NEMIS registration compliance (Phase 2)
- TSC staff registration verification
- KNEC examination security protocols
- Basic Education Act, 2013 record retention requirements

---

## 9. Implementation Phases

### Phase 1 — Foundation (Months 1–4)

**Deliverables:**
- Identity & Access (full RBAC, JWT, MFA placeholder)
- Student Lifecycle (admission → enrollment → profile)
- Parent Portal (dashboard, M-Pesa fee payment, attendance view)
- Academics (classes, subjects, timetable, attendance, gradebook)
- Finance (fee structures, invoicing, receipting, basic ledger)
- Communication (announcements, SMS stub, in-app messaging)
- Digital Locker (upload, view, basic permissions)
- Basic Reporting (CSV exports, dashboards)
- All database schemas for ALL modules

**Tech:**
- Go backend, Next.js frontend, PostgreSQL, Redis
- Docker Compose deployment
- GitHub Actions CI/CD to VPS

### Phase 2 — Core Operations (Months 5–8)

**Deliverables:**
- HR & Payroll (full payroll run, statutory deductions, payslips)
- Procurement & Inventory (full PR → PO → GRN → payment flow)
- Asset Management (register, assignment, depreciation)
- Boarding (dormitories, night attendance, inspections)
- Clinic (visits, prescriptions, isolation, emergency alerts)
- Transport (routes, manifests, trip logs)
- Library (catalogue, borrowing, fines)
- Security & Visitor Management
- Governance (board portal, minutes, policies)
- Advanced Reporting (Student Risk Engine, full financial reports)
- WhatsApp integration

### Phase 3 — Intelligence & Growth (Months 9–12)

**Deliverables:**
- AI Insights (grade prediction, attendance risk models)
- Public Portal (school website + online admissions)
- Ministry Reports (NEMIS integration)
- Alumni Network
- GPS Transport Tracking
- Multi-tenant Admin Console
- Advanced Analytics (custom report builder)
- Mobile PWA offline sync

---

## 10. Pilot School Definition

### School: Upendo Secondary School, Nairobi

**Demographics:**
- 850 students (Form 1–4)
- 42 teaching staff
- 18 non-teaching staff
- 3 streams per form (e.g., Form 2 East, Form 2 West, Form 2 North)
- 6 subjects per stream per form
- 60% boarders, 40% day scholars
- School bus: 3 routes, 2 buses

**Term Structure:**
- Term 1: January – April
- Term 2: May – July
- Term 3: September – November

**Fee Structure (per term):**
- Form 1–4 Day Scholar: KES 22,000
- Form 1–4 Boarder: KES 42,000
- Components: Tuition KES 15,000, Boarding KES 18,000 (boarders only), Activity KES 2,000, Transport KES 3,500 (bus users), Lunch KES 2,500 (day scholars)

**Financial Scale:**
- Annual fee revenue: approximately KES 32 million
- Monthly payroll: approximately KES 4.2 million
- Annual procurement: approximately KES 8 million

**Test Users (Seed Data):**
- 1 Super Admin (Principal)
- 2 Department Admins (Finance HOD, Academic HOD)
- 42 Teachers
- 850 Students
- 850 Parent accounts (linked to students)
- 1 Nurse, 1 Accountant, 1 Librarian, 1 Storekeeper, 2 Drivers, 1 Security Officer

---

## 11. Glossary

| Term | Definition |
|------|-----------|
| BRS | Business Requirements Specification |
| CAT | Continuous Assessment Test |
| CMS | Content Management System |
| CPD | Continuing Professional Development |
| FDSE | Free Day Secondary Education (government programme) |
| GRN | Goods Received Note |
| HOD | Head of Department |
| IAM | Identity and Access Management |
| KNEC | Kenya National Examinations Council |
| KRA | Kenya Revenue Authority |
| LPO | Local Purchase Order |
| MFA | Multi-Factor Authentication |
| NEMIS | National Education Management Information System |
| NHIF/SHIF | National Hospital Insurance Fund / Social Health Insurance Fund |
| NSSF | National Social Security Fund |
| ODPC | Office of the Data Protection Commissioner |
| PAYE | Pay As You Earn (income tax) |
| PR | Purchase Request |
| PRN | Personal Registration Number (KNEC) |
| PWA | Progressive Web App |
| RBAC | Role-Based Access Control |
| RTO | Recovery Time Objective |
| RPO | Recovery Point Objective |
| STK | SIM Toolkit (M-Pesa push payment mechanism) |
| TSC | Teachers Service Commission |
| YTD | Year to Date |
