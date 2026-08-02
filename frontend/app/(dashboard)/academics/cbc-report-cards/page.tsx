"use client";
import React, { useEffect, useMemo, useState } from "react";
import {
  FileText, CheckCircle2, AlertTriangle, Loader2,
  Send, Undo2, RefreshCw,
} from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { StatCard } from "@/components/shared/StatCard";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { academicsAPI, cbcAPI } from "@/lib/api";
import { useAuthStore } from "@/lib/store";

// ─── Types (mirrors backend internal/cbc/model.go) ──────────────────

interface ClassOption {
  id: string;
  name: string;
  stream: string;
  level: number;
  enrollment: number;
}

interface TermOption {
  id: string;
  name: string;
  is_current: boolean;
}

interface ReportCard {
  id: string;
  student_id: string;
  student_name: string; // joined server-side, unlike the 8-4-4 version
  mean_points: number | null;
  class_position: number | null; // usually null — CBC ranking_allowed defaults false
  status: "draft" | "published";
}

interface BulkFailure {
  student_id: string;
  reason: string;
}

interface BulkResult {
  class_id: string;
  term_id: string;
  total_students: number;
  succeeded: ReportCard[];
  failed: BulkFailure[];
  succeeded_count: number;
  failed_count: number;
}

export default function CbcReportCardAdminPage() {
  const { hasPermission } = useAuthStore();
  const canPublish = hasPermission("grades:publish");

  const [classes, setClasses] = useState<ClassOption[]>([]);
  const [terms, setTerms] = useState<TermOption[]>([]);
  const [classId, setClassId] = useState("");
  const [termId, setTermId] = useState("");

  const [result, setResult] = useState<BulkResult | null>(null);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actioningId, setActioningId] = useState<string | null>(null);

  // Classes and terms are curriculum-agnostic — same endpoints as the
  // 8-4-4 admin page.
  useEffect(() => {
    academicsAPI.listClasses().then((res) => setClasses(res.data?.data ?? []));
    academicsAPI.listTerms().then((res) => {
      const list: TermOption[] = res.data?.data ?? [];
      setTerms(list);
      const current = list.find((t) => t.is_current);
      if (current) setTermId(current.id);
    });
  }, []);

  const selectedClass = useMemo(
    () => classes.find((c) => c.id === classId),
    [classes, classId]
  );

  const handleGenerate = async () => {
    if (!classId || !termId) return;
    setGenerating(true);
    setError(null);
    try {
      const res = await cbcAPI.generateReportCardsForClass({ class_id: classId, term_id: termId });
      setResult(res.data?.data ?? null);
    } catch (err: any) {
      const message = err?.response?.data?.error?.message;
      if (err?.response?.data?.error?.code === "NO_RATING_BANDS") {
        setError("No CBC rating bands are configured yet for this grade level. Contact your administrator.");
      } else {
        setError(message ?? "Couldn't generate report cards. Try again.");
      }
    } finally {
      setGenerating(false);
    }
  };

  const handlePublish = async (studentId: string) => {
    setActioningId(studentId);
    try {
      await cbcAPI.publishReportCard({ student_id: studentId, term_id: termId });
      setResult((prev) => prev && ({
        ...prev,
        succeeded: prev.succeeded.map((rc) =>
          rc.student_id === studentId ? { ...rc, status: "published" } : rc
        ),
      }));
    } catch {
      setError("Couldn't publish that report card. It may already be published.");
    } finally {
      setActioningId(null);
    }
  };

  const handleUnpublish = async (studentId: string) => {
    setActioningId(studentId);
    try {
      await cbcAPI.unpublishReportCard({ student_id: studentId, term_id: termId });
      setResult((prev) => prev && ({
        ...prev,
        succeeded: prev.succeeded.map((rc) =>
          rc.student_id === studentId ? { ...rc, status: "draft" } : rc
        ),
      }));
    } catch {
      setError("Couldn't unpublish that report card.");
    } finally {
      setActioningId(null);
    }
  };

  const publishedCount = result?.succeeded.filter((rc) => rc.status === "published").length ?? 0;
  const draftCount = (result?.succeeded.length ?? 0) - publishedCount;

  return (
    <DashboardLayout title="Report Cards">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Report Cards</h1>
            <p className="page-subtitle">Generate and publish CBC term results for a class</p>
          </div>
        </div>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle>Select class and term</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap items-end gap-4">
              <div className="w-full sm:w-64">
                <label className="block text-xs font-medium text-neutral-500 mb-1.5">Class</label>
                <select
                  value={classId}
                  onChange={(e) => { setClassId(e.target.value); setResult(null); }}
                  className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm text-neutral-800 focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  <option value="">Choose a class&hellip;</option>
                  {classes.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name} {c.stream} &middot; {c.enrollment} students
                    </option>
                  ))}
                </select>
              </div>

              <div className="w-full sm:w-64">
                <label className="block text-xs font-medium text-neutral-500 mb-1.5">Term</label>
                <select
                  value={termId}
                  onChange={(e) => { setTermId(e.target.value); setResult(null); }}
                  className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm text-neutral-800 focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  <option value="">Choose a term&hellip;</option>
                  {terms.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name} {t.is_current ? "(current)" : ""}
                    </option>
                  ))}
                </select>
              </div>

              <Button onClick={handleGenerate} disabled={!classId || !termId || generating}>
                {generating ? (
                  <><Loader2 className="h-4 w-4 mr-2 animate-spin" /> Generating&hellip;</>
                ) : result ? (
                  <><RefreshCw className="h-4 w-4 mr-2" /> Regenerate</>
                ) : (
                  <><FileText className="h-4 w-4 mr-2" /> Generate report cards</>
                )}
              </Button>
            </div>

            {error && (
              <p className="mt-3 text-sm text-error flex items-center gap-1.5">
                <AlertTriangle className="h-4 w-4" /> {error}
              </p>
            )}

            <p className="mt-3 text-xs text-neutral-400">
              Regenerating never overwrites a published report card — unpublish it first if it needs a correction.
            </p>
          </CardContent>
        </Card>

        {result && (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard label="Total students" value={result.total_students} icon={FileText} iconColor="text-primary-500" />
            <StatCard label="Generated" value={result.succeeded_count} sub={`${draftCount} draft, ${publishedCount} published`} icon={CheckCircle2} iconColor="text-success" />
            <StatCard label="Skipped" value={result.failed_count} sub="No entries yet" icon={AlertTriangle} iconColor="text-warning" />
            <StatCard label="Published" value={publishedCount} sub="Visible to parents" icon={Send} iconColor="text-info" />
          </div>
        )}

        {result && result.succeeded.length > 0 && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>{selectedClass ? `${selectedClass.name} ${selectedClass.stream}` : "Results"}</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-100">
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Student</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Mean Points</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Status</th>
                    <th className="px-6 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-50">
                  {result.succeeded.map((rc) => (
                    <tr key={rc.student_id} className="hover:bg-neutral-50">
                      <td className="px-6 py-3.5 text-sm font-medium text-neutral-800">
                        {rc.student_name}
                      </td>
                      <td className="px-6 py-3.5 text-sm text-neutral-600">
                        {rc.mean_points != null ? rc.mean_points.toFixed(1) : "—"}
                      </td>
                      <td className="px-6 py-3.5">
                        <StatusBadge status={rc.status} />
                      </td>
                      <td className="px-6 py-3.5 text-right">
                        {!canPublish ? (
                          <span className="text-xs text-neutral-400">Admin only</span>
                        ) : rc.status === "published" ? (
                          <Button size="sm" variant="ghost" disabled={actioningId === rc.student_id} onClick={() => handleUnpublish(rc.student_id)}>
                            <Undo2 className="h-4 w-4 mr-1.5" /> Unpublish
                          </Button>
                        ) : (
                          <Button size="sm" variant="outline" disabled={actioningId === rc.student_id} onClick={() => handlePublish(rc.student_id)}>
                            <Send className="h-4 w-4 mr-1.5" /> Publish
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </CardContent>
          </Card>
        )}

        {result && result.failed.length > 0 && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Skipped ({result.failed.length})</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <div className="divide-y divide-neutral-50">
                {result.failed.map((f) => (
                  <div key={f.student_id} className="px-6 py-3 flex items-center justify-between">
                    <span className="text-sm font-medium text-neutral-800">{f.student_id}</span>
                    <span className="text-xs text-neutral-500">{f.reason}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </DashboardLayout>
  );
}