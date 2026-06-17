"use client";
import React, { useEffect, useState } from "react";
import {
  Users, DollarSign, AlertTriangle, Stethoscope,
  Calendar, TrendingUp, CheckCircle2, Clock,
  BookOpen, MessageSquare, UserCheck,
} from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { StatCard } from "@/components/shared/StatCard";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { financeAPI } from "@/lib/api";
import { formatKES, formatDate } from "@/lib/utils";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Cell,
} from "recharts";

const mockStats = {
  studentsPresent: 743,
  studentsTotal: 850,
  attendanceRate: 87.4,
  outstandingFees: 2340000,
  incidentsWeek: 3,
  studentsInClinic: 2,
};

const mockFeeData = [
  { class: "Form 1", collected: 680000, target: 780000 },
  { class: "Form 2", collected: 720000, target: 780000 },
  { class: "Form 3", collected: 590000, target: 780000 },
  { class: "Form 4", collected: 810000, target: 780000 },
];

const mockUpcomingEvents = [
  { id: 1, title: "Form 4 Mock Exams Begin", date: "2024-10-07", type: "academic" },
  { id: 2, title: "Parent-Teacher Meeting",  date: "2024-10-12", type: "general" },
  { id: 3, title: "Term 3 Closing Day",      date: "2024-11-29", type: "academic" },
];

const mockPendingAdmissions = [
  { id: "a1", name: "David Kimani",  class: "Form 1", status: "submitted",        date: "2024-09-28" },
  { id: "a2", name: "Faith Achieng", class: "Form 1", status: "under_review",     date: "2024-09-25" },
  { id: "a3", name: "George Mwangi", class: "Form 2", status: "awaiting_payment", date: "2024-09-20" },
];

const mockTeacherAttendance = [
  { name: "Mon", present: 40, absent: 2 },
  { name: "Tue", present: 41, absent: 1 },
  { name: "Wed", present: 38, absent: 4 },
  { name: "Thu", present: 42, absent: 0 },
  { name: "Fri", present: 39, absent: 3 },
];

const quickActions = [
  { label: "New Admission",  icon: UserCheck,     href: "/admissions/new" },
  { label: "Record Payment", icon: DollarSign,    href: "/finance/pay" },
  { label: "Announcement",   icon: MessageSquare, href: "/communications/new" },
  { label: "Attendance",     icon: CheckCircle2,  href: "/academics/attendance" },
  { label: "Fee Summary",    icon: TrendingUp,    href: "/finance" },
  { label: "Pending Tasks",  icon: Clock,         href: "/tasks" },
];

export default function PrincipalDashboard() {
  const [feeSummary, setFeeSummary] = useState<Record<string, number> | null>(null);

  useEffect(() => {
    financeAPI.getSummary()
      .then((res) => setFeeSummary(res.data?.data))
      .catch(() => {});
  }, []);

  const outstandingFees = feeSummary
    ? (feeSummary.total_outstanding as number)
    : mockStats.outstandingFees;

  return (
    <DashboardLayout title="Principal Dashboard">
      <div className="space-y-6">

        {/* Page header */}
        <div className="page-header">
          <div>
            <h1 className="page-title">Good morning, Principal</h1>
            <p className="page-subtitle">
              {new Date().toLocaleDateString("en-KE", {
                weekday: "long", year: "numeric", month: "long", day: "numeric",
              })} &middot; Term 3, 2024
            </p>
          </div>
          <Button size="sm" variant="outline" onClick={() => { window.location.href = "/calendar"; }}>
            <Calendar className="h-4 w-4 mr-2" />
            View Calendar
          </Button>
        </div>

        {/* Stat cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            label="Students Present"
            value={mockStats.studentsPresent}
            sub={"of " + mockStats.studentsTotal + " enrolled \u00b7 " + mockStats.attendanceRate + "%"}
            icon={Users}
            iconColor="text-primary-500"
          />
          <StatCard
            label="Outstanding Fees"
            value={formatKES(outstandingFees)}
            sub="This term"
            icon={DollarSign}
            iconColor="text-warning"
          />
          <StatCard
            label="Incidents This Week"
            value={mockStats.incidentsWeek}
            sub="Discipline & security"
            icon={AlertTriangle}
            iconColor="text-error"
          />
          <StatCard
            label="Students in Clinic"
            value={mockStats.studentsInClinic}
            sub="1 in isolation"
            icon={Stethoscope}
            iconColor="text-info"
          />
        </div>

        {/* Charts row */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card>
            <CardHeader>
              <CardTitle>Fee Collection &mdash; Term 3</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={mockFeeData} barGap={4}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#F1F5F9" vertical={false} />
                  <XAxis
                    dataKey="class"
                    tick={{ fontSize: 12, fill: "#94A3B8" }}
                    axisLine={false}
                    tickLine={false}
                  />
                  <YAxis
                    tick={{ fontSize: 12, fill: "#94A3B8" }}
                    axisLine={false}
                    tickLine={false}
                    tickFormatter={(v: number) => v / 1000 + "K"}
                  />
                  <Tooltip
                    formatter={(v: number) => formatKES(v)}
                    contentStyle={{ borderRadius: 8, border: "1px solid #E2E8F0", fontSize: 12 }}
                  />
                  <Bar dataKey="target" name="Target" fill="#DBEAFE" radius={[4, 4, 0, 0]} />
                  <Bar dataKey="collected" name="Collected" radius={[4, 4, 0, 0]}>
                    {mockFeeData.map((entry, index) => (
                      <Cell
                        key={index}
                        fill={entry.collected >= entry.target ? "#16A34A" : "#2563EB"}
                      />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Teacher Attendance &mdash; This Week</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={mockTeacherAttendance} barGap={4}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#F1F5F9" vertical={false} />
                  <XAxis
                    dataKey="name"
                    tick={{ fontSize: 12, fill: "#94A3B8" }}
                    axisLine={false}
                    tickLine={false}
                  />
                  <YAxis
                    tick={{ fontSize: 12, fill: "#94A3B8" }}
                    axisLine={false}
                    tickLine={false}
                    domain={[0, 45]}
                  />
                  <Tooltip
                    contentStyle={{ borderRadius: 8, border: "1px solid #E2E8F0", fontSize: 12 }}
                  />
                  <Bar dataKey="present" name="Present" fill="#2563EB" radius={[4, 4, 0, 0]} />
                  <Bar dataKey="absent"  name="Absent"  fill="#FEE2E2" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </div>

        {/* Bottom row */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

          {/* Pending admissions */}
          <Card className="lg:col-span-2">
            <CardHeader className="flex-row items-center justify-between pb-3">
              <CardTitle>Pending Admissions</CardTitle>
              <a
                href="/admissions"
                className="text-sm font-medium text-primary-600 hover:underline"
              >
                View All
              </a>
            </CardHeader>
            <CardContent className="p-0">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-100">
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Name</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden sm:table-cell">Class</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Status</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden md:table-cell">Date</th>
                    <th className="px-6 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-50">
                  {mockPendingAdmissions.map((a) => (
                    <tr key={a.id} className="hover:bg-neutral-50 transition-colors">
                      <td className="px-6 py-3.5 text-sm font-medium text-neutral-800">{a.name}</td>
                      <td className="px-6 py-3.5 text-sm text-neutral-600 hidden sm:table-cell">{a.class}</td>
                      <td className="px-6 py-3.5"><StatusBadge status={a.status} /></td>
                      <td className="px-6 py-3.5 text-sm text-neutral-500 hidden md:table-cell">{formatDate(a.date)}</td>
                      <td className="px-6 py-3.5 text-right">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => { window.location.href = "/admissions/" + a.id; }}
                        >
                          Review
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </CardContent>
          </Card>

          {/* Upcoming events */}
          <Card>
            <CardHeader className="flex-row items-center justify-between pb-3">
              <CardTitle>Upcoming Events</CardTitle>
              <Button size="sm" variant="ghost" onClick={() => { window.location.href = "/calendar"; }}>
                <Calendar className="h-4 w-4" />
              </Button>
            </CardHeader>
            <CardContent className="p-0">
              <div className="divide-y divide-neutral-50">
                {mockUpcomingEvents.map((event) => (
                  <div key={event.id} className="px-6 py-3.5 flex items-start gap-3">
                    <div className="flex-shrink-0 w-8 h-8 rounded-md bg-primary-50 flex items-center justify-center">
                      {event.type === "academic"
                        ? <BookOpen    className="h-4 w-4 text-primary-600" />
                        : <MessageSquare className="h-4 w-4 text-primary-600" />
                      }
                    </div>
                    <div>
                      <p className="text-sm font-medium text-neutral-800">{event.title}</p>
                      <p className="text-xs text-neutral-500 mt-0.5">{formatDate(event.date)}</p>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Quick actions */}
        <div>
          <h3 className="text-sm font-semibold text-neutral-600 mb-3 uppercase tracking-wide">
            Quick Actions
          </h3>
          <div className="flex flex-wrap gap-3">
            {quickActions.map((action) => {
              const Icon = action.icon;
              return (
                <Button
                  key={action.label}
                  variant="outline"
                  size="sm"
                  onClick={() => { window.location.href = action.href; }}
                >
                  <Icon className="h-4 w-4 mr-2" />
                  {action.label}
                </Button>
              );
            })}
          </div>
        </div>

      </div>
    </DashboardLayout>
  );
}