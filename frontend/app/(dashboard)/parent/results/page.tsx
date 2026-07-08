"use client";
import React, { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Loader2, AlertTriangle, BookOpen, Award, Users, Calendar } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { parentAPI } from "@/lib/api";

interface SubjectScore {
  subject_id: string;
  subject_name: string;
  percentage: number;
}

interface ReportCard {
  id: string;
  student_name: string;
  class_name: string;
  term: string;
  aggregate_marks: number;
  grade_points: number;
  class_position: number | null;
  attendance_days_present: number;
  attendance_days_total: number;
  teacher_comment: string | null;
  principal_comment: string | null;
  published_at: string;
}

function gradeFromPercentage(pct: number): { letter: string; color: string } {
  if (pct >= 80) return { letter: "A",  color: "text-success" };
  if (pct >= 75) return { letter: "A-", color: "text-success" };
  if (pct >= 70) return { letter: "B+", color: "text-primary-600" };
  if (pct >= 65) return { letter: "B",  color: "text-primary-600" };
  if (pct >= 60) return { letter: "B-", color: "text-primary-600" };
  if (pct >= 55) return { letter: "C+", color: "text-warning" };
  if (pct >= 50) return { letter: "C",  color: "text-warning" };
  if (pct >= 45) return { letter: "C-", color: "text-warning" };
  if (pct >= 40) return { letter: "D+", color: "text-error" };
  if (pct >= 35) return { letter: "D",  color: "text-error" };
  if (pct >= 30) return { letter: "D-", color: "text-error" };
  return { letter: "E", color: "text-error" };
}

function PercentageBar({ value }: { value: number }) {
  const { color } = gradeFromPercentage(value);
  const barColor = color.replace("text-", "bg-").replace("text-primary-600", "bg-primary-500");
  return (
    <div className="flex items-center gap-2 flex-1">
      <div className="flex-1 bg-neutral-100 rounded-full h-1.5 overflow-hidden">
        <div
          className={`h-full rounded-full ${barColor}`}
          style={{ width: `${Math.min(value, 100)}%` }}
        />
      </div>
      <span className="text-xs font-medium text-neutral-600 w-10 text-right">
        {value.toFixed(1)}%
      </span>
    </div>
  );
}

export default function ParentResultsPage() {
  const searchParams = useSearchParams();
  const childId = searchParams.get("child");

  const [reportCard, setReportCard] = useState<ReportCard | null>(null);
  const [subjectScores, setSubjectScores] = useState<SubjectScore[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!childId) {
      setError("No student specified.");
      setLoading(false);
      return;
    }

    Promise.all([
      parentAPI.getChildReportCard(childId),
      parentAPI.getChildSubjectScores(childId),
    ])
      .then(([rcRes, scoresRes]) => {
        setReportCard(rcRes.data?.data ?? null);
        setSubjectScores(scoresRes.data?.data ?? []);
      })
      .catch(() => setError("Could not load report card."))
      .finally(() => setLoading(false));
  }, [childId]);

  if (loading) {
    return (
      <DashboardLayout title="Report Card">
        <div className="flex items-center justify-center h-64 gap-2 text-neutral-500">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span className="text-sm">Loading report card…</span>
        </div>
      </DashboardLayout>
    );
  }

  if (error || !reportCard) {
    return (
      <DashboardLayout title="Report Card">
        <Card className="border-neutral-200">
          <CardContent className="py-12 text-center">
            <BookOpen className="h-10 w-10 text-neutral-300 mx-auto mb-3" />
            <p className="text-sm font-medium text-neutral-700">
              {error ?? "No published report card available yet."}
            </p>
            <p className="text-xs text-neutral-400 mt-1">
              Report cards are published by the school at the end of each term.
            </p>
          </CardContent>
        </Card>
      </DashboardLayout>
    );
  }

  const { letter: overallGrade, color: gradeColor } = gradeFromPercentage(reportCard.aggregate_marks);
  const attendancePct = reportCard.attendance_days_total > 0
    ? (reportCard.attendance_days_present / reportCard.attendance_days_total) * 100
    : 0;

  return (
    <DashboardLayout title="Report Card">
      <div className="space-y-5 max-w-2xl">
        {/* Header */}
        <div>
          <h1 className="page-title">Report Card</h1>
          <p className="page-subtitle">
            {reportCard.class_name} · {reportCard.term}
          </p>
        </div>

        {/* Summary card */}
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-4">
              <div className="w-14 h-14 rounded-full bg-primary-100 flex items-center justify-center flex-shrink-0">
                <span className="text-lg font-semibold text-primary-700">
                  {reportCard.student_name.split(" ").map((n) => n[0]).slice(0, 2).join("")}
                </span>
              </div>
              <div>
                <p className="font-semibold text-neutral-900">{reportCard.student_name}</p>
                <p className="text-sm text-neutral-500">{reportCard.class_name} · {reportCard.term}</p>
              </div>
            </div>

            <div className="mt-5 grid grid-cols-3 gap-3">
              <div className="text-center p-3 rounded-lg bg-neutral-50">
                <p className={`text-2xl font-bold ${gradeColor}`}>{overallGrade}</p>
                <p className="text-xs text-neutral-500 mt-0.5">Grade</p>
              </div>
              <div className="text-center p-3 rounded-lg bg-neutral-50">
                <p className="text-2xl font-bold text-neutral-900">
                  {reportCard.aggregate_marks.toFixed(1)}%
                </p>
                <p className="text-xs text-neutral-500 mt-0.5">Mean Score</p>
              </div>
              <div className="text-center p-3 rounded-lg bg-neutral-50">
                {reportCard.class_position ? (
                  <>
                    <p className="text-2xl font-bold text-neutral-900">
                      #{reportCard.class_position}
                    </p>
                    <p className="text-xs text-neutral-500 mt-0.5">Position</p>
                  </>
                ) : (
                  <>
                    <p className="text-2xl font-bold text-neutral-900">
                      {reportCard.grade_points}
                    </p>
                    <p className="text-xs text-neutral-500 mt-0.5">Points</p>
                  </>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Subject breakdown */}
        {subjectScores.length > 0 && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Subject Performance</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <div className="divide-y divide-neutral-50">
                {subjectScores
                  .sort((a, b) => b.percentage - a.percentage)
                  .map((s) => {
                    const { letter, color } = gradeFromPercentage(s.percentage);
                    return (
                      <div key={s.subject_id} className="flex items-center gap-3 px-5 py-3">
                        <p className="text-sm font-medium text-neutral-700 w-40 flex-shrink-0 truncate">
                          {s.subject_name}
                        </p>
                        <PercentageBar value={s.percentage} />
                        <span className={`text-xs font-semibold w-6 text-right ${color}`}>
                          {letter}
                        </span>
                      </div>
                    );
                  })}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Attendance */}
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4 text-neutral-400" />
                <p className="text-sm font-medium text-neutral-700">Term Attendance</p>
              </div>
              <p className="text-sm font-semibold text-neutral-800">
                {reportCard.attendance_days_present}/{reportCard.attendance_days_total} days
              </p>
            </div>
            <div className="h-2 bg-neutral-100 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${attendancePct >= 80 ? "bg-success" : "bg-warning"}`}
                style={{ width: `${Math.min(attendancePct, 100)}%` }}
              />
            </div>
            <p className="text-xs text-neutral-500 mt-1.5">{attendancePct.toFixed(1)}% attendance rate</p>
          </CardContent>
        </Card>

        {/* Comments */}
        {(reportCard.teacher_comment || reportCard.principal_comment) && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Comments</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {reportCard.teacher_comment && (
                <div>
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide mb-1">
                    Class Teacher
                  </p>
                  <p className="text-sm text-neutral-700 leading-relaxed">
                    {reportCard.teacher_comment}
                  </p>
                </div>
              )}
              {reportCard.principal_comment && (
                <div>
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide mb-1">
                    Principal
                  </p>
                  <p className="text-sm text-neutral-700 leading-relaxed">
                    {reportCard.principal_comment}
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        <p className="text-xs text-neutral-400 text-center pb-4">
          Published {new Date(reportCard.published_at).toLocaleDateString("en-KE", {
            day: "numeric", month: "long", year: "numeric",
          })}
        </p>
      </div>
    </DashboardLayout>
  );
}