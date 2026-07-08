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

    if (error.response?.status === 401 && !originalRequest._retry) {
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
  recordManualPayment: (data: Record<string, unknown>) =>
    api.post("/finance/pay/manual", data),
  getSummary: (termId?: string) =>
    api.get("/finance/summary", { params: termId ? { term_id: termId } : {} }),
  getOutstandingBalances: () =>
    api.get("/finance/outstanding"),
};

export const announcementsAPI = {
  list: (params?: Record<string, string | number | undefined>) =>
    api.get("/announcements", { params }),
  create: (data: Record<string, unknown>) => api.post("/announcements", data),
};

export const academicsAPI = {
  listClasses: (params?: { level?: number; academic_year_id?: string }) =>
    api.get("/academics/classes", { params }),
  getAttendanceSheet: (classId: string, date?: string) =>
    api.get("/academics/attendance/sheet", { params: { class_id: classId, date } }),
  submitAttendance: (data: {
    class_id: string;
    session_date: string;
    session_type: "daily" | "period";
    records: Array<{ student_id: string; status: string; remarks?: string }>;
  }) => api.post("/academics/attendance", data),
  getAttendanceSummary: (classId: string, startDate?: string, endDate?: string) =>
    api.get("/academics/attendance/summary", {
      params: { class_id: classId, start_date: startDate, end_date: endDate },
    }),

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
};

export const parentAPI = {
  getChildren: () => api.get("/parent/children"),
  getChildReportCard: (studentId: string) =>
    api.get(`/parent/children/${studentId}/report-card`),
  getChildSubjectScores: (studentId: string) =>
    api.get(`/parent/children/${studentId}/subject-scores`),
  getChildFeeAccount: (studentId: string) =>
    api.get(`/parent/children/${studentId}/account`),
};
