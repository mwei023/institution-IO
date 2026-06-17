"use client";
import React, { useEffect, useState, useCallback } from "react";
import { DollarSign, TrendingUp, Users, AlertCircle, Search } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { StatCard } from "@/components/shared/StatCard";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { financeAPI } from "@/lib/api";
import { formatKES, formatDate } from "@/lib/utils";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";

interface FeeSummary {
  total_students:    number;
  total_billed:      number;
  total_collected:   number;
  total_outstanding: number;
  fully_paid_count:  number;
  partial_count:     number;
  unpaid_count:      number;
}

const mockByClass = [
  { class: "Form 1", billed: 2340000, collected: 1980000 },
  { class: "Form 2", billed: 2340000, collected: 2100000 },
  { class: "Form 3", billed: 2340000, collected: 1870000 },
  { class: "Form 4", billed: 2340000, collected: 2290000 },
];

const mockDefaulters = [
  { id: "s1", name: "Kioni Waweru",   class: "Form 3 East", balance: 40000, lastPayment: null },
  { id: "s2", name: "Brian Kipchoge", class: "Form 1 East", balance: 20000, lastPayment: "2024-09-24" },
  { id: "s3", name: "Robert Njenga",  class: "Form 4 East", balance: 35000, lastPayment: "2024-08-15" },
  { id: "s4", name: "Sharon Kerubo",  class: "Form 4 East", balance: 15000, lastPayment: "2024-09-10" },
];

export default function FinancePage() {
  const [summary, setSummary] = useState<FeeSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  useEffect(() => {
    financeAPI.getSummary()
      .then((res) => setSummary(res.data?.data))
      .catch(() => {
        // Mock fallback
        setSummary({
          total_students:    850,
          total_billed:      9360000,
          total_collected:   8240000,
          total_outstanding: 1120000,
          fully_paid_count:  721,
          partial_count:     87,
          unpaid_count:      42,
        });
      })
      .finally(() => setLoading(false));
  }, []);

  const collectionRate = summary
    ? ((summary.total_collected / summary.total_billed) * 100).toFixed(1)
    : "—";

  const filteredDefaulters = mockDefaulters.filter((d) =>
    search === "" || d.name.toLowerCase().includes(search.toLowerCase()) ||
    d.class.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardLayout title="Finance">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Finance</h1>
            <p className="page-subtitle">Term 3, 2024 · Fee Collection Overview</p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm">Export CSV</Button>
            <Button size="sm" asChild>
              <a href="/finance/pay">Record Payment</a>
            </Button>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            label="Total Billed"
            value={summary ? formatKES(summary.total_billed) : "—"}
            sub="Term 3 2024"
            icon={DollarSign}
            iconColor="text-neutral-400"
          />
          <StatCard
            label="Collected"
            value={summary ? formatKES(summary.total_collected) : "—"}
            sub={`${collectionRate}% collection rate`}
            icon={TrendingUp}
            iconColor="text-success"
          />
          <StatCard
            label="Outstanding"
            value={summary ? formatKES(summary.total_outstanding) : "—"}
            sub={`${summary?.unpaid_count ?? 0} fully unpaid`}
            icon={AlertCircle}
            iconColor="text-error"
          />
          <StatCard
            label="Fully Paid"
            value={summary ? `${summary.fully_paid_count}` : "—"}
            sub={`of ${summary?.total_students ?? 0} students`}
            icon={Users}
            iconColor="text-primary-500"
          />
        </div>

        {/* Payment status breakdown */}
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: "Fully Paid",    count: summary?.fully_paid_count ?? 0, color: "bg-success-light text-success-dark" },
            { label: "Partial",       count: summary?.partial_count ?? 0,    color: "bg-warning-light text-warning-dark" },
            { label: "Not Paid",      count: summary?.unpaid_count ?? 0,     color: "bg-error-light text-error-dark" },
          ].map((s) => (
            <div key={s.label} className={`rounded-lg p-4 ${s.color}`}>
              <p className="text-2xl font-semibold">{s.count}</p>
              <p className="text-sm font-medium mt-0.5">{s.label}</p>
            </div>
          ))}
        </div>

        {/* Collection by class chart */}
        <Card>
          <CardHeader>
            <CardTitle>Fee Collection by Class</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={mockByClass} barGap={6}>
                <CartesianGrid strokeDasharray="3 3" stroke="#F1F5F9" vertical={false} />
                <XAxis dataKey="class" tick={{ fontSize: 12, fill: "#94A3B8" }} axisLine={false} tickLine={false} />
                <YAxis
                  tick={{ fontSize: 12, fill: "#94A3B8" }}
                  axisLine={false}
                  tickLine={false}
                  tickFormatter={(v) => `${(v / 1000000).toFixed(1)}M`}
                />
                <Tooltip
                  formatter={(v: number) => formatKES(v)}
                  contentStyle={{ borderRadius: 8, border: "1px solid #E2E8F0", fontSize: 12 }}
                />
                <Bar dataKey="billed"    name="Billed"    fill="#DBEAFE" radius={[4,4,0,0]} />
                <Bar dataKey="collected" name="Collected" fill="#2563EB" radius={[4,4,0,0]} />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Fee defaulters table */}
        <Card>
          <CardHeader className="flex-row items-center justify-between pb-3">
            <CardTitle>Outstanding Balances</CardTitle>
            <div className="relative w-56">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-neutral-400" />
              <Input
                placeholder="Search student..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8 h-8 text-xs"
              />
            </div>
          </CardHeader>
          <CardContent className="p-0">
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
                {filteredDefaulters.map((d) => (
                  <tr key={d.id} className="hover:bg-neutral-50">
                    <td className="px-5 py-3.5 text-sm font-medium text-neutral-800">{d.name}</td>
                    <td className="px-5 py-3.5 text-sm text-neutral-600 hidden sm:table-cell">{d.class}</td>
                    <td className="px-5 py-3.5 text-sm font-semibold text-error text-right">
                      {formatKES(d.balance)}
                    </td>
                    <td className="px-5 py-3.5 text-sm text-neutral-500 hidden md:table-cell">
                      {d.lastPayment ? formatDate(d.lastPayment) : <span className="text-error">Never</span>}
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      <Button size="sm" variant="outline">Send Reminder</Button>
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
