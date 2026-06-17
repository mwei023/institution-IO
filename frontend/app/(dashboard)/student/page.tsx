"use client";
import React from "react";
import { BookOpen, Clock, CheckCircle2, DollarSign, Library, Bell } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { useAuthStore } from "@/lib/store";
import { formatDate, formatKES } from "@/lib/utils";

const todayTimetable = [
  { time: "7:30 – 8:30",   subject: "English",     teacher: "Mr. Odhiambo", room: "C4",  status: "done" },
  { time: "8:30 – 9:30",   subject: "Mathematics", teacher: "Ms. Njoroge",  room: "B1",  status: "current" },
  { time: "10:00 – 11:00", subject: "Biology",     teacher: "Mr. Mugo",     room: "Lab2",status: "upcoming" },
  { time: "11:00 – 12:00", subject: "Physics",     teacher: "Mr. Kariuki",  room: "Lab1",status: "upcoming" },
  { time: "2:00 – 3:00",   subject: "History",     teacher: "Ms. Otieno",   room: "A3",  status: "upcoming" },
];

const recentResults = [
  { subject: "English",     cat1: 68, cat2: 72, midterm: 70, total: 70, grade: "B-" },
  { subject: "Mathematics", cat1: 85, cat2: 82, midterm: 88, total: 85, grade: "A-" },
  { subject: "Biology",     cat1: 74, cat2: 78, midterm: 76, total: 76, grade: "B" },
  { subject: "Physics",     cat1: 62, cat2: 65, midterm: 68, total: 65, grade: "B-" },
  { subject: "Chemistry",   cat1: 71, cat2: 74, midterm: 73, total: 73, grade: "B" },
];

const pendingAssignments = [
  { id: "1", title: "Essay — Oral Literature", subject: "English",     dueDate: "Tomorrow",   status: "not_submitted" },
  { id: "2", title: "Algebra Problem Set 6",   subject: "Mathematics", dueDate: "2024-10-04", status: "submitted" },
];

const libraryBorrowings = [
  { title: "An Introduction to Biology", author: "K. Kimani", dueDate: "2024-10-10", status: "borrowed" },
];

export default function StudentDashboard() {
  const { user } = useAuthStore();
  const currentClass = todayTimetable.find((p) => p.status === "current");

  return (
    <DashboardLayout title="Student Portal">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Welcome, {user?.full_name.split(" ")[0] ?? "Student"}</h1>
            <p className="page-subtitle">
              Form 1 East · {new Date().toLocaleDateString("en-KE", {
                weekday: "long", day: "numeric", month: "long",
              })}
            </p>
          </div>
        </div>

        {/* Current lesson callout */}
        {currentClass && (
          <Card className="border-primary-200 bg-primary-50">
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-md bg-primary-600 flex items-center justify-center">
                    <BookOpen className="h-4 w-4 text-white" />
                  </div>
                  <div>
                    <p className="text-xs font-medium text-primary-600 uppercase tracking-wide">Now in Class</p>
                    <p className="text-sm font-semibold text-neutral-900">{currentClass.subject}</p>
                    <p className="text-xs text-neutral-500">
                      {currentClass.teacher} · Room {currentClass.room} · {currentClass.time}
                    </p>
                  </div>
                </div>
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-primary-100 text-primary-700">
                  <span className="w-1.5 h-1.5 rounded-full bg-primary-500 animate-pulse" />
                  Live
                </span>
              </div>
            </CardContent>
          </Card>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Today's timetable */}
          <Card className="lg:col-span-2">
            <CardHeader className="flex-row items-center justify-between pb-3">
              <CardTitle>Today&apos;s Schedule</CardTitle>
              <Button size="sm" variant="outline" asChild>
                <a href="/student/timetable">Full Timetable</a>
              </Button>
            </CardHeader>
            <CardContent className="p-0">
              {todayTimetable.map((period, i) => (
                <div
                  key={i}
                  className={`flex items-center gap-4 px-5 py-3.5 border-b border-neutral-50 last:border-0 ${
                    period.status === "current" ? "bg-primary-50" : ""
                  }`}
                >
                  <div className="w-24 flex-shrink-0">
                    <p className="text-xs text-neutral-500">{period.time}</p>
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-neutral-800">{period.subject}</p>
                    <p className="text-xs text-neutral-500">{period.teacher} · Room {period.room}</p>
                  </div>
                  {period.status === "done" && <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />}
                  {period.status === "current" && (
                    <span className="text-xs font-medium text-primary-600 bg-primary-100 px-2 py-0.5 rounded-full">Now</span>
                  )}
                  {period.status === "upcoming" && <Clock className="h-4 w-4 text-neutral-300 flex-shrink-0" />}
                </div>
              ))}
            </CardContent>
          </Card>

          {/* Right column */}
          <div className="space-y-4">
            {/* Fee balance */}
            <Card>
              <CardContent className="p-5">
                <div className="flex items-start justify-between mb-3">
                  <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">Fee Balance</p>
                  <DollarSign className="h-4 w-4 text-neutral-300" />
                </div>
                <p className="text-2xl font-semibold text-success">Fully Paid</p>
                <p className="text-xs text-neutral-500 mt-1">Term 3, 2024</p>
                <Button size="sm" variant="outline" className="mt-3 w-full" asChild>
                  <a href="/student/fees">Payment History</a>
                </Button>
              </CardContent>
            </Card>

            {/* Library */}
            <Card>
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <CardTitle>Library</CardTitle>
                  <Library className="h-4 w-4 text-neutral-400" />
                </div>
              </CardHeader>
              <CardContent className="pt-0">
                {libraryBorrowings.length === 0 ? (
                  <p className="text-xs text-neutral-500">No books currently borrowed.</p>
                ) : (
                  libraryBorrowings.map((book, i) => (
                    <div key={i} className="text-sm">
                      <p className="font-medium text-neutral-800 truncate">{book.title}</p>
                      <p className="text-xs text-neutral-500">{book.author}</p>
                      <p className="text-xs text-warning mt-0.5">Due: {formatDate(book.dueDate)}</p>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>
        </div>

        {/* Assignments */}
        <Card>
          <CardHeader className="flex-row items-center justify-between pb-3">
            <CardTitle>My Assignments</CardTitle>
            <Button size="sm" variant="outline" asChild>
              <a href="/student/assignments">All</a>
            </Button>
          </CardHeader>
          <CardContent className="p-0">
            {pendingAssignments.map((a) => (
              <div key={a.id} className="flex items-center gap-4 px-5 py-3.5 border-b border-neutral-50 last:border-0">
                <div className="flex-1">
                  <p className="text-sm font-medium text-neutral-800">{a.title}</p>
                  <p className="text-xs text-neutral-500">{a.subject} · Due: {a.dueDate}</p>
                </div>
                <StatusBadge status={a.status} />
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Recent grades */}
        <Card>
          <CardHeader className="flex-row items-center justify-between pb-3">
            <CardTitle>Term 2 Results</CardTitle>
            <Button size="sm" variant="outline" asChild>
              <a href="/student/results">Full Report</a>
            </Button>
          </CardHeader>
          <CardContent className="p-0">
            <table className="w-full">
              <thead>
                <tr className="border-b border-neutral-100">
                  <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Subject</th>
                  <th className="text-right px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden sm:table-cell">CAT 1</th>
                  <th className="text-right px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden sm:table-cell">CAT 2</th>
                  <th className="text-right px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden md:table-cell">Midterm</th>
                  <th className="text-right px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Total</th>
                  <th className="text-right px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Grade</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-50">
                {recentResults.map((r) => (
                  <tr key={r.subject} className="hover:bg-neutral-50">
                    <td className="px-5 py-3 text-sm font-medium text-neutral-800">{r.subject}</td>
                    <td className="px-5 py-3 text-sm text-neutral-600 text-right hidden sm:table-cell">{r.cat1}</td>
                    <td className="px-5 py-3 text-sm text-neutral-600 text-right hidden sm:table-cell">{r.cat2}</td>
                    <td className="px-5 py-3 text-sm text-neutral-600 text-right hidden md:table-cell">{r.midterm}</td>
                    <td className="px-5 py-3 text-sm font-semibold text-neutral-800 text-right">{r.total}</td>
                    <td className="px-5 py-3 text-right">
                      <span className={`text-xs font-semibold px-2 py-0.5 rounded ${
                        r.total >= 80 ? "bg-success-light text-success-dark" :
                        r.total >= 60 ? "bg-primary-50 text-primary-700" :
                        "bg-warning-light text-warning-dark"
                      }`}>
                        {r.grade}
                      </span>
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
