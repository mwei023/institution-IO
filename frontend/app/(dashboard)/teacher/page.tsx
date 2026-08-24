"use client";
import React, { useEffect, useState } from "react";
import { BookOpen, Users, ClipboardList, MessageSquare, Clock, CheckCircle2, Loader2, AlertTriangle } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { StatCard } from "@/components/shared/StatCard";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { academicsAPI } from "@/lib/api";
import { useAuthStore } from "@/lib/store";

// Matches academics.TimetablePeriod from the backend (model.go). Note there
// is no class_name field — only class_id — so we resolve names via a
// separate classes lookup below rather than assume one exists.
interface TimetablePeriod {
  id: string;
  class_id: string;
  subject_id: string;
  teacher_id: string;
  subject_name: string;
  teacher_name: string;
  day_of_week: number; // 1=Monday ... 5=Friday, enforced by DB CHECK constraint
  period_number: number;
  start_time: string; // "HH:MM"
  end_time: string;
  room?: string | null;
  is_published: boolean;
}

interface ClassOption {
  id: string;
  name: string;
  stream?: string | null;
}

interface Term {
  id: string;
  name: string;
  is_current: boolean;
}

type PeriodStatus = "done" | "current" | "upcoming";

// Placeholder sections below (assignments, messages) have no backend yet —
// no assignments module exists in internal/, no messagesAPI in api.ts.
// Kept as clearly-labeled sample data (same convention as the Principal
// Dashboard's charts) rather than wired to endpoints that don't exist.
const pendingAssignments = [
  { id: "1", title: "Essay - Oral Literature", class: "Form 2 East", due: "Tomorrow", submitted: 28, total: 42 },
  { id: "2", title: "Grammar Exercise 5", class: "Form 1 North", due: "Fri", submitted: 35, total: 40 },
  { id: "3", title: "Book Review", class: "Form 4 East", due: "Next Mon", submitted: 19, total: 38 },
];

const recentMessages = [
  { id: "1", from: "John Odhiambo (Amara's parent)", body: "Good morning. Is my daughter in school today?", time: "8:12 AM", unread: true },
  { id: "2", from: "Sarah Waweru (Kioni's parent)", body: "Thank you for the report card feedback.", time: "Yesterday", unread: false },
];

// JS Date.getDay() is 0=Sunday...6=Saturday. The backend/DB use ISO
// 1=Monday...5=Friday (enforced by a CHECK constraint — weekends don't
// exist as valid values). Convert explicitly rather than assume they line up.
function jsDayToIso(jsDay: number): number {
  return jsDay === 0 ? 7 : jsDay; // Sunday -> 7, everything else unchanged
}

function nowAsHHMM(): string {
  const d = new Date();
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

function periodStatus(p: TimetablePeriod, nowStr: string): PeriodStatus {
  if (nowStr < p.start_time) return "upcoming";
  if (nowStr > p.end_time) return "done";
  return "current";
}

const DAY_NAMES: Record<number, string> = {
  1: "Monday", 2: "Tuesday", 3: "Wednesday", 4: "Thursday", 5: "Friday",
};

// Finds the next class chronologically from right now, wrapping into next
// week if today (and the rest of this week) has nothing left. This is what
// lets the dashboard say something useful on a Saturday, a day off, or after
// the last class of the day — instead of just going blank.
function findNextClass(
  periods: TimetablePeriod[],
  todayIso: number,
  nowStr: string
): (TimetablePeriod & { daysUntil: number }) | null {
  if (periods.length === 0) return null;

  const withDistance = periods.map((p) => {
    let daysUntil = (p.day_of_week - todayIso + 7) % 7;
    // Same day, but already started or finished — push to next week's occurrence.
    if (daysUntil === 0 && nowStr >= p.start_time) daysUntil = 7;
    return { ...p, daysUntil };
  });

  withDistance.sort((a, b) => a.daysUntil - b.daysUntil || a.period_number - b.period_number);
  return withDistance[0];
}

export default function TeacherDashboard() {
  const { user } = useAuthStore();

  const [allPeriods, setAllPeriods] = useState<TimetablePeriod[] | null>(null);
  const [timetableError, setTimetableError] = useState(false);
  const [classMap, setClassMap] = useState<Record<string, string>>({});
  const [currentTermName, setCurrentTermName] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    academicsAPI
      .getMyTimetable()
      .then((res) => setAllPeriods(res.data?.data ?? []))
      .catch(() => setTimetableError(true))
      .finally(() => setLoading(false));

    // Best-effort class-name lookup. Degrades gracefully (falls back to a
    // shortened class_id) if the teacher's role doesn't have permission for
    // this route — not confirmed either way, so this must not break the page.
    academicsAPI
      .listClasses()
      .then((res) => {
        const classes: ClassOption[] = res.data?.data ?? [];
        const map: Record<string, string> = {};
        classes.forEach((c) => {
          map[c.id] = c.stream ? `${c.name} ${c.stream}` : c.name;
        });
        setClassMap(map);
      })
      .catch(() => setClassMap({}));

    academicsAPI
      .listTerms()
      .then((res) => {
        const terms: Term[] = res.data?.data ?? [];
        const current = terms.find((t) => t.is_current);
        setCurrentTermName(current ? current.name : null);
      })
      .catch(() => setCurrentTermName(null));
  }, []);

  const [scheduleView, setScheduleView] = useState<"today" | "week">("today");

  const classDisplayName = (classId: string) =>
    classMap[classId] ?? `Class ${classId.slice(0, 8)}`;

  const todayIso = jsDayToIso(new Date().getDay());
  const nowStr = nowAsHHMM();

  const todayPeriods = (allPeriods ?? [])
    .filter((p) => p.day_of_week === todayIso)
    .sort((a, b) => a.period_number - b.period_number)
    .map((p) => ({ ...p, status: periodStatus(p, nowStr) }));

  const currentPeriod = todayPeriods.find((p) => p.status === "current");
  const nextClass = findNextClass(allPeriods ?? [], todayIso, nowStr);
  const myClassCount = new Set((allPeriods ?? []).map((p) => p.class_id)).size;

  // Full week, grouped by day, for the "This Week" toggle — this is the
  // "view my this term timetable" view. It's scoped to the current week's
  // recurring schedule (what getMyTimetable already returns) rather than a
  // full term calendar with actual dates, which would need term start/end
  // dates woven in — flagged as a follow-on, not attempted here.
  const weekByDay: Record<number, TimetablePeriod[]> = {};
  (allPeriods ?? []).forEach((p) => {
    if (!weekByDay[p.day_of_week]) weekByDay[p.day_of_week] = [];
    weekByDay[p.day_of_week].push(p);
  });
  Object.values(weekByDay).forEach((list) => list.sort((a, b) => a.period_number - b.period_number));

  const firstName = user?.full_name?.split(" ")[0] ?? "Teacher";

  return (
    <DashboardLayout title="Teacher Dashboard">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Good morning, {firstName}</h1>
            <p className="page-subtitle">
              {currentTermName ?? "No active term set"}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            label="My Classes"
            value={allPeriods ? String(myClassCount) : "—"}
            sub="This term"
            icon={Users}
            iconColor="text-primary-500"
          />
          <StatCard
            label="Attendance Done"
            value="—"
            sub="Not available yet"
            icon={CheckCircle2}
            iconColor="text-neutral-400"
          />
          <StatCard
            label="Assignments Open"
            value={pendingAssignments.length}
            sub="Sample data"
            icon={ClipboardList}
            iconColor="text-warning"
          />
          <StatCard
            label="Parent Messages"
            value="1"
            sub="Sample data"
            icon={MessageSquare}
            iconColor="text-info"
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <Card className="lg:col-span-2">
            <CardHeader className="flex-row items-center justify-between pb-3">
              <CardTitle>My Schedule</CardTitle>
              <div className="flex items-center gap-2">
                <div className="flex rounded-md border border-neutral-200 overflow-hidden text-xs">
                  <button
                    onClick={() => setScheduleView("today")}
                    className={`px-2.5 py-1 font-medium ${scheduleView === "today" ? "bg-primary-600 text-white" : "text-neutral-600 hover:bg-neutral-50"}`}
                  >
                    Today
                  </button>
                  <button
                    onClick={() => setScheduleView("week")}
                    className={`px-2.5 py-1 font-medium ${scheduleView === "week" ? "bg-primary-600 text-white" : "text-neutral-600 hover:bg-neutral-50"}`}
                  >
                    This Week
                  </button>
                </div>
                <Button size="sm" variant="outline" onClick={() => { window.location.href = "/teacher/attendance"; }}>
                  Take Attendance
                </Button>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {loading ? (
                <div className="px-6 py-10 text-center text-sm text-neutral-500 flex items-center justify-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading timetable…
                </div>
              ) : timetableError ? (
                <div className="px-6 py-10 text-center text-sm text-error flex items-center justify-center gap-2">
                  <AlertTriangle className="h-4 w-4" /> Could not load your timetable.
                </div>
              ) : scheduleView === "today" ? (
                todayPeriods.length === 0 ? (
                  <div className="px-6 py-8">
                    <p className="text-sm text-neutral-500 text-center mb-4">No classes scheduled today.</p>
                    {nextClass && (
                      <div className="bg-primary-50 border border-primary-100 rounded-lg px-4 py-3 max-w-md mx-auto text-center">
                        <p className="text-xs font-medium text-primary-600 uppercase tracking-wide mb-1">
                          Next class
                        </p>
                        <p className="text-sm font-semibold text-neutral-900">{nextClass.subject_name}</p>
                        <p className="text-xs text-neutral-600">{classDisplayName(nextClass.class_id)}</p>
                        <p className="text-xs text-neutral-500 mt-1">
                          {DAY_NAMES[nextClass.day_of_week]}
                          {nextClass.daysUntil >= 7 ? " (next week)" : ""}
                          {", "}{nextClass.start_time} - {nextClass.end_time}
                          {nextClass.room ? ` · Room ${nextClass.room}` : ""}
                        </p>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="divide-y divide-neutral-50">
                    {todayPeriods.map((period) => (
                      <div
                        key={period.id}
                        className={`flex items-center gap-4 px-6 py-3.5 ${period.status === "current" ? "bg-primary-50" : ""}`}
                      >
                        <div className="w-24 flex-shrink-0">
                          <p className="text-xs font-medium text-neutral-500">
                            {period.start_time} - {period.end_time}
                          </p>
                        </div>
                        <div className="flex-1">
                          <p className="text-sm font-medium text-neutral-800">{period.subject_name}</p>
                          <p className="text-xs text-neutral-500">
                            {classDisplayName(period.class_id)}
                            {period.room ? ` - Room ${period.room}` : ""}
                          </p>
                        </div>
                        <div>
                          {period.status === "current" && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-primary-100 text-primary-700">
                              <span className="w-1.5 h-1.5 rounded-full bg-primary-500 animate-pulse" />
                              Now
                            </span>
                          )}
                          {period.status === "done" && <CheckCircle2 className="h-4 w-4 text-success" />}
                          {period.status === "upcoming" && <Clock className="h-4 w-4 text-neutral-300" />}
                        </div>
                      </div>
                    ))}
                  </div>
                )
              ) : (
                // "This Week" view — every day that has periods, Monday first.
                <div className="divide-y divide-neutral-100">
                  {[1, 2, 3, 4, 5].map((day) => {
                    const dayPeriods = weekByDay[day] ?? [];
                    if (dayPeriods.length === 0) return null;
                    return (
                      <div key={day} className="px-6 py-3">
                        <p className={`text-xs font-semibold uppercase tracking-wide mb-2 ${day === todayIso ? "text-primary-600" : "text-neutral-500"}`}>
                          {DAY_NAMES[day]}{day === todayIso ? " · Today" : ""}
                        </p>
                        <div className="space-y-1.5">
                          {dayPeriods.map((p) => (
                            <div key={p.id} className="flex items-center gap-3 text-sm">
                              <span className="w-20 flex-shrink-0 text-xs text-neutral-500">{p.start_time}</span>
                              <span className="font-medium text-neutral-800">{p.subject_name}</span>
                              <span className="text-neutral-400">·</span>
                              <span className="text-neutral-600">{classDisplayName(p.class_id)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    );
                  })}
                  {Object.keys(weekByDay).length === 0 && (
                    <p className="text-sm text-neutral-500 text-center py-8">No timetable assigned this term.</p>
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          <div className="space-y-4">
            {currentPeriod ? (
              <Card className="border-primary-200 bg-primary-50">
                <CardContent className="p-4">
                  <p className="text-xs font-medium text-primary-600 uppercase tracking-wide mb-2">
                    Current Lesson
                  </p>
                  <p className="text-base font-semibold text-neutral-900">{currentPeriod.subject_name}</p>
                  <p className="text-sm text-neutral-600">{classDisplayName(currentPeriod.class_id)}</p>
                  <p className="text-xs text-neutral-500 mt-1">
                    {currentPeriod.room ? `Room ${currentPeriod.room} - ` : ""}
                    {currentPeriod.start_time} - {currentPeriod.end_time}
                  </p>
                  <Button size="sm" className="mt-3 w-full" onClick={() => { window.location.href = "/teacher/attendance"; }}>
                    Mark Attendance
                  </Button>
                </CardContent>
              </Card>
            ) : nextClass ? (
              <Card className="border-neutral-200">
                <CardContent className="p-4">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide mb-2">
                    Next Lesson
                  </p>
                  <p className="text-base font-semibold text-neutral-900">{nextClass.subject_name}</p>
                  <p className="text-sm text-neutral-600">{classDisplayName(nextClass.class_id)}</p>
                  <p className="text-xs text-neutral-500 mt-1">
                    {DAY_NAMES[nextClass.day_of_week]}
                    {nextClass.daysUntil >= 7 ? " (next week)" : ""}
                    {", "}{nextClass.start_time} - {nextClass.end_time}
                  </p>
                </CardContent>
              </Card>
            ) : null}

            <Card>
              <CardHeader className="flex-row items-center justify-between pb-3">
                <CardTitle>Parent Messages</CardTitle>
                <span className="text-xs text-neutral-400">Sample data</span>
              </CardHeader>
              <CardContent className="p-0">
                {recentMessages.map((msg) => (
                  <div
                    key={msg.id}
                    className={`px-4 py-3 border-b border-neutral-50 last:border-0 ${
                      msg.unread ? "bg-primary-50/50" : ""
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <p className="text-xs font-semibold text-neutral-700 leading-tight">{msg.from}</p>
                      <span className="text-xs text-neutral-400 whitespace-nowrap">{msg.time}</span>
                    </div>
                    <p className="text-xs text-neutral-500 mt-0.5 truncate">{msg.body}</p>
                  </div>
                ))}
                <div className="p-4">
                  <Button size="sm" variant="outline" className="w-full" onClick={() => { window.location.href = "/teacher/messages"; }}>
                    View All Messages
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>

        <Card>
          <CardHeader className="flex-row items-center justify-between pb-3">
            <CardTitle>Assignments - Pending Action</CardTitle>
            <span className="text-xs text-neutral-400">Sample data</span>
          </CardHeader>
          <CardContent className="p-0">
            <table className="w-full">
              <thead>
                <tr className="border-b border-neutral-100">
                  <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Assignment</th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden sm:table-cell">Class</th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Due</th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden md:table-cell">Submissions</th>
                  <th className="px-6 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-50">
                {pendingAssignments.map((a) => (
                  <tr key={a.id} className="hover:bg-neutral-50">
                    <td className="px-6 py-3.5 text-sm font-medium text-neutral-800">{a.title}</td>
                    <td className="px-6 py-3.5 text-sm text-neutral-600 hidden sm:table-cell">{a.class}</td>
                    <td className="px-6 py-3.5 text-sm text-neutral-600">{a.due}</td>
                    <td className="px-6 py-3.5 hidden md:table-cell">
                      <div className="flex items-center gap-2">
                        <div className="flex-1 bg-neutral-100 rounded-full h-1.5">
                          <div
                            className="bg-primary-500 h-1.5 rounded-full"
                            style={{ width: `${(a.submitted / a.total) * 100}%` }}
                          />
                        </div>
                        <span className="text-xs text-neutral-500">{a.submitted}/{a.total}</span>
                      </div>
                    </td>
                    <td className="px-6 py-3.5 text-right">
                      <Button size="sm" variant="ghost" onClick={() => { window.location.href = `/teacher/assignments/${a.id}`; }}>
                        Mark
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}