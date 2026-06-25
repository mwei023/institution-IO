"use client";
import React, { useEffect, useState, useCallback } from "react";
import { CheckCircle2, XCircle, Clock3, FileWarning, AlertTriangle, Loader2 } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { academicsAPI } from "@/lib/api";
import { formatDate } from "@/lib/utils";

interface ClassOption {
  id: string;
  name: string;
  stream: string | null;
  level: number;
}

interface AttendanceRecordRow {
  id: string;
  student_id: string;
  name: string;
  status: string;
  remarks?: string | null;
}

interface AttendanceSheetResponse {
  id: string;
  class_id: string;
  class_name: string;
  session_date: string;
  session_type: string;
  status: string;
  records: AttendanceRecordRow[];
}

type AttendanceStatus = "present" | "absent" | "late" | "excused";

const STATUS_OPTIONS: { value: AttendanceStatus; label: string; icon: typeof CheckCircle2 }[] = [
  { value: "present", label: "Present", icon: CheckCircle2 },
  { value: "absent", label: "Absent", icon: XCircle },
  { value: "late", label: "Late", icon: Clock3 },
  { value: "excused", label: "Excused", icon: FileWarning },
];

const STATUS_STYLES: Record<AttendanceStatus, { active: string; inactive: string }> = {
  present: { active: "bg-success-light text-success-dark border-success/30", inactive: "text-neutral-400 border-neutral-200 hover:border-success/40 hover:text-success-dark" },
  absent: { active: "bg-error-light text-error-dark border-error/30", inactive: "text-neutral-400 border-neutral-200 hover:border-error/40 hover:text-error-dark" },
  late: { active: "bg-warning-light text-warning-dark border-warning/30", inactive: "text-neutral-400 border-neutral-200 hover:border-warning/40 hover:text-warning-dark" },
  excused: { active: "bg-neutral-100 text-neutral-600 border-neutral-300", inactive: "text-neutral-400 border-neutral-200 hover:border-neutral-400 hover:text-neutral-600" },
};

function todayISO(): string {
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

export default function TeacherAttendancePage() {
  const [classes, setClasses] = useState<ClassOption[] | null>(null);
  const [classesError, setClassesError] = useState(false);
  const [selectedClassId, setSelectedClassId] = useState<string>("");
  const [sessionDate, setSessionDate] = useState<string>(todayISO());

  const [sheet, setSheet] = useState<AttendanceSheetResponse | null>(null);
  const [sheetLoading, setSheetLoading] = useState(false);
  const [sheetError, setSheetError] = useState<string | null>(null);

  // Local editable status per student_id, seeded from the sheet response
  const [statusMap, setStatusMap] = useState<Record<string, AttendanceStatus>>({});
  const [remarksMap, setRemarksMap] = useState<Record<string, string>>({});

  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState(false);

  // Load real classes for the dropdown
  useEffect(() => {
    academicsAPI
      .listClasses()
      .then((res) => setClasses(res.data?.data ?? []))
      .catch(() => setClassesError(true));
  }, []);

  const loadSheet = useCallback((classId: string, date: string) => {
    if (!classId) return;
    setSheetLoading(true);
    setSheetError(null);
    setSubmitSuccess(false);
    setSubmitError(null);
    academicsAPI
      .getAttendanceSheet(classId, date)
      .then((res) => {
        const data: AttendanceSheetResponse = res.data?.data;
        setSheet(data);
        const sMap: Record<string, AttendanceStatus> = {};
        const rMap: Record<string, string> = {};
        (data.records ?? []).forEach((r) => {
          sMap[r.student_id] = (r.status as AttendanceStatus) || "present";
          rMap[r.student_id] = r.remarks ?? "";
        });
        setStatusMap(sMap);
        setRemarksMap(rMap);
      })
      .catch((err) => {
        setSheet(null);
        setSheetError(
          err?.response?.data?.error?.message ?? "Could not load the attendance sheet for this class."
        );
      })
      .finally(() => setSheetLoading(false));
  }, []);

  useEffect(() => {
    if (selectedClassId) {
      loadSheet(selectedClassId, sessionDate);
    } else {
      setSheet(null);
    }
  }, [selectedClassId, sessionDate, loadSheet]);

  const setStatus = (studentId: string, status: AttendanceStatus) => {
    setStatusMap((prev) => ({ ...prev, [studentId]: status }));
  };

  const setRemark = (studentId: string, value: string) => {
    setRemarksMap((prev) => ({ ...prev, [studentId]: value }));
  };

  const markAllPresent = () => {
    if (!sheet) return;
    const sMap: Record<string, AttendanceStatus> = {};
    sheet.records.forEach((r) => { sMap[r.student_id] = "present"; });
    setStatusMap(sMap);
  };

  const handleSubmit = async () => {
    if (!sheet) return;
    setSubmitting(true);
    setSubmitError(null);
    setSubmitSuccess(false);
    try {
      await academicsAPI.submitAttendance({
        class_id: sheet.class_id,
        session_date: sessionDate,
        session_type: "daily",
        records: sheet.records.map((r) => ({
          student_id: r.student_id,
          status: statusMap[r.student_id] ?? "present",
          remarks: remarksMap[r.student_id] ?? "",
        })),
      });
      setSubmitSuccess(true);
      // Refresh the sheet so the locked/submitted state reflects reality
      loadSheet(sheet.class_id, sessionDate);
    } catch (err: any) {
      setSubmitError(
        err?.response?.data?.error?.message ?? "Could not submit attendance. Please try again."
      );
    } finally {
      setSubmitting(false);
    }
  };

  const isLocked = sheet?.status === "submitted" || sheet?.status === "locked";
  const absentCount = Object.values(statusMap).filter((s) => s === "absent").length;
  const presentCount = Object.values(statusMap).filter((s) => s === "present").length;
  const lateCount = Object.values(statusMap).filter((s) => s === "late").length;
  const excusedCount = Object.values(statusMap).filter((s) => s === "excused").length;

  return (
    <DashboardLayout title="Take Attendance">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Take Attendance</h1>
            <p className="page-subtitle">Mark daily attendance for your class</p>
          </div>
        </div>

        {/* Class + date selection */}
        <Card>
          <CardContent className="py-4 flex flex-wrap items-end gap-4">
            <div className="flex-1 min-w-[220px]">
              <label className="block text-xs font-medium text-neutral-500 uppercase tracking-wide mb-1.5">
                Class
              </label>
              {classesError ? (
                <p className="text-sm text-error flex items-center gap-1.5">
                  <AlertTriangle className="h-3.5 w-3.5" /> Could not load classes
                </p>
              ) : (
                <select
                  className="w-full h-9 rounded-md border border-neutral-200 bg-white px-3 text-sm text-neutral-800 focus:outline-none focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                  value={selectedClassId}
                  onChange={(e) => setSelectedClassId(e.target.value)}
                >
                  <option value="">Select a class…</option>
                  {(classes ?? []).map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}{c.stream ? ` ${c.stream}` : ""}
                    </option>
                  ))}
                </select>
              )}
            </div>

            <div className="min-w-[180px]">
              <label className="block text-xs font-medium text-neutral-500 uppercase tracking-wide mb-1.5">
                Date
              </label>
              <Input
                type="date"
                value={sessionDate}
                max={todayISO()}
                onChange={(e) => setSessionDate(e.target.value)}
                className="h-9 text-sm"
              />
            </div>

            {sheet && (
              <div className="text-sm text-neutral-500 pb-2">
                {sheet.class_name} · {formatDate(sessionDate)}
              </div>
            )}
          </CardContent>
        </Card>

        {!selectedClassId && (
          <Card>
            <CardContent className="py-12 text-center text-sm text-neutral-500">
              Select a class above to view its attendance roster.
            </CardContent>
          </Card>
        )}

        {selectedClassId && sheetLoading && (
          <Card>
            <CardContent className="py-12 text-center text-sm text-neutral-500 flex items-center justify-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin" /> Loading roster…
            </CardContent>
          </Card>
        )}

        {selectedClassId && !sheetLoading && sheetError && (
          <Card className="border-error/30 bg-error-light/30">
            <CardContent className="py-6 flex items-center gap-2 text-error text-sm">
              <AlertTriangle className="h-4 w-4" /> {sheetError}
            </CardContent>
          </Card>
        )}

        {sheet && !sheetLoading && !sheetError && (
          <>
            {sheet.records.length === 0 ? (
              <Card>
                <CardContent className="py-12 text-center text-sm text-neutral-500">
                  No active students are enrolled in this class.
                </CardContent>
              </Card>
            ) : (
              <>
                {/* Summary strip */}
                <div className="grid grid-cols-4 gap-3">
                  <div className="rounded-lg p-3 bg-success-light text-success-dark text-center">
                    <p className="text-xl font-semibold">{presentCount}</p>
                    <p className="text-xs font-medium mt-0.5">Present</p>
                  </div>
                  <div className="rounded-lg p-3 bg-error-light text-error-dark text-center">
                    <p className="text-xl font-semibold">{absentCount}</p>
                    <p className="text-xs font-medium mt-0.5">Absent</p>
                  </div>
                  <div className="rounded-lg p-3 bg-warning-light text-warning-dark text-center">
                    <p className="text-xl font-semibold">{lateCount}</p>
                    <p className="text-xs font-medium mt-0.5">Late</p>
                  </div>
                  <div className="rounded-lg p-3 bg-neutral-100 text-neutral-600 text-center">
                    <p className="text-xl font-semibold">{excusedCount}</p>
                    <p className="text-xs font-medium mt-0.5">Excused</p>
                  </div>
                </div>

                <Card>
                  <CardHeader className="flex-row items-center justify-between pb-3">
                    <CardTitle>{sheet.class_name} Roster</CardTitle>
                    {!isLocked && (
                      <Button size="sm" variant="outline" onClick={markAllPresent}>
                        Mark all present
                      </Button>
                    )}
                  </CardHeader>
                  <CardContent className="p-0">
                    {isLocked && (
                      <div className="px-6 py-3 bg-neutral-50 border-b border-neutral-100 text-xs text-neutral-500">
                        This session has already been submitted and is locked. Showing the saved record.
                      </div>
                    )}
                    <div className="divide-y divide-neutral-50">
                      {sheet.records.map((r) => {
                        const current = statusMap[r.student_id] ?? "present";
                        return (
                          <div key={r.student_id} className="px-6 py-3.5 flex flex-wrap items-center gap-3">
                            <div className="flex-1 min-w-[160px]">
                              <p className="text-sm font-medium text-neutral-800">{r.name}</p>
                            </div>

                            <div className="flex gap-1.5">
                              {STATUS_OPTIONS.map((opt) => {
                                const Icon = opt.icon;
                                const active = current === opt.value;
                                const styles = STATUS_STYLES[opt.value];
                                return (
                                  <button
                                    key={opt.value}
                                    type="button"
                                    disabled={isLocked}
                                    onClick={() => setStatus(r.student_id, opt.value)}
                                    className={`inline-flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${
                                      active ? styles.active : styles.inactive
                                    }`}
                                  >
                                    <Icon className="h-3.5 w-3.5" />
                                    {opt.label}
                                  </button>
                                );
                              })}
                            </div>

                            {(current === "late" || current === "excused") && (
                              <Input
                                placeholder={current === "late" ? "e.g. Arrived 8:15am" : "Reason for excusal"}
                                value={remarksMap[r.student_id] ?? ""}
                                onChange={(e) => setRemark(r.student_id, e.target.value)}
                                disabled={isLocked}
                                className="h-8 text-xs w-48"
                              />
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </CardContent>
                </Card>

                {submitError && (
                  <Card className="border-error/30 bg-error-light/30">
                    <CardContent className="py-3 flex items-center gap-2 text-error text-sm">
                      <AlertTriangle className="h-4 w-4" /> {submitError}
                    </CardContent>
                  </Card>
                )}

                {submitSuccess && (
                  <Card className="border-success/30 bg-success-light/30">
                    <CardContent className="py-3 flex items-center gap-2 text-success-dark text-sm">
                      <CheckCircle2 className="h-4 w-4" /> Attendance submitted. Parents of absent students have been notified.
                    </CardContent>
                  </Card>
                )}

                {!isLocked && (
                  <div className="flex justify-end">
                    <Button onClick={handleSubmit} disabled={submitting}>
                      {submitting ? (
                        <span className="inline-flex items-center gap-2">
                          <Loader2 className="h-4 w-4 animate-spin" /> Submitting…
                        </span>
                      ) : (
                        "Submit Attendance"
                      )}
                    </Button>
                  </div>
                )}
              </>
            )}
          </>
        )}
      </div>
    </DashboardLayout>
  );
}