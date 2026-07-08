"use client";
import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
  CheckCircle2, AlertCircle, DollarSign, BookOpen,
  Bell, Phone, Loader2, AlertTriangle, ChevronRight,
} from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { parentAPI, announcementsAPI, academicsAPI } from "@/lib/api";
import { formatKES, formatDate } from "@/lib/utils";
import { useAuthStore } from "@/lib/store";
import Link from "next/link";

interface Child {
  id: string;
  full_name: string;
  admission_number: string;
  class_name: string;
  status: string;
}

interface FeeAccount {
  balance_due: number;
  total_billed: number;
  total_paid: number;
}

interface ReportCardSummary {
  aggregate_marks: number;
  grade_points: number;
  class_position: number | null;
  term: string;
  status: string;
}

interface AttendanceSummary {
  present: number;
  total: number;
  percentage: number;
}

interface Announcement {
  id: string;
  title: string;
  body: string;
  created_at: string;
  is_read: boolean;
}

export default function ParentDashboard() {
  const { user } = useAuthStore();
  const [children, setChildren] = useState<Child[]>([]);
  const [selectedChild, setSelectedChild] = useState<Child | null>(null);
  const [feeAccount, setFeeAccount] = useState<FeeAccount | null>(null);
  const [reportCard, setReportCard] = useState<ReportCardSummary | null>(null);
  const [attendance, setAttendance] = useState<AttendanceSummary | null>(null);
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load children on mount
  useEffect(() => {
    parentAPI.getChildren()
      .then((res) => {
        const kids: Child[] = res.data?.data ?? [];
        setChildren(kids);
        if (kids.length > 0) setSelectedChild(kids[0]);
      })
      .catch(() => setError("Could not load your children's data."))
      .finally(() => setLoading(false));
  }, []);

  // Load child-specific data when selected child changes
  useEffect(() => {
    if (!selectedChild) return;

    const childId = selectedChild.id;

    // Fee account
    parentAPI.getChildFeeAccount(childId)
      .then((res) => setFeeAccount(res.data?.data ?? null))
      .catch(() => setFeeAccount(null));

    // Report card
    parentAPI.getChildReportCard(childId)
      .then((res) => setReportCard(res.data?.data ?? null))
      .catch(() => setReportCard(null));

    // Announcements
    announcementsAPI.list({ page_size: 5 })
      .then((res) => setAnnouncements(res.data?.data ?? []))
      .catch(() => setAnnouncements([]));

  }, [selectedChild]);

  const firstName = user?.full_name?.split(" ")[0] ?? "Parent";

  if (loading) {
    return (
      <DashboardLayout title="Parent Portal">
        <div className="flex items-center justify-center h-64 gap-2 text-neutral-500">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span className="text-sm">Loading your portal…</span>
        </div>
      </DashboardLayout>
    );
  }

  if (error || children.length === 0) {
    return (
      <DashboardLayout title="Parent Portal">
        <Card className="border-error/30 bg-error-light/20">
          <CardContent className="py-8 text-center">
            <AlertTriangle className="h-8 w-8 text-error mx-auto mb-3" />
            <p className="text-sm font-medium text-neutral-800">
              {error ?? "No students are linked to your account yet."}
            </p>
            <p className="text-xs text-neutral-500 mt-1">
              Please contact the school office if this is incorrect.
            </p>
          </CardContent>
        </Card>
      </DashboardLayout>
    );
  }

  const child = selectedChild!;
  const attendancePct = attendance?.percentage ?? 0;

  return (
    <DashboardLayout title="Parent Portal">
      <div className="space-y-5 max-w-2xl lg:max-w-none">
        {/* Header */}
        <div>
          <h1 className="page-title">Hello, {firstName}</h1>
          <p className="page-subtitle">
            {new Date().toLocaleDateString("en-KE", {
              weekday: "long", year: "numeric", month: "long", day: "numeric",
            })}
          </p>
        </div>

        {/* Child selector — only shown if multiple children */}
        {children.length > 1 && (
          <div className="flex gap-2 flex-wrap">
            {children.map((c) => (
              <button
                key={c.id}
                onClick={() => setSelectedChild(c)}
                className={`px-3 py-1.5 rounded-full text-sm font-medium border transition-colors ${
                  selectedChild?.id === c.id
                    ? "bg-primary-600 text-white border-primary-600"
                    : "bg-white text-neutral-600 border-neutral-200 hover:border-primary-300"
                }`}
              >
                {c.full_name.split(" ")[0]}
              </button>
            ))}
          </div>
        )}

        {/* Child card */}
        <motion.div
          key={child.id}
          initial={{ opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.18 }}
        >
          <Card>
            <CardContent className="p-5">
              <div className="flex items-center gap-4">
                <div className="w-14 h-14 rounded-full bg-primary-100 flex items-center justify-center flex-shrink-0">
                  <span className="text-lg font-semibold text-primary-700">
                    {child.full_name.split(" ").map((n) => n[0]).slice(0, 2).join("")}
                  </span>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-semibold text-neutral-900">{child.full_name}</p>
                  <p className="text-sm text-neutral-500">
                    {child.class_name} · {child.admission_number}
                  </p>
                </div>
                <StatusBadge status={child.status} />
              </div>

              {/* Attendance bar */}
              {attendance && (
                <div className="mt-4 pt-4 border-t border-neutral-100">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs text-neutral-500">Term attendance</p>
                    <p className="text-xs font-semibold text-neutral-700">
                      {attendance.present}/{attendance.total} days ({attendancePct.toFixed(1)}%)
                    </p>
                  </div>
                  <div className="h-2 bg-neutral-100 rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all ${
                        attendancePct >= 80 ? "bg-success" : "bg-warning"
                      }`}
                      style={{ width: `${Math.min(attendancePct, 100)}%` }}
                    />
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>

        {/* Fee balance + latest results */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Card>
            <CardContent className="p-5">
              <div className="flex items-start justify-between">
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Fee Balance</p>
                <DollarSign className="h-4 w-4 text-neutral-300" />
              </div>
              {feeAccount ? (
                <>
                  <p className={`mt-3 text-2xl font-semibold ${
                    feeAccount.balance_due > 0 ? "text-error" : "text-success"
                  }`}>
                    {feeAccount.balance_due > 0
                      ? formatKES(feeAccount.balance_due)
                      : "Fully Paid"}
                  </p>
                  {feeAccount.balance_due > 0 ? (
                    <Button size="sm" className="mt-3 w-full">
                      Pay Now via M-Pesa
                    </Button>
                  ) : (
                    <p className="text-xs text-success mt-1.5">✓ No outstanding balance</p>
                  )}
                </>
              ) : (
                <p className="mt-3 text-sm text-neutral-400">No fee data available</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-5">
              <div className="flex items-start justify-between">
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Latest Results</p>
                <BookOpen className="h-4 w-4 text-neutral-300" />
              </div>
              {reportCard ? (
                <>
                  <p className="mt-3 text-2xl font-semibold text-neutral-900">
                    {reportCard.aggregate_marks.toFixed(1)}%
                  </p>
                  <p className="text-xs text-neutral-500 mt-1">
                    {reportCard.class_position
                      ? `Position ${reportCard.class_position}`
                      : `${reportCard.grade_points} points`}
                    {" · "}{reportCard.term}
                  </p>
                  <Link href={`/parent/results?child=${child.id}`} className="mt-3 block w-full">
                  <Button size="sm" variant="outline" className="w-full">
                    View Report Card
                    <ChevronRight className="h-3.5 w-3.5 ml-1" />
                    </Button>
                    </Link>
                </>
              ) : (
                <p className="mt-3 text-sm text-neutral-400">No published report card yet</p>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Announcements */}
        {announcements.length > 0 && (
          <Card>
            <CardHeader className="flex-row items-center justify-between pb-3">
              <CardTitle>Announcements</CardTitle>
              <Bell className="h-4 w-4 text-neutral-400" />
            </CardHeader>
            <CardContent className="p-0">
              {announcements.slice(0, 3).map((ann) => (
                <div
                  key={ann.id}
                  className={`px-5 py-4 border-b border-neutral-50 last:border-0 ${
                    !ann.is_read ? "bg-primary-50/40" : ""
                  }`}
                >
                  <div className="flex items-start gap-3">
                    {!ann.is_read && (
                      <div className="w-2 h-2 rounded-full bg-primary-500 mt-1.5 flex-shrink-0" />
                    )}
                    <div className="flex-1">
                      <div className="flex items-start justify-between gap-2">
                        <p className="text-sm font-semibold text-neutral-800">{ann.title}</p>
                        <p className="text-xs text-neutral-400 whitespace-nowrap">
                          {formatDate(ann.created_at)}
                        </p>
                      </div>
                      <p className="text-sm text-neutral-600 mt-1 line-clamp-2">{ann.body}</p>
                    </div>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        )}

        {/* Contact school */}
        <Card className="bg-neutral-50 border-neutral-200">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-semibold text-neutral-800">Need help?</p>
                <p className="text-xs text-neutral-500 mt-0.5">Contact the school office</p>
              </div>
              <Link href="/parent/messages">
              <Button size="sm" variant="outline">
                <Phone className="h-4 w-4 mr-1.5" />
                Message
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}