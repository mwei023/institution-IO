"use client";
import React, { useEffect, useState, useCallback } from "react";
import { Plus, AlertTriangle, Loader2, Calendar } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { academicsAPI } from "@/lib/api";

interface ClassOption {
  id: string;
  name: string;
  stream: string | null;
  level: number;
}

interface TermOption {
  id: string;
  name: string;
  is_current?: boolean;
}

interface TeachingAssignmentOption {
  id: string;
  offering_name: string;
  offering_type: "subject" | "learning_area";
  teacher_id: string;
  teacher_name: string;
}

interface TimetablePeriodRow {
  id: string;
  day_of_week: number;
  period_number: number;
  start_time: string;
  end_time: string;
  room: string | null;
  offering_name: string | null;
  teacher_name: string | null;
}

const DAYS = [
  { value: 1, label: "Monday" },
  { value: 2, label: "Tuesday" },
  { value: 3, label: "Wednesday" },
  { value: 4, label: "Thursday" },
  { value: 5, label: "Friday" },
];
const PERIODS = Array.from({ length: 10 }, (_, i) => i + 1);

export default function TimetableAdminPage() {
  const [classes, setClasses] = useState<ClassOption[]>([]);
  const [terms, setTerms] = useState<TermOption[]>([]);
  const [classId, setClassId] = useState("");
  const [termId, setTermId] = useState("");

  const [assignments, setAssignments] = useState<TeachingAssignmentOption[]>([]);
  const [periods, setPeriods] = useState<TimetablePeriodRow[]>([]);

  const [loading, setLoading] = useState(true);
  const [loadingSchedule, setLoadingSchedule] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Create-period form state
  const [showCreate, setShowCreate] = useState(false);
  const [teachingAssignmentId, setTeachingAssignmentId] = useState("");
  const [dayOfWeek, setDayOfWeek] = useState("");
  const [periodNumber, setPeriodNumber] = useState("");
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
  const [room, setRoom] = useState("");
  const [creating, setCreating] = useState(false);

  const loadClassesAndTerms = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [classesRes, termsRes] = await Promise.all([
        academicsAPI.listClasses(),
        academicsAPI.listTerms(),
      ]);
      const classList: ClassOption[] = classesRes.data?.data ?? [];
      const termList: TermOption[] = termsRes.data?.data ?? [];
      setClasses(classList);
      setTerms(termList);
      const current = termList.find((t) => t.is_current);
      if (current) setTermId(current.id);
      if (classList.length > 0) setClassId(classList[0].id);
    } catch {
      setError("Couldn't load classes or terms. Try refreshing.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadClassesAndTerms();
  }, [loadClassesAndTerms]);

  const loadSchedule = useCallback(async () => {
    if (!classId || !termId) return;
    setLoadingSchedule(true);
    setError(null);
    try {
      const [assignmentsRes, timetableRes] = await Promise.all([
        academicsAPI.listTeachingAssignmentsForClass(classId, termId),
        academicsAPI.getClassTimetable(classId, termId),
      ]);
      setAssignments(assignmentsRes.data?.data ?? []);
      setPeriods(timetableRes.data?.data ?? []);
    } catch {
      setError("Couldn't load the timetable for this class.");
    } finally {
      setLoadingSchedule(false);
    }
  }, [classId, termId]);

  useEffect(() => {
    loadSchedule();
  }, [loadSchedule]);

  const handleCreate = async () => {
    if (!teachingAssignmentId || !dayOfWeek || !periodNumber || !startTime || !endTime) return;
    setCreating(true);
    setError(null);
    try {
      await academicsAPI.createTimetablePeriod({
        class_id: classId,
        term_id: termId,
        teaching_assignment_id: teachingAssignmentId,
        day_of_week: parseInt(dayOfWeek, 10),
        period_number: parseInt(periodNumber, 10),
        start_time: startTime,
        end_time: endTime,
        room: room || undefined,
      });
      setTeachingAssignmentId("");
      setDayOfWeek("");
      setPeriodNumber("");
      setStartTime("");
      setEndTime("");
      setRoom("");
      setShowCreate(false);
      await loadSchedule();
    } catch (err: any) {
      // Slot/teacher conflicts come back as 409s with a real message from
      // the backend (ErrSlotAlreadyBooked / ErrTeacherDoubleBooked) —
      // surfaced as-is rather than a generic failure, since it tells the
      // person exactly what to fix.
      setError(err?.response?.data?.error?.message ?? "Couldn't schedule this period.");
    } finally {
      setCreating(false);
    }
  };

  const dayLabel = (d: number) => DAYS.find((x) => x.value === d)?.label ?? `Day ${d}`;

  return (
    <DashboardLayout title="Timetable">
      <div className="space-y-6">
        <div className="page-header flex items-center justify-between">
          <div>
            <h1 className="page-title">Timetable</h1>
            <p className="page-subtitle">Schedule teaching assignments into class periods</p>
          </div>
          <Button onClick={() => setShowCreate((v) => !v)} disabled={!classId || !termId}>
            <Plus className="h-4 w-4 mr-2" /> Schedule a period
          </Button>
        </div>

        {error && (
          <p className="text-sm text-error flex items-center gap-1.5">
            <AlertTriangle className="h-4 w-4" /> {error}
          </p>
        )}

        <Card>
          <CardContent className="pt-6">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-neutral-500 mb-1.5">Class</label>
                <select
                  value={classId}
                  onChange={(e) => setClassId(e.target.value)}
                  className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                >
                  {loading && <option>Loading&hellip;</option>}
                  {classes.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name} {c.stream ?? ""}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-neutral-500 mb-1.5">Term</label>
                <select
                  value={termId}
                  onChange={(e) => setTermId(e.target.value)}
                  className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                >
                  {loading && <option>Loading&hellip;</option>}
                  {terms.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name} {t.is_current ? "(current)" : ""}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {classId && termId && assignments.length === 0 && !loadingSchedule && (
              <p className="mt-3 text-xs text-warning flex items-center gap-1.5">
                <AlertTriangle className="h-3.5 w-3.5" />
                This class has no teaching assignments for this term yet — assign a teacher to a
                subject or learning area before scheduling a period.
              </p>
            )}
          </CardContent>
        </Card>

        {showCreate && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Schedule a period</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                <div className="col-span-2 sm:col-span-3">
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Lesson</label>
                  <select
                    value={teachingAssignmentId}
                    onChange={(e) => setTeachingAssignmentId(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  >
                    <option value="">Choose&hellip;</option>
                    {assignments.map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.offering_name} — {a.teacher_name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Day</label>
                  <select
                    value={dayOfWeek}
                    onChange={(e) => setDayOfWeek(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  >
                    <option value="">Choose&hellip;</option>
                    {DAYS.map((d) => (
                      <option key={d.value} value={d.value}>{d.label}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Period</label>
                  <select
                    value={periodNumber}
                    onChange={(e) => setPeriodNumber(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  >
                    <option value="">Choose&hellip;</option>
                    {PERIODS.map((p) => (
                      <option key={p} value={p}>Period {p}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Room</label>
                  <input
                    value={room}
                    onChange={(e) => setRoom(e.target.value)}
                    placeholder="Optional"
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Start time</label>
                  <input
                    type="time"
                    value={startTime}
                    onChange={(e) => setStartTime(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">End time</label>
                  <input
                    type="time"
                    value={endTime}
                    onChange={(e) => setEndTime(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
              </div>
              <div className="mt-4">
                <Button
                  onClick={handleCreate}
                  disabled={creating || !teachingAssignmentId || !dayOfWeek || !periodNumber || !startTime || !endTime}
                >
                  {creating ? <><Loader2 className="h-4 w-4 mr-2 animate-spin" /> Scheduling&hellip;</> : "Schedule period"}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader className="pb-3">
            <CardTitle>Current schedule</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {loadingSchedule ? (
              <div className="p-8 text-center text-sm text-neutral-400">Loading schedule&hellip;</div>
            ) : periods.length === 0 ? (
              <div className="p-8 text-center text-sm text-neutral-400">
                No periods scheduled for this class/term yet.
              </div>
            ) : (
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-100">
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Day</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Period</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Time</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Lesson</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Room</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-50">
                  {periods
                    .slice()
                    .sort((a, b) => a.day_of_week - b.day_of_week || a.period_number - b.period_number)
                    .map((p) => (
                      <tr key={p.id} className="hover:bg-neutral-50">
                        <td className="px-6 py-3.5 text-sm text-neutral-800">{dayLabel(p.day_of_week)}</td>
                        <td className="px-6 py-3.5 text-sm text-neutral-600">{p.period_number}</td>
                        <td className="px-6 py-3.5 text-sm text-neutral-600">{p.start_time}–{p.end_time}</td>
                        <td className="px-6 py-3.5 text-sm">
                          {p.offering_name ? (
                            <span className="text-neutral-800">
                              {p.offering_name} — {p.teacher_name}
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-warning">
                              <AlertTriangle className="h-3.5 w-3.5" /> Not linked to an assignment
                            </span>
                          )}
                        </td>
                        <td className="px-6 py-3.5 text-sm text-neutral-600">{p.room ?? "—"}</td>
                      </tr>
                    ))}
                </tbody>
              </table>
            )}
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}