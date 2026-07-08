"use client";
import React, { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Plus, Search, ChevronRight } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { EmptyState } from "@/components/shared/EmptyState";
import { admissionsAPI } from "@/lib/api";
import { formatDate } from "@/lib/utils";

interface Admission {
  id: string;
  application_number: string;
  first_name: string;
  last_name: string;
  intended_class: string;
  status: string;
  academic_year: string;
  created_at: string;
}

const STATUS_FILTERS = [
  { key: "",                  label: "All" },
  { key: "submitted",        label: "Submitted" },
  { key: "under_review",     label: "Under Review" },
  { key: "approved",         label: "Approved" },
  { key: "awaiting_payment", label: "Awaiting Payment" },
  { key: "enrolled",         label: "Enrolled" },
  { key: "rejected",         label: "Rejected" },
];

const mockAdmissions: Admission[] = [
  { id: "a1", application_number: "APP/2024/0001", first_name: "David",  last_name: "Kimani",  intended_class: "Form 1", status: "submitted",         academic_year: "2024", created_at: "2024-09-28" },
  { id: "a2", application_number: "APP/2024/0002", first_name: "Faith",  last_name: "Achieng", intended_class: "Form 1", status: "under_review",       academic_year: "2024", created_at: "2024-09-25" },
  { id: "a3", application_number: "APP/2024/0003", first_name: "George", last_name: "Mwangi",  intended_class: "Form 2", status: "awaiting_payment",   academic_year: "2024", created_at: "2024-09-20" },
  { id: "a4", application_number: "APP/2024/0004", first_name: "Hannah", last_name: "Njeri",   intended_class: "Form 1", status: "approved",            academic_year: "2024", created_at: "2024-09-18" },
  { id: "a5", application_number: "APP/2024/0005", first_name: "Ivan",   last_name: "Koech",   intended_class: "Form 1", status: "rejected",            academic_year: "2024", created_at: "2024-09-10" },
];

export default function AdmissionsPage() {
  const router = useRouter();
  const [admissions, setAdmissions] = useState<Admission[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [page, setPage] = useState(1);

  const fetchAdmissions = useCallback(async () => {
    setLoading(true);
    try {
      const res = await admissionsAPI.list({
        status: statusFilter || undefined,
        page,
        page_size: 50,
      });
      setAdmissions(res.data?.data ?? []);
      setTotal(res.data?.pagination?.total ?? 0);
    } catch {
      const filtered = mockAdmissions.filter(
        (a) => !statusFilter || a.status === statusFilter
      );
      setAdmissions(filtered);
      setTotal(filtered.length);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, page]);

  useEffect(() => {
    fetchAdmissions();
  }, [fetchAdmissions]);

  const filteredAdmissions = search
    ? admissions.filter(
        (a) =>
          `${a.first_name} ${a.last_name}`.toLowerCase().includes(search.toLowerCase()) ||
          a.application_number.toLowerCase().includes(search.toLowerCase())
      )
    : admissions;

  return (
    <DashboardLayout title="Admissions">
      <div className="space-y-5">
        <div className="page-header">
          <div>
            <h1 className="page-title">Admissions</h1>
            <p className="page-subtitle">{total} applications · Academic Year 2024</p>
          </div>
          <a href="/admissions/new">
            <Button size="sm">
              <Plus className="h-4 w-4" />
              New Application
            </Button>
          </a>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px] max-w-xs">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <Input
              placeholder="Search applicant..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
          <div className="flex items-center gap-1 overflow-x-auto">
            {STATUS_FILTERS.map((s) => (
              <button
                key={s.key}
                onClick={() => { setStatusFilter(s.key); setPage(1); }}
                className={`px-3 py-1.5 rounded-md text-sm font-medium whitespace-nowrap transition-colors ${
                  statusFilter === s.key
                    ? "bg-primary-600 text-white"
                    : "text-neutral-600 hover:bg-neutral-100"
                }`}
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>

        <Card>
          {loading ? (
            <CardContent className="p-6 space-y-3">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="h-14 bg-neutral-100 rounded animate-pulse" />
              ))}
            </CardContent>
          ) : filteredAdmissions.length === 0 ? (
            <CardContent className="p-6">
              <EmptyState
                title="No applications found"
                description="Admissions will appear here once submitted."
                action={
              <Button size="sm" asChild>
                <a href="/admissions/new">New Application</a>
              </Button>
            }

              />
            </CardContent>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-100">
                    <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Applicant</th>
                    <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden sm:table-cell">Ref No.</th>
                    <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden md:table-cell">Class</th>
                    <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Status</th>
                    <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden lg:table-cell">Date</th>
                    <th className="px-5 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-50">
                  {filteredAdmissions.map((a) => (
                    <tr
                      key={a.id}
                      className="hover:bg-neutral-50 cursor-pointer transition-colors"
                      onClick={() => router.push(`/admissions/${a.id}`)}
                    >
                      <td className="px-5 py-3.5">
                        <p className="text-sm font-medium text-neutral-800">{a.first_name} {a.last_name}</p>
                        <p className="text-xs text-neutral-400">Year {a.academic_year}</p>
                      </td>
                      <td className="px-5 py-3.5 text-sm text-neutral-600 font-mono hidden sm:table-cell">{a.application_number}</td>
                      <td className="px-5 py-3.5 text-sm text-neutral-600 hidden md:table-cell">{a.intended_class}</td>
                      <td className="px-5 py-3.5"><StatusBadge status={a.status} /></td>
                      <td className="px-5 py-3.5 text-sm text-neutral-500 hidden lg:table-cell">{formatDate(a.created_at)}</td>
                      <td className="px-5 py-3.5 text-right">
                        <ChevronRight className="h-4 w-4 text-neutral-300 ml-auto" />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      </div>
    </DashboardLayout>
  );
}
