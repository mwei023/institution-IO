"use client";
import React, { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, User, DollarSign, BookOpen, Activity, FileText, Edit, AlertTriangle, Link2, Unlink } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { studentsAPI, financeAPI } from "@/lib/api";
import { formatDate, computeAge, getInitials, formatKES } from "@/lib/utils";

interface StudentProfile {
  id: string;
  admission_number: string;
  first_name: string;
  last_name: string;
  date_of_birth: string;
  gender: string;
  nationality: string;
  blood_group: string | null;
  status: string;
  admission_date: string;
  is_boarder: boolean;
  uses_transport: boolean;
  medical_conditions: string | null;
  allergies: string | null;
}

interface TimelineEvent {
  id: string;
  event_type: string;
  title: string;
  description: string | null;
  event_date: string;
}

interface FeeInvoice {
  id: string;
  term_id: string;
  invoice_number: string;
  total_amount: number;
  paid_amount: number;
  balance: number;
  status: string; // unpaid|partial|paid|waived|cancelled
  due_date: string | null;
  created_at: string;
}

// Real allocation state — replaces the old reliance on fee_payments.invoice_id,
// which is a static column set once at creation and never updated. This is
// the actual source of truth for "what invoice(s) is this payment applied to",
// and it correctly reflects deallocation.
interface PaymentAllocation {
  id: string;
  invoice_id: string;
  amount_applied: number;
  is_reversed: boolean;
  reversed_at: string | null;
  reversal_reason: string | null;
  created_at: string;
}

interface FeePayment {
  id: string;
  invoice_id: string | null; // stale — displayed nowhere in this page anymore
  receipt_number: string;
  amount: number;
  payment_method: string;
  mpesa_reference: string | null;
  bank_reference: string | null;
  received_at: string;
  is_record_only: boolean;
  allocations: PaymentAllocation[];
  unapplied_amount: number;
}

interface FeeAccount {
  student_id: string;
  student_name: string;
  total_billed: number;
  total_paid: number;
  balance: number;
  last_payment: string | null;
  invoices: FeeInvoice[];
}

const TABS = ["Overview", "Finance", "Academics", "Timeline", "Documents"] as const;
type Tab = typeof TABS[number];

const eventTypeIcon: Record<string, string> = {
  enrolled:           "🎓",
  fee_paid:           "💰",
  absent:             "📅",
  discipline_incident:"⚠️",
  clinic_visit:       "🏥",
  library_borrowed:   "📚",
  class_promoted:     "⬆️",
  graduated:          "🏆",
};

const statusColor: Record<string, string> = {
  paid: "text-success",
  partial: "text-warning-dark",
  unpaid: "text-error",
  waived: "text-neutral-400",
  cancelled: "text-neutral-400",
};

export default function StudentProfilePage() {
  const params = useParams();
  const router = useRouter();
  const studentId = params.id as string;

  const [student, setStudent] = useState<StudentProfile | null>(null);
  const [timeline, setTimeline] = useState<TimelineEvent[]>([]);
  const [feeAccount, setFeeAccount] = useState<FeeAccount | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>("Overview");
  const [loading, setLoading] = useState(true);

  const [showStatement, setShowStatement] = useState(false);
  const [payments, setPayments] = useState<FeePayment[] | null>(null);
  const [paymentsLoading, setPaymentsLoading] = useState(false);
  const [paymentsError, setPaymentsError] = useState(false);

  // Allocate mini-form — which payment's form is open, and its fields
  const [allocatingPaymentId, setAllocatingPaymentId] = useState<string | null>(null);
  const [allocateInvoiceId, setAllocateInvoiceId] = useState("");
  const [allocateAmount, setAllocateAmount] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionBusy, setActionBusy] = useState<string | null>(null); // id of the payment/allocation being acted on

  const reload = () => {
    financeAPI.getFeeAccount(studentId)
      .then((res) => setFeeAccount(res.data?.data))
      .catch(() => {});
    financeAPI.getPaymentHistory(studentId)
      .then((res) => setPayments(res.data?.data ?? []))
      .catch(() => setPaymentsError(true));
  };

  useEffect(() => {
    const load = async () => {
      try {
        const [sRes, tRes, fRes] = await Promise.allSettled([
          studentsAPI.get(studentId),
          studentsAPI.getTimeline(studentId),
          financeAPI.getFeeAccount(studentId),
        ]);
        if (sRes.status === "fulfilled") setStudent(sRes.value.data?.data);
        if (tRes.status === "fulfilled") setTimeline(tRes.value.data?.data ?? []);
        if (fRes.status === "fulfilled") setFeeAccount(fRes.value.data?.data);
      } catch {
        setStudent(mockStudent);
        setTimeline(mockTimeline);
        setFeeAccount({
          student_id: studentId,
          student_name: `${mockStudent.first_name} ${mockStudent.last_name}`,
          total_billed: 19500,
          total_paid: 19500,
          balance: 0,
          last_payment: null,
          invoices: [],
        });
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [studentId]);

  const handleViewStatement = () => {
    const next = !showStatement;
    setShowStatement(next);
    if (next && payments === null && !paymentsLoading) {
      setPaymentsLoading(true);
      setPaymentsError(false);
      financeAPI.getPaymentHistory(studentId)
        .then((res) => setPayments(res.data?.data ?? []))
        .catch(() => setPaymentsError(true))
        .finally(() => setPaymentsLoading(false));
    }
  };

  const openAllocateForm = (paymentId: string) => {
    setAllocatingPaymentId(paymentId);
    setAllocateInvoiceId("");
    setAllocateAmount("");
    setActionError(null);
  };

  const handleAllocate = async (paymentId: string) => {
    setActionError(null);
    if (!allocateInvoiceId) {
      setActionError("Choose an invoice to apply this credit to.");
      return;
    }
    setActionBusy(paymentId);
    try {
      await financeAPI.allocatePayment(paymentId, {
        invoice_id: allocateInvoiceId,
        amount: allocateAmount ? Number(allocateAmount) : undefined,
      });
      setAllocatingPaymentId(null);
      reload();
    } catch (err: any) {
      setActionError(err?.response?.data?.error?.message ?? "Could not allocate this payment.");
    } finally {
      setActionBusy(null);
    }
  };

  const handleDeallocate = async (allocationId: string) => {
    setActionError(null);
    setActionBusy(allocationId);
    try {
      await financeAPI.deallocatePayment(allocationId, "Deallocated from student statement");
      reload();
    } catch (err: any) {
      setActionError(err?.response?.data?.error?.message ?? "Could not deallocate this payment.");
    } finally {
      setActionBusy(null);
    }
  };

  if (loading) {
    return (
      <DashboardLayout>
        <div className="space-y-4">
          <div className="h-8 w-48 bg-neutral-100 rounded animate-pulse" />
          <div className="h-32 bg-neutral-100 rounded-lg animate-pulse" />
        </div>
      </DashboardLayout>
    );
  }

  if (!student) {
    return (
      <DashboardLayout>
        <div className="text-center py-12">
          <p className="text-neutral-500">Student not found.</p>
          <Button variant="outline" className="mt-4" onClick={() => router.back()}>Go Back</Button>
        </div>
      </DashboardLayout>
    );
  }

  const fullName = `${student.first_name} ${student.last_name}`;
  const outstandingInvoices = (feeAccount?.invoices ?? []).filter(
    (inv) => inv.status === "unpaid" || inv.status === "partial"
  );

  return (
    <DashboardLayout title={fullName}>
      <div className="space-y-5">
        <div className="flex items-center justify-between">
          <button
            onClick={() => router.back()}
            className="flex items-center gap-2 text-sm text-neutral-500 hover:text-neutral-800 transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Students
          </button>
          <div className="flex gap-2">
            <Button variant="outline" size="sm">
              <Edit className="h-4 w-4" />
              Edit Profile
            </Button>
          </div>
        </div>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-start gap-5">
              <div className="w-16 h-16 rounded-full bg-primary-100 flex items-center justify-center flex-shrink-0">
                <span className="text-xl font-semibold text-primary-700">{getInitials(fullName)}</span>
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="text-xl font-semibold text-neutral-900">{fullName}</h2>
                    <p className="text-sm text-neutral-500 mt-0.5 font-mono">{student.admission_number}</p>
                  </div>
                  <StatusBadge status={student.status} />
                </div>
                <div className="mt-4 grid grid-cols-2 sm:grid-cols-4 gap-4">
                  {[
                    { label: "Class", value: "Form 1 East" },
                    { label: "Age", value: `${computeAge(student.date_of_birth)} yrs` },
                    { label: "Gender", value: student.gender === "M" ? "Male" : "Female" },
                    { label: "Type", value: student.is_boarder ? "Boarder" : "Day Scholar" },
                  ].map((item) => (
                    <div key={item.label}>
                      <p className="text-xs text-neutral-400 uppercase tracking-wide">{item.label}</p>
                      <p className="text-sm font-medium text-neutral-800 mt-0.5">{item.value}</p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <div className="flex items-center gap-0.5 border-b border-neutral-200 overflow-x-auto">
          {TABS.map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2.5 text-sm font-medium whitespace-nowrap border-b-2 transition-colors ${
                activeTab === tab
                  ? "border-primary-600 text-primary-700"
                  : "border-transparent text-neutral-500 hover:text-neutral-700"
              }`}
            >
              {tab}
            </button>
          ))}
        </div>

        {activeTab === "Overview" && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card>
              <CardHeader><CardTitle>Personal Information</CardTitle></CardHeader>
              <CardContent className="space-y-3">
                {[
                  { label: "Full Name",        value: fullName },
                  { label: "Date of Birth",    value: formatDate(student.date_of_birth) },
                  { label: "Nationality",      value: student.nationality },
                  { label: "Blood Group",      value: student.blood_group ?? "Not recorded" },
                  { label: "Admission Date",   value: formatDate(student.admission_date) },
                ].map((item) => (
                  <div key={item.label} className="flex justify-between py-1.5 border-b border-neutral-50 last:border-0">
                    <span className="text-sm text-neutral-500">{item.label}</span>
                    <span className="text-sm font-medium text-neutral-800">{item.value}</span>
                  </div>
                ))}
              </CardContent>
            </Card>

            <Card>
              <CardHeader><CardTitle>Medical Summary</CardTitle></CardHeader>
              <CardContent className="space-y-3">
                <div className="py-1.5">
                  <p className="text-sm text-neutral-500">Known Conditions</p>
                  <p className="text-sm font-medium text-neutral-800 mt-1">
                    {student.medical_conditions ?? "None recorded"}
                  </p>
                </div>
                <div className="py-1.5 border-t border-neutral-50">
                  <p className="text-sm text-neutral-500">Allergies</p>
                  <p className="text-sm font-medium text-neutral-800 mt-1">
                    {student.allergies ?? "None recorded"}
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        )}

        {activeTab === "Finance" && (
          <Card>
            <CardHeader><CardTitle>Fee Account</CardTitle></CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-6 mb-6">
                <div>
                  <p className="text-xs text-neutral-500 uppercase tracking-wide">Total Paid</p>
                  <p className="text-2xl font-semibold text-success mt-1">
                    {formatKES(feeAccount?.total_paid ?? 0)}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-neutral-500 uppercase tracking-wide">Balance Due</p>
                  <p className={`text-2xl font-semibold mt-1 ${(feeAccount?.balance ?? 0) > 0 ? "text-error" : "text-success"}`}>
                    {(feeAccount?.balance ?? 0) > 0 ? formatKES(feeAccount!.balance) : "Fully Paid"}
                  </p>
                </div>
              </div>
              <Button variant="outline" size="sm" onClick={handleViewStatement}>
                {showStatement ? "Hide Full Statement" : "View Full Statement"}
              </Button>

              {showStatement && (
                <div className="mt-6 space-y-6">
                  <div>
                    <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide mb-2">
                      Invoices
                    </p>
                    {(feeAccount?.invoices ?? []).length === 0 ? (
                      <p className="text-sm text-neutral-400">No invoices on record.</p>
                    ) : (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="text-xs text-neutral-500 uppercase tracking-wide border-b border-neutral-100">
                            <th className="text-left py-1.5">Invoice #</th>
                            <th className="text-left py-1.5">Status</th>
                            <th className="text-right py-1.5">Total</th>
                            <th className="text-right py-1.5">Paid</th>
                            <th className="text-right py-1.5">Balance</th>
                            <th className="text-left py-1.5">Date</th>
                          </tr>
                        </thead>
                        <tbody>
                          {feeAccount!.invoices.map((inv) => (
                            <tr key={inv.id} className="border-b border-neutral-50 last:border-0">
                              <td className="py-1.5 font-mono text-xs">{inv.invoice_number}</td>
                              <td className={`py-1.5 capitalize ${statusColor[inv.status] ?? ""}`}>{inv.status}</td>
                              <td className="py-1.5 text-right">{formatKES(inv.total_amount)}</td>
                              <td className="py-1.5 text-right text-success">{formatKES(inv.paid_amount)}</td>
                              <td className="py-1.5 text-right">{formatKES(inv.balance)}</td>
                              <td className="py-1.5 text-neutral-500">{formatDate(inv.created_at)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}
                  </div>

                  <div>
                    <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide mb-2">
                      Payment History
                    </p>
                    {actionError && (
                      <div className="text-xs text-error flex items-center gap-1.5 mb-2">
                        <AlertTriangle className="h-3.5 w-3.5" /> {actionError}
                      </div>
                    )}
                    {paymentsLoading ? (
                      <p className="text-sm text-neutral-400">Loading…</p>
                    ) : paymentsError ? (
                      <div className="text-sm text-error flex items-center gap-2">
                        <AlertTriangle className="h-4 w-4" /> Could not load payment history.
                      </div>
                    ) : (payments ?? []).length === 0 ? (
                      <p className="text-sm text-neutral-400">No payments recorded.</p>
                    ) : (
                      <div className="space-y-2">
                        {payments!.map((p) => {
                          const activeAllocations = (p.allocations ?? []).filter((a) => !a.is_reversed);
                          const canAllocate = !p.is_record_only && p.unapplied_amount > 0;
                          return (
                            <div key={p.id} className="border border-neutral-100 rounded-md p-3">
                              <div className="flex items-center justify-between flex-wrap gap-2">
                                <div>
                                  <span className="font-mono text-xs text-neutral-500">{p.receipt_number}</span>
                                  <span className="text-sm capitalize ml-2">{p.payment_method.replace("_", " ")}</span>
                                  {p.is_record_only && (
                                    <span className="text-xs text-neutral-400 ml-2">(no financial effect)</span>
                                  )}
                                </div>
                                <div className="flex items-center gap-3">
                                  <span className="text-sm font-medium">{formatKES(p.amount)}</span>
                                  <span className="text-xs text-neutral-400">{formatDate(p.received_at)}</span>
                                </div>
                              </div>

                              {!p.is_record_only && (
                                <div className="mt-2 space-y-1">
                                  {activeAllocations.length === 0 ? (
                                    <p className="text-xs text-warning-dark">Fully unallocated — sitting as account credit</p>
                                  ) : (
                                    activeAllocations.map((a) => {
                                      const inv = (feeAccount?.invoices ?? []).find((i) => i.id === a.invoice_id);
                                      return (
                                        <div key={a.id} className="flex items-center justify-between text-xs bg-neutral-50 rounded px-2 py-1">
                                          <span className="text-neutral-600">
                                            <Link2 className="h-3 w-3 inline mr-1 text-success" />
                                            {formatKES(a.amount_applied)} → {inv?.invoice_number ?? a.invoice_id}
                                          </span>
                                          <button
                                            onClick={() => handleDeallocate(a.id)}
                                            disabled={actionBusy === a.id}
                                            className="text-error hover:underline flex items-center gap-1 disabled:opacity-50"
                                          >
                                            <Unlink className="h-3 w-3" />
                                            {actionBusy === a.id ? "Removing…" : "Deallocate"}
                                          </button>
                                        </div>
                                      );
                                    })
                                  )}
                                  {p.unapplied_amount > 0 && (
                                    <p className="text-xs text-neutral-500">
                                      {formatKES(p.unapplied_amount)} unapplied
                                    </p>
                                  )}
                                </div>
                              )}

                              {canAllocate && (
                                allocatingPaymentId === p.id ? (
                                  <div className="mt-2 flex flex-wrap items-end gap-2 bg-neutral-50 rounded p-2">
                                    <div>
                                      <label className="text-xs text-neutral-500 block mb-0.5">Invoice</label>
                                      <select
                                        value={allocateInvoiceId}
                                        onChange={(e) => setAllocateInvoiceId(e.target.value)}
                                        className="h-8 text-xs border border-neutral-200 rounded px-2"
                                      >
                                        <option value="">Select…</option>
                                        {outstandingInvoices.map((inv) => (
                                          <option key={inv.id} value={inv.id}>
                                            {inv.invoice_number} — bal {formatKES(inv.balance)}
                                          </option>
                                        ))}
                                      </select>
                                    </div>
                                    <div>
                                      <label className="text-xs text-neutral-500 block mb-0.5">
                                        Amount (max {formatKES(p.unapplied_amount)})
                                      </label>
                                      <Input
                                        type="number"
                                        placeholder={String(p.unapplied_amount)}
                                        value={allocateAmount}
                                        onChange={(e) => setAllocateAmount(e.target.value)}
                                        className="h-8 text-xs w-32"
                                      />
                                    </div>
                                    <Button size="sm" onClick={() => handleAllocate(p.id)} disabled={actionBusy === p.id}>
                                      {actionBusy === p.id ? "Allocating…" : "Confirm"}
                                    </Button>
                                    <Button size="sm" variant="outline" onClick={() => setAllocatingPaymentId(null)}>
                                      Cancel
                                    </Button>
                                  </div>
                                ) : (
                                  <button
                                    onClick={() => openAllocateForm(p.id)}
                                    className="mt-2 text-xs text-primary-600 hover:underline flex items-center gap-1"
                                  >
                                    <Link2 className="h-3 w-3" /> Allocate to invoice
                                  </button>
                                )
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {activeTab === "Timeline" && (
          <Card>
            <CardHeader><CardTitle>Student Journey</CardTitle></CardHeader>
            <CardContent>
              {timeline.length === 0 ? (
                <p className="text-sm text-neutral-500 py-4 text-center">No timeline events yet.</p>
              ) : (
                <div className="relative">
                  <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-neutral-100" />
                  <div className="space-y-4">
                    {timeline.map((event) => (
                      <div key={event.id} className="flex items-start gap-4 pl-10 relative">
                        <div className="absolute left-0 w-8 h-8 rounded-full bg-neutral-100 flex items-center justify-center text-sm flex-shrink-0">
                          {eventTypeIcon[event.event_type] ?? "📌"}
                        </div>
                        <div className="flex-1 pb-4">
                          <p className="text-sm font-medium text-neutral-800">{event.title}</p>
                          {event.description && (
                            <p className="text-xs text-neutral-500 mt-0.5">{event.description}</p>
                          )}
                          <p className="text-xs text-neutral-400 mt-1">{formatDate(event.event_date)}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {activeTab === "Documents" && (
          <Card>
            <CardHeader className="flex-row items-center justify-between">
              <CardTitle>Digital Locker</CardTitle>
              <Button size="sm"><FileText className="h-4 w-4" />Upload</Button>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-neutral-500 py-4 text-center">
                Document management coming in Phase 2.
              </p>
            </CardContent>
          </Card>
        )}

        {(activeTab === "Academics") && (
          <Card>
            <CardHeader><CardTitle>Academic Records</CardTitle></CardHeader>
            <CardContent>
              <p className="text-sm text-neutral-500 py-4 text-center">
                Full academic history coming in Phase 2.
              </p>
            </CardContent>
          </Card>
        )}
      </div>
    </DashboardLayout>
  );
}

const mockStudent: StudentProfile = {
  id: "st000001-0000-0000-0000-000000000001",
  admission_number: "2024/F1E/0001",
  first_name: "Amara",
  last_name: "Odhiambo",
  date_of_birth: "2009-03-15",
  gender: "F",
  nationality: "Kenyan",
  blood_group: "O+",
  status: "active",
  admission_date: "2024-01-08",
  is_boarder: false,
  uses_transport: false,
  medical_conditions: null,
  allergies: null,
};

const mockTimeline: TimelineEvent[] = [
  { id: "1", event_type: "enrolled",  title: "Enrolled at Upendo Secondary School", description: "Enrolled in Form 1 East", event_date: "2024-01-08" },
  { id: "2", event_type: "fee_paid",  title: "Fee payment received",               description: "KES 19,500 · Term 3 2024",  event_date: "2024-09-24" },
];