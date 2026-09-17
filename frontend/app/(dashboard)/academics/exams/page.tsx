"use client";
import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Plus, AlertTriangle, Loader2 } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { examsAPI } from "@/lib/api";

interface ExamSession {
  id: string;
  name: string;
  exam_type: string;
  start_date: string;
  end_date: string;
  status: string;
  version: number;
}

const EXAM_TYPES = ["internal", "mock", "mid_term", "end_of_term", "external", "other"];

const statusBadge = (s: string) => {
  const base = "inline-block rounded px-2 py-0.5 text-xs font-medium";
  if (s === "published") return `${base} bg-green-100 text-green-800`;
  if (s === "validated") return `${base} bg-blue-100 text-blue-800`;
  if (s === "archived") return `${base} bg-neutral-200 text-neutral-600`;
  return `${base} bg-amber-100 text-amber-800`;
};

export default function ExamSessionsPage() {
  const [sessions, setSessions] = useState<ExamSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [examType, setExamType] = useState("internal");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [creating, setCreating] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await examsAPI.listSessions();
      setSessions(res.data?.data ?? []);
    } catch {
      setError("Couldn't load examination sessions.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleCreate = async () => {
    if (!name || !startDate || !endDate) return;
    setCreating(true);
    setError(null);
    try {
      await examsAPI.createSession({
        name, exam_type: examType, start_date: startDate, end_date: endDate,
      });
      setName("");
      setStartDate("");
      setEndDate("");
      setShowCreate(false);
      await load();
    } catch (err: any) {
      setError(err?.response?.data?.error?.message ?? "Couldn't create this session.");
    } finally {
      setCreating(false);
    }
  };

  return (
    <DashboardLayout title="Exam Timetable">
      <div className="space-y-6">
        <div className="page-header flex items-center justify-between">
          <div>
            <h1 className="page-title">Examination Timetable</h1>
            <p className="page-subtitle">Draft, validate and publish exam schedules — one canonical timetable per session</p>
          </div>
          <Button onClick={() => setShowCreate((v) => !v)}>
            <Plus className="h-4 w-4 mr-2" /> New session
          </Button>
        </div>

        {error && (
          <p className="text-sm text-error flex items-center gap-1.5">
            <AlertTriangle className="h-4 w-4" /> {error}
          </p>
        )}

        {showCreate && (
          <Card>
            <CardHeader className="pb-3"><CardTitle>New examination session</CardTitle></CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2">
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Name</label>
                  <input value={name} onChange={(e) => setName(e.target.value)}
                    placeholder="e.g. End of Term 2 Exams"
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Type</label>
                  <select value={examType} onChange={(e) => setExamType(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm">
                    {EXAM_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-medium text-neutral-500 mb-1.5">Start</label>
                    <input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)}
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-neutral-500 mb-1.5">End</label>
                    <input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)}
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm" />
                  </div>
                </div>
              </div>
              <div className="mt-4">
                <Button onClick={handleCreate} disabled={creating || !name || !startDate || !endDate}>
                  {creating ? <><Loader2 className="h-4 w-4 mr-2 animate-spin" /> Creating&hellip;</> : "Create session"}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardContent className="p-0">
            {loading ? (
              <div className="p-8 text-center text-sm text-neutral-400">Loading sessions&hellip;</div>
            ) : sessions.length === 0 ? (
              <div className="p-8 text-center text-sm text-neutral-400">
                No examination sessions yet — create one to start scheduling.
              </div>
            ) : (
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-100">
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Session</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Type</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Dates</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Status</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide"></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-50">
                  {sessions.map((s) => (
                    <tr key={s.id} className="hover:bg-neutral-50">
                      <td className="px-6 py-3 text-sm font-medium text-neutral-800">{s.name}</td>
                      <td className="px-6 py-3 text-sm text-neutral-500">{s.exam_type}</td>
                      <td className="px-6 py-3 text-sm text-neutral-500">{s.start_date} → {s.end_date}</td>
                      <td className="px-6 py-3 text-sm">
                        <span className={statusBadge(s.status)}>{s.status} · v{s.version}</span>
                      </td>
                      <td className="px-6 py-3 text-sm text-right">
                        <Link href={`/academics/exams/${s.id}`} className="text-primary-600 underline">
                          Open
                        </Link>
                      </td>
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
