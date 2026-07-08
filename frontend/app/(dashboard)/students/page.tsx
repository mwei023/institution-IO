"use client";
import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Search, Plus, Download, Filter } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { EmptyState } from "@/components/shared/EmptyState";
import { studentsAPI } from "@/lib/api";
import { formatDate, getInitials } from "@/lib/utils";

interface Student {
  id: string;
  admission_number: string;
  full_name: string;
  gender: string;
  class_name: string | null;
  status: string;
  photo_url: string | null;
  is_boarder: boolean;
}

const STATUSES = ["active", "inactive", "graduated", "transferred"];

export default function StudentsPage() {
  const router = useRouter();
  const [students, setStudents] = useState<Student[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("active");
  const [page, setPage] = useState(1);
  const pageSize = 50;

  const fetchStudents = useCallback(async () => {
    setLoading(true);
    try {
      const res = await studentsAPI.list({
        search: search || undefined,
        status: status || undefined,
        page,
        page_size: pageSize,
      });
      setStudents(res.data?.data ?? []);
      setTotal(res.data?.pagination?.total ?? 0);
    } catch (err) {
      // Use mock data if API not available
      setStudents(mockStudents);
      setTotal(mockStudents.length);
    } finally {
      setLoading(false);
    }
  }, [search, status, page]);

  useEffect(() => {
    const t = setTimeout(fetchStudents, 300);
    return () => clearTimeout(t);
  }, [fetchStudents]);

  const totalPages = Math.ceil(total / pageSize);

  return (
    <DashboardLayout title="Students">
      <div className="space-y-5">
        {/* Page header */}
        <div className="page-header">
          <div>
            <h1 className="page-title">Students</h1>
            <p className="page-subtitle">{total} students {status !== "all" ? `· ${status}` : ""}</p>
          </div>
            <div className="flex items-center gap-2">
            <Button variant="outline" size="sm">
              <Download className="h-4 w-4" />
              Export
            </Button>
            <a href="/admissions/new">
              <Button size="sm">
                <Plus className="h-4 w-4" />
                New Admission
              </Button>
            </a>
          </div>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px] max-w-xs">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <Input
              placeholder="Search by name or admission no..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
              className="pl-9"
            />
          </div>
          <div className="flex items-center gap-1">
            {STATUSES.map((s) => (
              <button
                key={s}
                onClick={() => { setStatus(s); setPage(1); }}
                className={`px-3 py-1.5 rounded-md text-sm font-medium capitalize transition-colors ${
                  status === s
                    ? "bg-primary-600 text-white"
                    : "text-neutral-600 hover:bg-neutral-100"
                }`}
              >
                {s}
              </button>
            ))}
          </div>
        </div>

        {/* Table */}
        <Card>
          {loading ? (
            <CardContent className="p-6">
              <div className="space-y-3">
                {[...Array(8)].map((_, i) => (
                  <div key={i} className="h-12 bg-neutral-100 rounded animate-pulse" />
                ))}
              </div>
            </CardContent>
          ) : students.length === 0 ? (
            <CardContent className="p-6">
              <EmptyState
                title="No students found"
                description={
                  search
                    ? `No students match "${search}". Try a different search.`
                    : "No students with this status. Add students via the admission process."
                }
                action={
                  <Button size="sm" asChild>
                    <a href="/admissions/new">Begin Admission</a>
                  </Button>
                }
              />
            </CardContent>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-neutral-100">
                      <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Student</th>
                      <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden sm:table-cell">Admission No.</th>
                      <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden md:table-cell">Class</th>
                      <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden lg:table-cell">Type</th>
                      <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Status</th>
                      <th className="px-5 py-3" />
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-50">
                    {students.map((student) => (
                      <tr
                        key={student.id}
                        className="hover:bg-neutral-50 cursor-pointer transition-colors"
                        onClick={() => router.push(`/students/${student.id}`)}
                      >
                        <td className="px-5 py-3.5">
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center flex-shrink-0">
                              <span className="text-xs font-semibold text-primary-700">
                                {getInitials(student.full_name)}
                              </span>
                            </div>
                            <div>
                              <p className="text-sm font-medium text-neutral-800">{student.full_name}</p>
                              <p className="text-xs text-neutral-400">{student.gender === "M" ? "Male" : student.gender === "F" ? "Female" : "Other"}</p>
                            </div>
                          </div>
                        </td>
                        <td className="px-5 py-3.5 text-sm text-neutral-600 font-mono hidden sm:table-cell">
                          {student.admission_number}
                        </td>
                        <td className="px-5 py-3.5 text-sm text-neutral-600 hidden md:table-cell">
                          {student.class_name ?? "—"}
                        </td>
                        <td className="px-5 py-3.5 text-sm text-neutral-500 hidden lg:table-cell">
                          {student.is_boarder ? "Boarder" : "Day Scholar"}
                        </td>
                        <td className="px-5 py-3.5">
                          <StatusBadge status={student.status} />
                        </td>
                        <td className="px-5 py-3.5 text-right">
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation();
                              router.push(`/students/${student.id}`);
                            }}
                          >
                            View
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between px-5 py-4 border-t border-neutral-100">
                  <p className="text-sm text-neutral-500">
                    Showing {(page - 1) * pageSize + 1}–{Math.min(page * pageSize, total)} of {total}
                  </p>
                  <div className="flex items-center gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={page <= 1}
                      onClick={() => setPage(p => p - 1)}
                    >
                      Previous
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={page >= totalPages}
                      onClick={() => setPage(p => p + 1)}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </Card>
      </div>
    </DashboardLayout>
  );
}

// Mock data used when API is not yet connected
const mockStudents: Student[] = [
  { id: "st000001-0000-0000-0000-000000000001", admission_number: "2024/F1E/0001", full_name: "Amara Odhiambo",  gender: "F", class_name: "Form 1 East",  status: "active", photo_url: null, is_boarder: false },
  { id: "st000001-0000-0000-0000-000000000002", admission_number: "2024/F1E/0002", full_name: "Brian Kipchoge",  gender: "M", class_name: "Form 1 East",  status: "active", photo_url: null, is_boarder: true  },
  { id: "st000001-0000-0000-0000-000000000003", admission_number: "2024/F1E/0003", full_name: "Cynthia Muthoni", gender: "F", class_name: "Form 1 East",  status: "active", photo_url: null, is_boarder: true  },
  { id: "st000001-0000-0000-0000-000000000011", admission_number: "2023/F1E/0001", full_name: "Felix Kamau",     gender: "M", class_name: "Form 2 East",  status: "active", photo_url: null, is_boarder: true  },
  { id: "st000001-0000-0000-0000-000000000012", admission_number: "2023/F1E/0002", full_name: "Gloria Achieng",  gender: "F", class_name: "Form 2 East",  status: "active", photo_url: null, is_boarder: false },
  { id: "st000001-0000-0000-0000-000000000021", admission_number: "2022/F1E/0001", full_name: "Kioni Waweru",    gender: "F", class_name: "Form 3 East",  status: "active", photo_url: null, is_boarder: true  },
  { id: "st000001-0000-0000-0000-000000000031", admission_number: "2021/F1E/0001", full_name: "Patrick Mwangi",  gender: "M", class_name: "Form 4 East",  status: "active", photo_url: null, is_boarder: true  },
  { id: "st000001-0000-0000-0000-000000000032", admission_number: "2021/F1E/0002", full_name: "Queen Otieno",    gender: "F", class_name: "Form 4 East",  status: "active", photo_url: null, is_boarder: false },
];
