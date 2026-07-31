"use client";
import React from "react";
import { BookOpen, Users, ClipboardList, MessageSquare, Clock, CheckCircle2 } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { StatCard } from "@/components/shared/StatCard";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { useAuthStore } from "@/lib/store";

const todayPeriods = [
  { time: "7:30 - 8:30",   subject: "English",     class: "Form 2 East",  room: "C4", status: "done" },
  { time: "8:30 - 9:30",   subject: "English",     class: "Form 1 North", room: "B2", status: "current" },
  { time: "10:00 - 11:00", subject: "English",     class: "Form 3 West",  room: "C4", status: "upcoming" },
  { time: "11:00 - 12:00", subject: "English Lit", class: "Form 4 East",  room: "C4", status: "upcoming" },
  { time: "2:00 - 3:00",   subject: "English",     class: "Form 2 West",  room: "B3", status: "upcoming" },
];

const pendingAssignments = [
  { id: "1", title: "Essay - Oral Literature", class: "Form 2 East", due: "Tomorrow",  submitted: 28, total: 42 },
  { id: "2", title: "Grammar Exercise 5",      class: "Form 1 North", due: "Fri",      submitted: 35, total: 40 },
  { id: "3", title: "Book Review",             class: "Form 4 East", due: "Next Mon",  submitted: 19, total: 38 },
];

const recentMessages = [
  { id: "1", from: "John Odhiambo (Amara's parent)", body: "Good morning. Is my daughter in school today?", time: "8:12 AM", unread: true },
  { id: "2", from: "Sarah Waweru (Kioni's parent)",  body: "Thank you for the report card feedback.",       time: "Yesterday", unread: false },
];

export default function TeacherDashboard() {
  const { user } = useAuthStore();
  const currentPeriod = todayPeriods.find((p) => p.status === "current");
  const nextPeriod = todayPeriods.find((p) => p.status === "upcoming");

  return (
    <DashboardLayout title="Teacher Dashboard">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Good morning, {user?.full_name.split(" ")[0]}</h1>
            <p className="page-subtitle">Term 3, 2024 - Week 4</p>
          </div>
        </div>

        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard label="My Classes"       value="5" sub="This term"                  icon={Users}          iconColor="text-primary-500" />
          <StatCard label="Attendance Done"  value="3/5" sub="Today's sessions"         icon={CheckCircle2}   iconColor="text-success" />
          <StatCard label="Assignments Open" value={pendingAssignments.length} sub="Pending marking" icon={ClipboardList} iconColor="text-warning" />
          <StatCard label="Parent Messages"  value="1" sub="Unread today"               icon={MessageSquare}  iconColor="text-info" />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <Card className="lg:col-span-2">
            <CardHeader className="flex-row items-center justify-between pb-3">
              <CardTitle>Today&apos;s Timetable</CardTitle>
              <Button size="sm" variant="outline" onClick={() => { window.location.href = "/teacher/attendance"; }}>
                Take Attendance
              </Button>
            </CardHeader>
            <CardContent className="p-0">
              <div className="divide-y divide-neutral-50">
                {todayPeriods.map((period, i) => (
                  <div
                    key={i}
                    className={`flex items-center gap-4 px-6 py-3.5 ${
                      period.status === "current" ? "bg-primary-50" : ""
                    }`}
                  >
                    <div className="w-24 flex-shrink-0">
                      <p className="text-xs font-medium text-neutral-500">{period.time}</p>
                    </div>
                    <div className="flex-1">
                      <p className="text-sm font-medium text-neutral-800">{period.subject}</p>
                      <p className="text-xs text-neutral-500">{period.class} - Room {period.room}</p>
                    </div>
                    <div>
                      {period.status === "current" && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-primary-100 text-primary-700">
                          <span className="w-1.5 h-1.5 rounded-full bg-primary-500 animate-pulse" />
                          Now
                        </span>
                      )}
                      {period.status === "done" && (
                        <CheckCircle2 className="h-4 w-4 text-success" />
                      )}
                      {period.status === "upcoming" && (
                        <Clock className="h-4 w-4 text-neutral-300" />
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <div className="space-y-4">
            {currentPeriod && (
              <Card className="border-primary-200 bg-primary-50">
                <CardContent className="p-4">
                  <p className="text-xs font-medium text-primary-600 uppercase tracking-wide mb-2">
                    Current Lesson
                  </p>
                  <p className="text-base font-semibold text-neutral-900">{currentPeriod.subject}</p>
                  <p className="text-sm text-neutral-600">{currentPeriod.class}</p>
                  <p className="text-xs text-neutral-500 mt-1">Room {currentPeriod.room} - {currentPeriod.time}</p>
                  <Button size="sm" className="mt-3 w-full" onClick={() => { window.location.href = "/teacher/attendance"; }}>
                    Mark Attendance
                  </Button>
                </CardContent>
              </Card>
            )}

            <Card>
              <CardHeader className="pb-3">
                <CardTitle>Parent Messages</CardTitle>
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
            <Button size="sm" variant="outline" onClick={() => { window.location.href = "/teacher/assignments"; }}>
              All Assignments
            </Button>
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