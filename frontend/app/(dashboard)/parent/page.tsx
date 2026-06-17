"use client";
import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import { CheckCircle2, AlertCircle, DollarSign, BookOpen, Bell, Phone } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { formatKES, formatDate } from "@/lib/utils";

// Demo data for parent dashboard
const childData = {
  id: "st000001-0000-0000-0000-000000000001",
  name: "Amara Odhiambo",
  class: "Form 1 East",
  admissionNumber: "2024/F1E/0001",
  photoUrl: null,
  attendanceToday: "present" as const,
  attendanceThisTerm: { present: 42, total: 48, percentage: 87.5 },
  feeBalance: 0,
  latestPosition: { position: 8, outOf: 42, term: "Term 2" },
};

const recentPayments = [
  { id: "1", date: "2024-09-24", amount: 19500, receipt: "RCP/2024/000001", method: "M-Pesa", ref: "QKA6YTRNP1" },
];

const announcements = [
  {
    id: "1",
    title: "Term 3 2024 Fee Deadline Reminder",
    body: "School fees are due by 13th September 2024. Pay via M-Pesa Paybill 400200, Account = admission number.",
    type: "fee_notice",
    date: "2024-09-26",
    isRead: false,
  },
  {
    id: "2",
    title: "Form 4 Mock Examination Schedule",
    body: "Mock exams will be held from 7th to 18th October 2024.",
    type: "academic",
    date: "2024-09-23",
    isRead: true,
  },
];

export default function ParentDashboard() {
  const [showPayModal, setShowPayModal] = useState(false);
  const isPresent = childData.attendanceToday === "present";

  return (
    <DashboardLayout title="Parent Portal">
      <div className="space-y-5 max-w-2xl lg:max-w-none">
        {/* Header */}
        <div>
          <h1 className="page-title">Hello, John</h1>
          <p className="page-subtitle">
            {new Date().toLocaleDateString("en-KE", {
              weekday: "long", year: "numeric", month: "long", day: "numeric",
            })}
          </p>
        </div>

        {/* Child attendance card — most important info */}
        <motion.div
          initial={{ opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.18 }}
        >
          <Card className={`border-2 ${isPresent ? "border-success/30" : "border-error/30"}`}>
            <CardContent className="p-5">
              <div className="flex items-center gap-4">
                {/* Avatar */}
                <div className="w-14 h-14 rounded-full bg-primary-100 flex items-center justify-center flex-shrink-0">
                  <span className="text-lg font-semibold text-primary-700">AO</span>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-semibold text-neutral-900">{childData.name}</p>
                  <p className="text-sm text-neutral-500">{childData.class} · {childData.admissionNumber}</p>
                </div>
                <div className="text-right flex-shrink-0">
                  {isPresent ? (
                    <div className="flex items-center gap-1.5 text-success">
                      <CheckCircle2 className="h-5 w-5" />
                      <span className="text-sm font-semibold">In School</span>
                    </div>
                  ) : (
                    <div className="flex items-center gap-1.5 text-error">
                      <AlertCircle className="h-5 w-5" />
                      <span className="text-sm font-semibold">Absent</span>
                    </div>
                  )}
                  <p className="text-xs text-neutral-400 mt-0.5">Today</p>
                </div>
              </div>

              {/* Attendance this term */}
              <div className="mt-4 pt-4 border-t border-neutral-100">
                <div className="flex items-center justify-between mb-1.5">
                  <p className="text-xs text-neutral-500">Term attendance</p>
                  <p className="text-xs font-semibold text-neutral-700">
                    {childData.attendanceThisTerm.present}/{childData.attendanceThisTerm.total} days
                    ({childData.attendanceThisTerm.percentage}%)
                  </p>
                </div>
                <div className="h-2 bg-neutral-100 rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full ${
                      childData.attendanceThisTerm.percentage >= 80 ? "bg-success" : "bg-warning"
                    }`}
                    style={{ width: `${childData.attendanceThisTerm.percentage}%` }}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>

        {/* Info cards row */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {/* Fee balance */}
          <Card>
            <CardContent className="p-5">
              <div className="flex items-start justify-between">
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Fee Balance</p>
                <DollarSign className="h-4 w-4 text-neutral-300" />
              </div>
              <p className={`mt-3 text-2xl font-semibold ${childData.feeBalance > 0 ? "text-error" : "text-success"}`}>
                {childData.feeBalance > 0 ? formatKES(childData.feeBalance) : "Fully Paid"}
              </p>
              {childData.feeBalance > 0 ? (
                <Button size="sm" className="mt-3 w-full" onClick={() => setShowPayModal(true)}>
                  Pay Now via M-Pesa
                </Button>
              ) : (
                <p className="text-xs text-success mt-1.5">✓ No outstanding balance</p>
              )}
            </CardContent>
          </Card>

          {/* Latest performance */}
          <Card>
            <CardContent className="p-5">
              <div className="flex items-start justify-between">
                <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Latest Results</p>
                <BookOpen className="h-4 w-4 text-neutral-300" />
              </div>
              <p className="mt-3 text-2xl font-semibold text-neutral-900">
                Position {childData.latestPosition.position}
              </p>
              <p className="text-xs text-neutral-500 mt-1">
                out of {childData.latestPosition.outOf} students · {childData.latestPosition.term}
              </p>
              <Button size="sm" variant="outline" className="mt-3 w-full" asChild>
                <a href="/parent/results">View Report Card</a>
              </Button>
            </CardContent>
          </Card>
        </div>

        {/* Recent payments */}
        {recentPayments.length > 0 && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Recent Payments</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {recentPayments.map((p) => (
                <div key={p.id} className="flex items-center justify-between px-5 py-3.5 border-b border-neutral-50 last:border-0">
                  <div>
                    <p className="text-sm font-medium text-neutral-800">{formatKES(p.amount)}</p>
                    <p className="text-xs text-neutral-500">
                      {formatDate(p.date)} · {p.method} · {p.ref}
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <StatusBadge status="paid" />
                    <Button size="sm" variant="ghost" asChild>
                      <a href={`/parent/receipts/${p.id}`}>Receipt</a>
                    </Button>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        )}

        {/* Announcements */}
        <Card>
          <CardHeader className="flex-row items-center justify-between pb-3">
            <CardTitle>Announcements</CardTitle>
            <Bell className="h-4 w-4 text-neutral-400" />
          </CardHeader>
          <CardContent className="p-0">
            {announcements.map((ann) => (
              <div
                key={ann.id}
                className={`px-5 py-4 border-b border-neutral-50 last:border-0 ${
                  !ann.isRead ? "bg-primary-50/40" : ""
                }`}
              >
                <div className="flex items-start gap-3">
                  {!ann.isRead && (
                    <div className="w-2 h-2 rounded-full bg-primary-500 mt-1.5 flex-shrink-0" />
                  )}
                  <div className="flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <p className="text-sm font-semibold text-neutral-800">{ann.title}</p>
                      <p className="text-xs text-neutral-400 whitespace-nowrap">{formatDate(ann.date)}</p>
                    </div>
                    <p className="text-sm text-neutral-600 mt-1 line-clamp-2">{ann.body}</p>
                  </div>
                </div>
              </div>
            ))}
            <div className="px-5 py-3">
              <Button size="sm" variant="outline" className="w-full" asChild>
                <a href="/parent/announcements">All Announcements</a>
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Contact school */}
        <Card className="bg-neutral-50 border-neutral-200">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-semibold text-neutral-800">Need help?</p>
                <p className="text-xs text-neutral-500 mt-0.5">Upendo Secondary School · +254 722 100 200</p>
              </div>
              <Button size="sm" variant="outline" asChild>
                <a href="/parent/messages">
                  <Phone className="h-4 w-4" />
                  Message Teacher
                </a>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}
