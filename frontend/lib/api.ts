import axios, { AxiosError, InternalAxiosRequestConfig } from "axios";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1";

export const api = axios.create({
  baseURL: API_BASE,
  timeout: 30_000,
  headers: { "Content-Type": "application/json" },
  withCredentials: true, // send httpOnly cookie for refresh token
});

// Attach institution slug and access token to every request
api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("access_token");
    const slug = localStorage.getItem("institution_slug");
    if (token) config.headers["Authorization"] = `Bearer ${token}`;
    if (slug) config.headers["X-Institution-Slug"] = slug;
  }
  return config;
});

// Auto-refresh on 401
let isRefreshing = false;
let failedQueue: Array<{ resolve: (v: unknown) => void; reject: (e: unknown) => void }> = [];

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((p) => (error ? p.reject(error) : p.resolve(token)));
  failedQueue = [];
};

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };
    const isAuthEndpoint = originalRequest.url?.includes("/auth/login") || originalRequest.url?.includes("/auth/refresh");

    if (error.response?.status === 401 && !originalRequest._retry && !isAuthEndpoint) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        })
          .then((token) => {
            originalRequest.headers["Authorization"] = `Bearer ${token}`;
            return api(originalRequest);
          })
          .catch((err) => Promise.reject(err));
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const { data } = await api.post("/auth/refresh");
        const newToken = data.data?.access_token;
        if (newToken) {
          localStorage.setItem("access_token", newToken);
          api.defaults.headers.common["Authorization"] = `Bearer ${newToken}`;
          processQueue(null, newToken);
          originalRequest.headers["Authorization"] = `Bearer ${newToken}`;
          return api(originalRequest);
        }
      } catch (refreshError) {
        processQueue(refreshError, null);
        localStorage.removeItem("access_token");
        window.location.href = "/login";
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

// ─── Typed API helpers ─────────────────────────────────────────

export const authAPI = {
  login: (email: string, password: string, institutionSlug: string) =>
    api.post("/auth/login", { email, password }, {
      headers: { "X-Institution-Slug": institutionSlug },
    }),
  logout: () => api.post("/auth/logout"),
  me: () => api.get("/auth/me"),
  changePassword: (currentPassword: string, newPassword: string) =>
    api.post("/auth/change-password", { current_password: currentPassword, new_password: newPassword }),
};

export const studentsAPI = {
  list: (params?: Record<string, string | number | undefined>) =>
    api.get("/students", { params }),
  get: (id: string) => api.get(`/students/${id}`),
  getTimeline: (id: string) => api.get(`/students/${id}/timeline`),
  getParents: (id: string) => api.get(`/students/${id}/parents`),
};

export const admissionsAPI = {
  list: (params?: Record<string, string | number | undefined>) =>
    api.get("/admissions", { params }),
  get: (id: string) => api.get(`/admissions/${id}`),
  create: (data: Record<string, unknown>) => api.post("/admissions", data),
  review: (id: string, action: "approve" | "reject", notes?: string) =>
    api.post(`/admissions/${id}/review`, { action, notes }),
  enroll: (id: string, data: Record<string, unknown>) =>
    api.post(`/admissions/${id}/enroll`, data),
};

export const financeAPI = {
  getFeeAccount: (studentId: string) =>
    api.get(`/finance/students/${studentId}/account`),
  getPaymentHistory: (studentId: string) =>
    api.get(`/finance/students/${studentId}/payments`),
  initiateStkPush: (data: { student_id: string; amount: number; phone: string }) =>
    api.post("/finance/pay/stk-push", data),
  recordManualPayment: (data: {
    student_id: string;
    invoice_id?: string;
    amount: number;
    payment_method: "cash" | "bank_transfer" | "cheque" | "waiver";
    bank_reference?: string;
    cheque_number?: string;
    notes?: string;
    is_record_only?: boolean; // new
  }) => 
    api.post("/finance/pay/manual", data),
  getSummary: (termId?: string) =>
    api.get("/finance/summary", { params: termId ? { term_id: termId } : {} }),
  getOutstandingBalances: () =>
    api.get("/finance/outstanding"),

  // ── Fee billing (fee-structures / invoices) ──────────────────
  // Mirrors backend/internal/finance/handler.go: CreateFeeStructure,
  // ListFeeStructures, AddFeeComponent, GenerateInvoicesForClass.
  createFeeStructure: (data: { academic_year_id: string; class_id?: string; name: string }) =>
    api.post("/finance/fee-structures", data),
  listFeeStructures: (academicYearId?: string) =>
    api.get("/finance/fee-structures", { params: academicYearId ? { academic_year_id: academicYearId } : {} }),
  addFeeComponent: (
    feeStructureId: string,
    data: { name: string; amount: number; term_number?: number; is_mandatory: boolean; due_date?: string }
  ) => api.post(`/finance/fee-structures/${feeStructureId}/components`, data),
  generateInvoicesForClass: (data: {
    fee_structure_id: string;
    class_id: string;
    term_id: string;
    term_number: number;
  }) => api.post("/finance/invoices/generate-class", data),

  
 
  allocatePayment: (paymentId: string, data: { invoice_id: string; amount?: number }) =>
    api.post(`/finance/payments/${paymentId}/allocate`, data),
 
  deallocatePayment: (allocationId: string, reason?: string) =>
    api.post(`/finance/allocations/${allocationId}/deallocate`, { reason }),
};

export const announcementsAPI = {
  list: (params?: Record<string, string | number | undefined>) =>
    api.get("/announcements", { params }),
  create: (data: Record<string, unknown>) => api.post("/announcements", data),
};

export const academicsAPI = {
  listClasses: (params?: { level?: number; academic_year_id?: string }) =>
    api.get("/academics/classes", { params }),
  getAttendanceSheet: (classId: string, date?: string, sessionType?: string, periodId?: string) =>
    api.get("/academics/attendance/sheet", {
      params: { class_id: classId, date, session_type: sessionType, period_id: periodId },
    }),
  submitAttendance: (data: {
  class_id: string;
  session_date: string;
  session_type: "daily" | "period";
  period_id?: string;
  records: Array<{ student_id: string; status: string; remarks?: string }>;
}) => api.post("/academics/attendance", data),
  getAttendanceSummary: (classId: string, startDate?: string, endDate?: string) =>
    api.get("/academics/attendance/summary", {
      params: { class_id: classId, start_date: startDate, end_date: endDate },
    }),
  // Institution-wide attendance snapshot for the principal dashboard —
  // distinct from getAttendanceSummary, which is always per-class.
  // Defaults to today; pass a date to check historical/seeded data.
  getAttendanceToday: (date?: string) =>
    api.get("/academics/attendance/today", { params: date ? { date } : {} }),

  listSubjects: () => api.get("/academics/subjects"),
  getGradeComponents: (classId: string, subjectId: string, termId: string) =>
    api.get("/academics/grades/components", {
      params: { class_id: classId, subject_id: subjectId, term_id: termId },
    }),
  getGradebook: (componentId: string) =>
    api.get("/academics/grades/entries", { params: { component_id: componentId } }),
  saveGrades: (data: {
    component_id: string;
    entries: Array<{ student_id: string; marks: number | null; is_absent: boolean }>;
  }) => api.post("/academics/grades/entries", data),
  createGradeComponent: (data: {
    class_id: string;
    subject_id: string;
    term_id: string;
    name: string;
    weight: number;
    max_marks: number;
  }) => api.post("/academics/grades/components", data),
  generateReportCard: (data: { student_id: string; class_id: string; term_id: string }) =>
    api.post("/academics/report-cards/generate", data),
  generateReportCardsForClass: (data: { class_id: string; term_id: string }) =>
    api.post("/academics/report-cards/generate-class", data),
  publishReportCard: (data: { student_id: string; term_id: string }) =>
    api.post("/academics/report-cards/publish", data),
  unpublishReportCard: (data: { student_id: string; term_id: string }) =>
    api.post("/academics/report-cards/unpublish", data),
  createClass: (data: {
    academic_year_id: string;
    name: string;
    stream?: string;
    level: number;
    capacity: number;
    class_teacher_id?: string;
  }) => api.post("/academics/classes", data),
  assignClassTeacher: (classId: string, classTeacherId: string) =>
    api.post(`/academics/classes/${classId}/assign-teacher`, { class_teacher_id: classTeacherId }),
  listTeachers: () => api.get("/academics/teachers"),
  listAcademicYears: () => api.get("/academics/academic-years"),

  listTerms: () => api.get("/academics/terms"),
  getMyTimetable: (termId?: string) =>
    api.get("/academics/my-timetable", { params: termId ? { term_id: termId } : {} }),

  // frontend/lib/api.ts — add to academicsAPI
createTeachingAssignment: (data: {
  class_id: string; subject_id: string; teacher_id: string; term_id: string;
}) => api.post("/academics/teaching-assignments", data),

 // frontend/lib/api.ts — additions to academicsAPI
  listTeachingAssignmentsForClass: (classId: string, termId: string) =>
    api.get(`/academics/classes/${classId}/teaching-assignments`, { params: { term_id: termId } }),
  createTimetablePeriod: (data: {
    class_id: string; term_id: string; teaching_assignment_id: string;
    day_of_week: number; period_number: number; start_time: string; end_time: string; room?: string;
  }) => api.post("/academics/timetable-periods", data),
  // frontend/lib/api.ts — one more addition to academicsAPI
getClassTimetable: (classId: string, termId?: string) =>
  api.get(`/academics/classes/${classId}/timetable`, { params: termId ? { term_id: termId } : {} }),
  // Timetable engine: bell-schedule configuration (profiles + slots),
  // lesson requirements, and the admin teacher-timetable view.
  listScheduleProfiles: () => api.get("/academics/schedule-profiles"),
  createScheduleProfile: (data: { name: string; term_id?: string; is_default?: boolean }) =>
    api.post("/academics/schedule-profiles", data),
  listScheduleSlots: (profileId: string) =>
    api.get(`/academics/schedule-profiles/${profileId}/slots`),
  createScheduleSlot: (profileId: string, data: {
    day_of_week: number; sequence: number; start_time: string; end_time: string;
    slot_type: string; is_instructional?: boolean; is_active?: boolean;
  }) => api.post(`/academics/schedule-profiles/${profileId}/slots`, data),
  updateScheduleSlot: (slotId: string, data: {
    start_time?: string; end_time?: string; slot_type?: string;
    is_instructional?: boolean; is_active?: boolean;
  }) => api.patch(`/academics/schedule-slots/${slotId}`, data),
  bulkCreateScheduleSlots: (profileId: string, data: {
    days: number[]; sequence: number; start_time: string; end_time: string;
    slot_type: string; is_instructional?: boolean; is_active?: boolean;
  }) => api.post(`/academics/schedule-profiles/${profileId}/slots/bulk`, data),
  copyScheduleDay: (profileId: string, data: {
    from_day: number; to_days: number[]; overwrite?: boolean;
  }) => api.post(`/academics/schedule-profiles/${profileId}/slots/copy`, data),
  propagateScheduleSlot: (slotId: string, data: {
    start_time?: string; end_time?: string; slot_type?: string;
    is_instructional?: boolean; is_active?: boolean; target_days: number[];
  }) => api.post(`/academics/schedule-slots/${slotId}/propagate`, data),
  deleteScheduleSlot: (slotId: string) =>
    api.delete(`/academics/schedule-slots/${slotId}`),
  getTeacherTimetable: (teacherId: string, termId?: string) =>
    api.get(`/academics/teachers/${teacherId}/timetable`, { params: termId ? { term_id: termId } : {} }),
  listLessonRequirements: (classId: string, termId: string) =>
    api.get(`/academics/classes/${classId}/lesson-requirements`, { params: { term_id: termId } }),
  upsertLessonRequirement: (classId: string, data: {
    term_id: string; academic_offering_id: string; sessions_per_week: number;
    allow_single: boolean; allow_double: boolean; allow_triple: boolean;
    spread_across_days: boolean; max_per_day?: number | null;
  }) => api.put(`/academics/classes/${classId}/lesson-requirements`, data),
  deleteLessonRequirement: (requirementId: string) =>
    api.delete(`/academics/lesson-requirements/${requirementId}`),
};


export const examsAPI = {
  listSessions: () => api.get("/exams/sessions"),
  getSession: (id: string) => api.get(`/exams/sessions/${id}`),
  createSession: (data: {
    academic_year_id?: string; term_id?: string; name: string;
    exam_type: string; start_date: string; end_date: string;
  }) => api.post("/exams/sessions", data),
  createPaper: (sessionId: string, data: {
    academic_offering_id?: string; paper_name: string; paper_code?: string;
    paper_number?: string; duration_minutes: number; max_marks?: number;
    instructions?: string; is_shared?: boolean;
  }) => api.post(`/exams/sessions/${sessionId}/papers`, data),
  listPapers: (sessionId: string) => api.get(`/exams/sessions/${sessionId}/papers`),
  addPaperGroup: (paperId: string, data: {
    class_id?: string; label?: string; candidate_count?: number;
  }) => api.post(`/exams/papers/${paperId}/groups`, data),
  createSitting: (sessionId: string, data: {
    exam_paper_id: string; exam_date: string; start_time: string; end_time: string;
    room_name?: string; room_capacity?: number; candidate_count?: number; seating_note?: string;
  }) => api.post(`/exams/sessions/${sessionId}/sittings`, data),
  updateSitting: (sittingId: string, data: Record<string, unknown>) =>
    api.patch(`/exams/sittings/${sittingId}`, data),
  deleteSitting: (sittingId: string) => api.delete(`/exams/sittings/${sittingId}`),
  assignInvigilator: (sittingId: string, data: { staff_id: string; role: string; notes?: string }) =>
    api.post(`/exams/sittings/${sittingId}/invigilators`, data),
  addArrangement: (sittingId: string, data: {
    student_id?: string; class_id?: string; arrangement_type: string;
    details?: string; extra_minutes?: number; room_name?: string;
  }) => api.post(`/exams/sittings/${sittingId}/arrangements`, data),
  validate: (sessionId: string) => api.post(`/exams/sessions/${sessionId}/validate`),
  publish: (sessionId: string) => api.post(`/exams/sessions/${sessionId}/publish`),
  reopen: (sessionId: string) => api.post(`/exams/sessions/${sessionId}/reopen`),
  generate: (sessionId: string, data: {
    start_date: string; end_date: string; daily_start_time: string;
    daily_end_time: string; gap_minutes: number; max_sittings_per_day?: number;
  }) => api.post(`/exams/sessions/${sessionId}/generate`, data),
  master: (sessionId: string) => api.get(`/exams/sessions/${sessionId}/timetable`),
  publishedMaster: (sessionId: string) => api.get(`/exams/sessions/${sessionId}/timetable/published`),
  candidateView: (sessionId: string, classId: string) =>
    api.get(`/exams/sessions/${sessionId}/timetable/candidate`, { params: { class_id: classId } }),
  paperView: (sessionId: string, paperId: string) =>
    api.get(`/exams/sessions/${sessionId}/timetable/paper`, { params: { paper_id: paperId } }),
  roomView: (sessionId: string, room: string) =>
    api.get(`/exams/sessions/${sessionId}/timetable/room`, { params: { room } }),
  invigilatorView: (sessionId: string, staffId: string) =>
    api.get(`/exams/sessions/${sessionId}/timetable/invigilator`, { params: { staff_id: staffId } }),
};

export const cbcAPI = {
  listLearningAreas: () => api.get("/cbc/learning-areas"),
  createLearningArea: (data: { name: string; code?: string; is_compulsory: boolean }) =>
    api.post("/cbc/learning-areas", data),
  getComponents: (classId: string, learningAreaId: string, termId: string) =>
    api.get("/cbc/components", { params: { class_id: classId, learning_area_id: learningAreaId, term_id: termId } }),
  createComponent: (data: {
    class_id: string; learning_area_id: string; term_id: string;
    name: string; weight: number; max_marks: number;
  }) => api.post("/cbc/components", data),
  getEntries: (componentId: string) =>
    api.get("/cbc/entries", { params: { component_id: componentId } }),
  saveEntries: (data: {
    component_id: string;
    entries: Array<{ student_id: string; marks: number | null; is_absent: boolean }>;
  }) => api.post("/cbc/entries", data),
  createTeacherAssignment: (data: {
    class_id: string; learning_area_id: string; teacher_id: string; term_id: string;
  }) => api.post("/cbc/teacher-assignments", data),
  generateReportCardsForClass: (data: { class_id: string; term_id: string }) =>
    api.post("/cbc/report-cards/generate-class", data),
  publishReportCard: (data: { student_id: string; term_id: string }) =>
    api.post("/cbc/report-cards/publish", data),
  unpublishReportCard: (data: { student_id: string; term_id: string }) =>
    api.post("/cbc/report-cards/unpublish", data),
}

export const parentAPI = {
  getChildren: () => api.get("/parent/children"),
  getChildReportCard: (studentId: string) =>
    api.get(`/parent/children/${studentId}/report-card`),
  getChildSubjectScores: (studentId: string) =>
    api.get(`/parent/children/${studentId}/subject-scores`),
  getChildFeeAccount: (studentId: string) =>
    api.get(`/parent/children/${studentId}/account`),
};

export const institutionAPI = {
  getContext: () => api.get("/institution/context"),
};

