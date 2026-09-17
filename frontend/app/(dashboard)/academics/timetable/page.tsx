"use client";
import React, { useEffect, useState, useCallback } from "react";
import { Plus, AlertTriangle, Loader2, Calendar, Trash2 } from "lucide-react";
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

interface TeacherOption {
  id: string;
  name?: string;
  first_name?: string;
  last_name?: string;
}

interface TeachingAssignmentOption {
  id: string;
  academic_offering_id: string;
  offering_name: string;
  offering_type: "subject" | "learning_area";
  teacher_id: string;
  teacher_name: string;
}

interface TimetablePeriodRow {
  id: string;
  class_id: string;
  day_of_week: number;
  period_number: number;
  start_time: string;
  end_time: string;
  room: string | null;
  teaching_assignment_id: string | null;
  offering_name: string | null;
  teacher_name: string | null;
}

interface ScheduleSlot {
  id: string;
  day_of_week: number;
  sequence: number;
  start_time: string;
  end_time: string;
  slot_type: string;
  is_instructional: boolean;
  is_active: boolean;
}

interface LessonRequirement {
  id: string;
  class_id: string;
  academic_offering_id: string;
  offering_name: string;
  offering_type: string;
  term_id: string;
  sessions_per_week: number;
  allow_single: boolean;
  allow_double: boolean;
  allow_triple: boolean;
  spread_across_days: boolean;
  max_per_day: number | null;
}

interface RequirementDraft {
  sessions: string;
  single: boolean;
  double: boolean;
  triple: boolean;
  spread: boolean;
  maxPerDay: string;
  existingId: string | null;
}

const DAYS = [
  { value: 1, label: "Mon" },
  { value: 2, label: "Tue" },
  { value: 3, label: "Wed" },
  { value: 4, label: "Thu" },
  { value: 5, label: "Fri" },
  { value: 6, label: "Sat" },
  { value: 7, label: "Sun" },
];

const dayLabel = (d: number) => DAYS.find((x) => x.value === d)?.label ?? `D${d}`;
const teacherDisplay = (t: TeacherOption) =>
  t.name ?? ([t.first_name, t.last_name].filter(Boolean).join(" ") || t.id);

export default function TimetableAdminPage() {
  const [view, setView] = useState<"class" | "teacher">("class");
  const [classes, setClasses] = useState<ClassOption[]>([]);
  const [terms, setTerms] = useState<TermOption[]>([]);
  const [teachers, setTeachers] = useState<TeacherOption[]>([]);
  const [classId, setClassId] = useState("");
  const [termId, setTermId] = useState("");
  const [teacherId, setTeacherId] = useState("");

  const [slots, setSlots] = useState<ScheduleSlot[]>([]);
  const [assignments, setAssignments] = useState<TeachingAssignmentOption[]>([]);
  const [periods, setPeriods] = useState<TimetablePeriodRow[]>([]);
  const [requirements, setRequirements] = useState<LessonRequirement[]>([]);
  const [drafts, setDrafts] = useState<Record<string, RequirementDraft>>({});

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
  const [savingReq, setSavingReq] = useState<string | null>(null);

  const loadBase = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [classesRes, termsRes, teachersRes, profilesRes] = await Promise.all([
        academicsAPI.listClasses(),
        academicsAPI.listTerms(),
        academicsAPI.listTeachers(),
        academicsAPI.listScheduleProfiles(),
      ]);
      const classList: ClassOption[] = classesRes.data?.data ?? [];
      const termList: TermOption[] = termsRes.data?.data ?? [];
      setClasses(classList);
      setTerms(termList);
      setTeachers(teachersRes.data?.data ?? []);
      const current = termList.find((t) => t.is_current);
      if (current) setTermId(current.id);
      if (classList.length > 0) setClassId((prev) => prev || classList[0].id);
      const profiles = profilesRes.data?.data ?? [];
      const def = profiles.find((p: any) => p.is_default) ?? profiles[0];
      if (def) {
        const slotsRes = await academicsAPI.listScheduleSlots(def.id);
        setSlots(slotsRes.data?.data ?? []);
      }
    } catch {
      setError("Couldn't load classes, terms or bell schedule. Try refreshing.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadBase();
  }, [loadBase]);

  const loadClassSchedule = useCallback(async () => {
    if (!classId || !termId) return;
    setLoadingSchedule(true);
    setError(null);
    try {
      const [assignmentsRes, timetableRes, reqRes] = await Promise.all([
        academicsAPI.listTeachingAssignmentsForClass(classId, termId),
        academicsAPI.getClassTimetable(classId, termId),
        academicsAPI.listLessonRequirements(classId, termId),
      ]);
      const a: TeachingAssignmentOption[] = assignmentsRes.data?.data ?? [];
      const reqs: LessonRequirement[] = reqRes.data?.data ?? [];
      setAssignments(a);
      setPeriods(timetableRes.data?.data ?? []);
      setRequirements(reqs);
      // Seed requirement drafts from saved values (one row per offering).
      const byOffering: Record<string, LessonRequirement> = {};
      for (const r of reqs) byOffering[r.academic_offering_id] = r;
      const d: Record<string, RequirementDraft> = {};
      for (const opt of a) {
        if (d[opt.academic_offering_id]) continue;
        const existing = byOffering[opt.academic_offering_id];
        d[opt.academic_offering_id] = existing
          ? {
              sessions: String(existing.sessions_per_week),
              single: existing.allow_single,
              double: existing.allow_double,
              triple: existing.allow_triple,
              spread: existing.spread_across_days,
              maxPerDay: existing.max_per_day != null ? String(existing.max_per_day) : "",
              existingId: existing.id,
            }
          : { sessions: "", single: true, double: false, triple: false, spread: true, maxPerDay: "", existingId: null };
      }
      setDrafts(d);
    } catch {
      setError("Couldn't load the timetable for this class.");
    } finally {
      setLoadingSchedule(false);
    }
  }, [classId, termId]);

  const loadTeacherSchedule = useCallback(async () => {
    if (!teacherId || !termId) return;
    setLoadingSchedule(true);
    setError(null);
    try {
      const res = await academicsAPI.getTeacherTimetable(teacherId, termId);
      setPeriods(res.data?.data ?? []);
    } catch {
      setError("Couldn't load this teacher's timetable.");
    } finally {
      setLoadingSchedule(false);
    }
  }, [teacherId, termId]);

  useEffect(() => {
    if (view === "class") loadClassSchedule();
    else loadTeacherSchedule();
  }, [view, loadClassSchedule, loadTeacherSchedule]);

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
      await loadClassSchedule();
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

  const handleSaveRequirement = async (offeringId: string) => {
    const d = drafts[offeringId];
    if (!d || !d.sessions) return;
    setSavingReq(offeringId);
    setError(null);
    try {
      await academicsAPI.upsertLessonRequirement(classId, {
        term_id: termId,
        academic_offering_id: offeringId,
        sessions_per_week: parseInt(d.sessions, 10),
        allow_single: d.single,
        allow_double: d.double,
        allow_triple: d.triple,
        spread_across_days: d.spread,
        max_per_day: d.maxPerDay ? parseInt(d.maxPerDay, 10) : null,
      });
      await loadClassSchedule();
    } catch (err: any) {
      setError(err?.response?.data?.error?.message ?? "Couldn't save this requirement.");
    } finally {
      setSavingReq(null);
    }
  };

  const handleDeleteRequirement = async (offeringId: string) => {
    const d = drafts[offeringId];
    if (!d?.existingId) return;
    if (!window.confirm("Delete this lesson requirement?")) return;
    setError(null);
    try {
      await academicsAPI.deleteLessonRequirement(d.existingId);
      await loadClassSchedule();
    } catch {
      setError("Couldn't delete this requirement.");
    }
  };

  // ── Grid computation ──────────────────────────────────────────
  // Days/sequences come from the active bell-schedule slots when a
  // profile exists, otherwise from whatever the periods reference
  // (legacy Mon-Fri data keeps rendering). Periods match a cell by
  // day_of_week + period_number == sequence.
  const activeSlots = slots.filter((s) => s.is_active);
  const gridDays = Array.from(
    new Set([
      ...activeSlots.map((s) => s.day_of_week),
      ...periods.map((p) => p.day_of_week),
    ])
  ).sort((a, b) => a - b);
  const days = gridDays.length > 0 ? gridDays : [1, 2, 3, 4, 5];
  const gridSeqs = Array.from(
    new Set([
      ...activeSlots.map((s) => s.sequence),
      ...periods.map((p) => p.period_number),
    ])
  ).sort((a, b) => a - b);
  const seqs = gridSeqs.length > 0 ? gridSeqs : Array.from({ length: 10 }, (_, i) => i + 1);
  const slotAt = (day: number, seq: number) =>
    activeSlots.find((s) => s.day_of_week === day && s.sequence === seq);
  const periodAt = (day: number, seq: number) =>
    periods.find((p) => p.day_of_week === day && p.period_number === seq);
  const className = (id: string) => {
    const c = classes.find((x) => x.id === id);
    return c ? `${c.name} ${c.stream ?? ""}`.trim() : "—";
  };

  const setDraft = (offeringId: string, patch: Partial<RequirementDraft>) =>
    setDrafts((prev) => ({ ...prev, [offeringId]: { ...prev[offeringId], ...patch } }));

  // One requirement row per offering (assignments may repeat an offering).
  const offeringRows = assignments.filter(
    (a, i, arr) => arr.findIndex((x) => x.academic_offering_id === a.academic_offering_id) === i
  );

  return (
    <DashboardLayout title="Timetable">
      <div className="space-y-6">
        <div className="page-header flex items-center justify-between">
          <div>
            <h1 className="page-title">Timetable</h1>
            <p className="page-subtitle">Weekly grid — lessons, activities and empty cells at a glance</p>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex rounded-md border border-neutral-200 overflow-hidden text-sm">
              <button
                onClick={() => setView("class")}
                className={`px-3 py-2 ${view === "class" ? "bg-neutral-800 text-white" : "text-neutral-600"}`}
              >
                Class
              </button>
              <button
                onClick={() => setView("teacher")}
                className={`px-3 py-2 ${view === "teacher" ? "bg-neutral-800 text-white" : "text-neutral-600"}`}
              >
                Teacher
              </button>
            </div>
            {view === "class" && (
              <Button onClick={() => setShowCreate((v) => !v)} disabled={!classId || !termId}>
                <Plus className="h-4 w-4 mr-2" /> Schedule a period
              </Button>
            )}
          </div>
        </div>

        {error && (
          <p className="text-sm text-error flex items-center gap-1.5">
            <AlertTriangle className="h-4 w-4" /> {error}
          </p>
        )}

        <Card>
          <CardContent className="pt-6">
            {view === "class" ? (
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
            ) : (
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Teacher</label>
                  <select
                    value={teacherId}
                    onChange={(e) => setTeacherId(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  >
                    <option value="">Choose&hellip;</option>
                    {teachers.map((t) => (
                      <option key={t.id} value={t.id}>{teacherDisplay(t)}</option>
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
            )}

            {view === "class" && classId && termId && assignments.length === 0 && !loadingSchedule && (
              <p className="mt-3 text-xs text-warning flex items-center gap-1.5">
                <AlertTriangle className="h-3.5 w-3.5" />
                This class has no teaching assignments for this term yet — assign a teacher to a
                subject or learning area before scheduling a period.
              </p>
            )}
          </CardContent>
        </Card>

        {showCreate && view === "class" && (
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
                      <option key={d.value} value={d.value}>{dayLabel(d.value)}</option>
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
                    {seqs.map((s) => (
                      <option key={s} value={s}>Period {s}</option>
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
            <CardTitle className="flex items-center gap-2">
              <Calendar className="h-4 w-4" />
              {view === "class" ? "Weekly schedule" : "Teacher schedule"}
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0 overflow-x-auto">
            {loadingSchedule ? (
              <div className="p-8 text-center text-sm text-neutral-400">Loading schedule&hellip;</div>
            ) : (
              <table className="w-full min-w-[720px]">
                <thead>
                  <tr className="border-b border-neutral-100">
                    <th className="text-left px-4 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide w-16">Seq</th>
                    {days.map((d) => (
                      <th key={d} className="text-left px-4 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">
                        {dayLabel(d)}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-50">
                  {seqs.map((seq) => (
                    <tr key={seq} className="hover:bg-neutral-50 align-top">
                      <td className="px-4 py-3 text-sm text-neutral-400">{seq}</td>
                      {days.map((d) => {
                        const slot = slotAt(d, seq);
                        const p = periodAt(d, seq);
                        return (
                          <td key={d} className="px-4 py-3 text-sm min-w-[140px]">
                            {slot && (
                              <div className="text-[11px] text-neutral-400 mb-0.5">
                                {slot.start_time}–{slot.end_time}
                              </div>
                            )}
                            {p ? (
                              p.offering_name ? (
                                <div>
                                  <div className="text-neutral-800 font-medium">{p.offering_name}</div>
                                  <div className="text-xs text-neutral-500">
                                    {view === "teacher" ? className(p.class_id) : (p.teacher_name ?? "—")}
                                  </div>
                                  {p.room && <div className="text-xs text-neutral-400">{p.room}</div>}
                                  {!slot && (
                                    <div className="text-[11px] text-neutral-400">{p.start_time}–{p.end_time}</div>
                                  )}
                                </div>
                              ) : (
                                <span className="inline-flex items-center gap-1 text-warning text-xs">
                                  <AlertTriangle className="h-3.5 w-3.5" /> Not linked to an assignment
                                </span>
                              )
                            ) : slot && !slot.is_instructional ? (
                              <span className="inline-block rounded bg-neutral-100 px-2 py-0.5 text-xs font-medium text-neutral-600">
                                {slot.slot_type}
                              </span>
                            ) : (
                              <span className="text-neutral-300">—</span>
                            )}
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </CardContent>
        </Card>

        {view === "class" && classId && termId && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Lesson requirements</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {assignments.length === 0 ? (
                <div className="p-8 text-center text-sm text-neutral-400">
                  Assign teachers to offerings first — requirements are set per offering.
                </div>
              ) : (
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-neutral-100">
                      <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Offering</th>
                      <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Sessions/wk</th>
                      <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Shapes</th>
                      <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Spread / max-day</th>
                      <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-50">
                    {offeringRows.map((a) => {
                      const d = drafts[a.academic_offering_id] ?? {
                        sessions: "", single: true, double: false, triple: false,
                        spread: true, maxPerDay: "", existingId: null,
                      };
                      return (
                        <tr key={a.academic_offering_id} className="hover:bg-neutral-50">
                          <td className="px-6 py-3 text-sm">
                            <div className="text-neutral-800 font-medium">{a.offering_name}</div>
                            <div className="text-xs text-neutral-500">{a.offering_type} — {a.teacher_name}</div>
                          </td>
                          <td className="px-6 py-3">
                            <input
                              type="number" min={1}
                              value={d.sessions}
                              onChange={(e) => setDraft(a.academic_offering_id, { sessions: e.target.value })}
                              className="w-20 rounded-md border border-neutral-200 px-2 py-1.5 text-sm"
                            />
                          </td>
                          <td className="px-6 py-3 text-sm">
                            <label className="inline-flex items-center gap-1 mr-3 text-neutral-700">
                              <input type="checkbox" checked={d.single} onChange={(e) => setDraft(a.academic_offering_id, { single: e.target.checked })} /> S
                            </label>
                            <label className="inline-flex items-center gap-1 mr-3 text-neutral-700">
                              <input type="checkbox" checked={d.double} onChange={(e) => setDraft(a.academic_offering_id, { double: e.target.checked })} /> D
                            </label>
                            <label className="inline-flex items-center gap-1 text-neutral-700">
                              <input type="checkbox" checked={d.triple} onChange={(e) => setDraft(a.academic_offering_id, { triple: e.target.checked })} /> T
                            </label>
                          </td>
                          <td className="px-6 py-3 text-sm">
                            <label className="inline-flex items-center gap-1 mr-3 text-neutral-700">
                              <input type="checkbox" checked={d.spread} onChange={(e) => setDraft(a.academic_offering_id, { spread: e.target.checked })} /> spread
                            </label>
                            <input
                              type="number" min={1} placeholder="max/day"
                              value={d.maxPerDay}
                              onChange={(e) => setDraft(a.academic_offering_id, { maxPerDay: e.target.value })}
                              className="w-20 rounded-md border border-neutral-200 px-2 py-1.5 text-sm"
                            />
                          </td>
                          <td className="px-6 py-3 text-sm whitespace-nowrap">
                            <Button
                              onClick={() => handleSaveRequirement(a.academic_offering_id)}
                              disabled={savingReq === a.academic_offering_id || !d.sessions}
                            >
                              {savingReq === a.academic_offering_id ? "Saving…" : d.existingId ? "Update" : "Save"}
                            </Button>
                            {d.existingId && (
                              <button
                                onClick={() => handleDeleteRequirement(a.academic_offering_id)}
                                className="ml-2 text-xs text-error underline inline-flex items-center gap-1"
                              >
                                <Trash2 className="h-3 w-3" /> Delete
                              </button>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </DashboardLayout>
  );
}
