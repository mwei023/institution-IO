"use client";
import React, { useEffect, useState } from "react";
import { DollarSign, TrendingUp, Users, AlertCircle, Search, AlertTriangle } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { StatCard } from "@/components/shared/StatCard";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { financeAPI } from "@/lib/api";
import { formatKES, formatDate } from "@/lib/utils";

interface FeeSummary {
  total_students: number;
  total_billed: number;
  total_collected: number;
  total_outstanding: number;
  fully_paid_count: number;
  partial_count: number;
  unpaid_count: number;
}

interface StudentBalance {
  student_id: string;
  student_name: string;
  admission_number: string;
  class_name: string | null;
  total_billed: number;
  total_paid: number;
  balance: number;
  last_payment_at: string | null;
}

export default function FinancePage() {
  const [summary, setSummary] = useState<FeeSummary | null>(null);
  const [summaryError, setSummaryError] = useState(false);

  const [balances, setBalances] = useState<StudentBalance[] | null>(null);
  const [balancesError, setBalancesError] = useState(false);
  const [balancesLoading, setBalancesLoading] = useState(true);

  const [search, setSearch] = useState("");

  useEffect(() => {
    financeAPI.getSummary()
      .then((res) => setSummary(res.data?.data))
      .catch(() => setSummaryError(true));
  }, []);

  useEffect(() => {
    financeAPI.getOutstandingBalances()
      .then((res) => setBalances(res.data?.data ?? []))
      .catch(() => setBalancesError(true))
      .finally(() => setBalancesLoading(false));
  }, []);

  const collectionRate = summary
    ? ((summary.total_collected / summary.total_billed) * 100).toFixed(1)
    : "—";

  const filteredBalances = (balances ?? []).filter((b) =>
    search === "" ||
    b.student_name.toLowerCase().includes(search.toLowerCase()) ||
    (b.class_name ?? "").toLowerCase().includes(search.toLowerCase())
  );

  const totalOutstandingFromTable = (balances ?? []).reduce((sum, b) => sum + b.balance, 0);

  return (
    <DashboardLayout title="Finance">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Finance</h1>
            <p className="page-subtitle">Fee Collection Overview</p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm">Export CSV</Button>
            <Button size="sm" onClick={() => { window.location.href = "/finance/pay"; }}>
              Record Payment
            </Button>
          </div>
        </div>

        {/* Stats */}
        {summaryError ? (
          <Card className="border-error/30 bg-error-light/30">
            <CardContent className="py-4 flex items-center gap-2 text-error text-sm">
              <AlertTriangle className="h-4 w-4" />
              Could not load fee summary. Showing no data rather than estimates.
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              label="Total Billed"
              value={summary ? formatKES(summary.total_billed) : "—"}
              sub="Current term"
              icon={DollarSign}
              iconColor="text-neutral-400"
            />
            <StatCard
              label="Collected"
              value={summary ? formatKES(summary.total_collected) : "—"}
              sub={summary ? `${collectionRate}% collection rate` : "—"}
              icon={TrendingUp}
              iconColor="text-success"
            />
            <StatCard
              label="Outstanding"
              value={summary ? formatKES(summary.total_outstanding) : "—"}
              sub={summary ? `${summary.unpaid_count} fully unpaid` : "—"}
              icon={AlertCircle}
              iconColor="text-error"
            />
            <StatCard
              label="Fully Paid"
              value={summary ? `${summary.fully_paid_count}` : "—"}
              sub={summary ? `of ${summary.total_students} students` : "—"}
              icon={Users}
              iconColor="text-primary-500"
            />
          </div>
        )}

        {/* Payment status breakdown */}
        {summary && (
          <div className="grid grid-cols-3 gap-4">
            {[
              { label: "Fully Paid", count: summary.fully_paid_count, color: "bg-success-light text-success-dark" },
              { label: "Partial", count: summary.partial_count, color: "bg-warning-light text-warning-dark" },
              { label: "Not Paid", count: summary.unpaid_count, color: "bg-error-light text-error-dark" },
            ].map((s) => (
              <div key={s.label} className={`rounded-lg p-4 ${s.color}`}>
                <p className="text-2xl font-semibold">{s.count}</p>
                <p className="text-sm font-medium mt-0.5">{s.label}</p>
              </div>
            ))}
          </div>
        )}

        {/* Outstanding Balances Table — real data only */}
        <Card>
          <CardHeader className="flex-row items-center justify-between pb-3">
            <div>
              <CardTitle>Outstanding Balances</CardTitle>
              {balances && (
                <p className="text-xs text-neutral-500 mt-1">
                  {balances.length} student{balances.length === 1 ? "" : "s"} owing{" "}
                  {formatKES(totalOutstandingFromTable)}
                </p>
              )}
            </div>
            <div className="relative w-56">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-neutral-400" />
              <Input
                placeholder="Search student or class..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8 h-8 text-xs"
              />
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {balancesLoading ? (
              <div className="px-5 py-8 text-center text-sm text-neutral-500">Loading…</div>
            ) : balancesError ? (
              <div className="px-5 py-8 text-center text-sm text-error flex items-center justify-center gap-2">
                <AlertTriangle className="h-4 w-4" />
                Could not load outstanding balances. Try refreshing.
              </div>
            ) : filteredBalances.length === 0 ? (
              <div className="px-5 py-8 text-center text-sm text-neutral-500">
                {balances?.length === 0
                  ? "No students currently have an outstanding balance."
                  : "No students match your search."}
              </div>
            ) : (
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-100">
                    <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Student</th>
                    <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden sm:table-cell">Class</th>
                    <th className="text-right px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Balance</th>
                    <th className="text-left px-5 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide hidden md:table-cell">Last Payment</th>
                    <th className="px-5 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-50">
                  {filteredBalances.map((b) => (
                    <tr key={b.student_id} className="hover:bg-neutral-50">
                      <td className="px-5 py-3.5 text-sm font-medium text-neutral-800">
                        {b.student_name}
                        <span className="block text-xs text-neutral-400">{b.admission_number}</span>
                      </td>
                      <td className="px-5 py-3.5 text-sm text-neutral-600 hidden sm:table-cell">
                        {b.class_name ?? "—"}
                      </td>
                      <td className="px-5 py-3.5 text-sm font-semibold text-error text-right">
                        {formatKES(b.balance)}
                      </td>
                      <td className="px-5 py-3.5 text-sm text-neutral-500 hidden md:table-cell">
                        {b.last_payment_at ? formatDate(b.last_payment_at) : <span className="text-error">Never</span>}
                      </td>
                      <td className="px-5 py-3.5 text-right">
                        <Button size="sm" variant="outline">Send Reminder</Button>
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