"use client";
import React, { useEffect, useRef, useState } from "react";
import { Search, Smartphone, Banknote, Loader2, CheckCircle2, AlertTriangle, ArrowLeft } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { financeAPI, studentsAPI } from "@/lib/api";
import { formatKES } from "@/lib/utils";

// Matches student.StudentListItem from the backend (confirmed from model.go).
interface StudentResult {
  id: string;
  full_name: string;
  admission_number: string;
  class_name?: string | null;
}

interface Invoice {
  id: string;
  invoice_number: string;
  status: string;
  balance: number;
}

interface FeeAccount {
  student_id: string;
  student_name: string;
  balance: number;
  invoices?: Invoice[];
}

type Mode = "mpesa" | "manual";
type Stage = "search" | "form" | "stk_pending" | "success" | "error";

const POLL_INTERVAL_MS = 4000;
const POLL_TIMEOUT_MS = 120_000; // 2 minutes — Daraja prompts are short-lived anyway

export default function RecordPaymentPage() {
  const [mode, setMode] = useState<Mode>("mpesa");
  const [stage, setStage] = useState<Stage>("search");

  // Student search
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<StudentResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [selectedStudent, setSelectedStudent] = useState<StudentResult | null>(null);

  // Fee account context for the selected student
  const [account, setAccount] = useState<FeeAccount | null>(null);
  const [accountLoading, setAccountLoading] = useState(false);

  // Form fields
  const [amount, setAmount] = useState("");
  const [phone, setPhone] = useState("");
  const [paymentMethod, setPaymentMethod] = useState<"cash" | "bank_transfer" | "cheque">("cash");
  const [invoiceId, setInvoiceId] = useState<string>("");
  const [bankReference, setBankReference] = useState("");
  const [chequeNumber, setChequeNumber] = useState("");
  const [notes, setNotes] = useState("");

  const [submitting, setSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const [receiptNumber, setReceiptNumber] = useState<string | null>(null);

  const pollTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollDeadline = useRef<number>(0);

  // --- Student search (debounced) ---
  useEffect(() => {
    if (query.trim().length < 2) {
      setResults([]);
      return;
    }
    setSearching(true);
    const t = setTimeout(() => {
      studentsAPI
        .list({ search: query.trim(), page_size: 8 })
        .then((res) => setResults(res.data?.data ?? []))
        .catch(() => setResults([]))
        .finally(() => setSearching(false));
    }, 300);
    return () => clearTimeout(t);
  }, [query]);

  // --- Once a student is picked, load their fee account for context ---
  useEffect(() => {
    if (!selectedStudent) {
      setAccount(null);
      return;
    }
    setAccountLoading(true);
    financeAPI
      .getFeeAccount(selectedStudent.id)
      .then((res) => setAccount(res.data?.data ?? null))
      .catch(() => setAccount(null))
      .finally(() => setAccountLoading(false));
  }, [selectedStudent]);

  // --- Cleanup any running poll on unmount ---
  useEffect(() => {
    return () => {
      if (pollTimer.current) clearInterval(pollTimer.current);
    };
  }, []);

  function selectStudent(s: StudentResult) {
    setSelectedStudent(s);
    setStage("form");
    setResults([]);
    setQuery("");
  }

  function resetAll() {
    setStage("search");
    setSelectedStudent(null);
    setAccount(null);
    setAmount("");
    setPhone("");
    setInvoiceId("");
    setBankReference("");
    setChequeNumber("");
    setNotes("");
    setErrorMsg("");
    setReceiptNumber(null);
    if (pollTimer.current) clearInterval(pollTimer.current);
  }

  async function submitMpesa() {
    if (!selectedStudent) return;
    const amt = Number(amount);
    if (!amt || amt <= 0) {
      setErrorMsg("Enter an amount greater than 0.");
      return;
    }
    if (!phone.trim()) {
      setErrorMsg("Enter the phone number to send the STK prompt to.");
      return;
    }

    setErrorMsg("");
    setSubmitting(true);
    try {
      await financeAPI.initiateStkPush({
        student_id: selectedStudent.id,
        amount: amt,
        phone: phone.trim(),
      });
      setStage("stk_pending");
      startPollingForPayment(amt);
    } catch (err: any) {
      setErrorMsg(err?.response?.data?.error?.message ?? "Could not send the STK prompt. Try again.");
      setStage("error");
    } finally {
      setSubmitting(false);
    }
  }

  // The STK push response doesn't confirm the payment — Daraja calls back
  // asynchronously. We poll payment history for a new mpesa payment matching
  // the amount we just requested, since that's the only signal the frontend
  // has that the callback has actually landed.
  function startPollingForPayment(expectedAmount: number) {
    if (!selectedStudent) return;
    const startedAt = Date.now();
    pollDeadline.current = startedAt + POLL_TIMEOUT_MS;

    pollTimer.current = setInterval(async () => {
      if (Date.now() > pollDeadline.current) {
        if (pollTimer.current) clearInterval(pollTimer.current);
        setErrorMsg(
          "No confirmation yet. The payment may still be processing — check Payment History shortly, or ask the payer to confirm the M-Pesa prompt on their phone."
        );
        setStage("error");
        return;
      }
      try {
        const res = await financeAPI.getPaymentHistory(selectedStudent.id);
        const payments: Array<{ amount: number; payment_method: string; received_at: string; receipt_number: string }> =
          res.data?.data ?? [];
        const match = payments.find(
          (p) =>
            p.payment_method === "mpesa" &&
            Number(p.amount) === expectedAmount &&
            new Date(p.received_at).getTime() >= startedAt - 5000
        );
        if (match) {
          if (pollTimer.current) clearInterval(pollTimer.current);
          setReceiptNumber(match.receipt_number);
          setStage("success");
        }
      } catch {
        // transient — keep polling until timeout
      }
    }, POLL_INTERVAL_MS);
  }

  async function submitManual() {
    if (!selectedStudent) return;
    const amt = Number(amount);
    if (!amt || amt <= 0) {
      setErrorMsg("Enter an amount greater than 0.");
      return;
    }
    if (paymentMethod === "bank_transfer" && !bankReference.trim()) {
      setErrorMsg("Enter the bank reference for this transfer.");
      return;
    }
    if (paymentMethod === "cheque" && !chequeNumber.trim()) {
      setErrorMsg("Enter the cheque number.");
      return;
    }

    setErrorMsg("");
    setSubmitting(true);
    try {
      const res = await financeAPI.recordManualPayment({
        student_id: selectedStudent.id,
        invoice_id: invoiceId || undefined,
        amount: amt,
        payment_method: paymentMethod,
        bank_reference: bankReference || undefined,
        cheque_number: chequeNumber || undefined,
        notes: notes || undefined,
      });
      setReceiptNumber(res.data?.data?.receipt_number ?? null);
      setStage("success");
    } catch (err: any) {
      setErrorMsg(err?.response?.data?.error?.message ?? "Could not record the payment. Try again.");
      setStage("error");
    } finally {
      setSubmitting(false);
    }
  }

  const outstandingInvoices = (account?.invoices ?? []).filter(
    (inv) => inv.status === "unpaid" || inv.status === "partial"
  );

  return (
    <DashboardLayout title="Record Payment">
      <div className="max-w-2xl mx-auto space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Record Payment</h1>
            <p className="page-subtitle">Send an M-Pesa prompt or log a manual payment</p>
          </div>
          {stage !== "search" && (
            <Button variant="outline" size="sm" onClick={resetAll}>
              <ArrowLeft className="h-3.5 w-3.5 mr-1.5" />
              Start over
            </Button>
          )}
        </div>

        {/* --- Stage: search --- */}
        {stage === "search" && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Find a student</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
                <Input
                  autoFocus
                  placeholder="Search by name or admission number..."
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  className="pl-9"
                />
              </div>

              {searching && (
                <p className="text-xs text-neutral-500 flex items-center gap-1.5">
                  <Loader2 className="h-3 w-3 animate-spin" /> Searching…
                </p>
              )}

              {!searching && query.trim().length >= 2 && results.length === 0 && (
                <p className="text-sm text-neutral-500 py-2">No students match &quot;{query}&quot;.</p>
              )}

              {results.length > 0 && (
                <div className="divide-y divide-neutral-100 border border-neutral-100 rounded-lg overflow-hidden">
                  {results.map((s) => (
                    <button
                      key={s.id}
                      onClick={() => selectStudent(s)}
                      className="w-full text-left px-4 py-3 hover:bg-neutral-50 flex items-center justify-between"
                    >
                      <div>
                        <p className="text-sm font-medium text-neutral-800">
                          {s.full_name}
                        </p>
                        <p className="text-xs text-neutral-400">
                          {s.admission_number}
                          {s.class_name ? ` · ${s.class_name}` : ""}
                        </p>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {/* --- Stage: form (student selected) --- */}
        {stage === "form" && selectedStudent && (
          <>
            <Card>
              <CardContent className="py-4 flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-neutral-800">
                    {selectedStudent.full_name}
                  </p>
                  <p className="text-xs text-neutral-400">{selectedStudent.admission_number}</p>
                </div>
                <div className="text-right">
                  {accountLoading ? (
                    <span className="text-xs text-neutral-400">Loading balance…</span>
                  ) : account ? (
                    <>
                      <p className="text-xs text-neutral-500">Current balance</p>
                      <p className="text-sm font-semibold text-error">{formatKES(account.balance)}</p>
                    </>
                  ) : (
                    <span className="text-xs text-neutral-400">Balance unavailable</span>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Mode tabs */}
            <div className="flex gap-2">
              <button
                onClick={() => setMode("mpesa")}
                className={`flex-1 flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium border ${
                  mode === "mpesa"
                    ? "border-primary-500 bg-primary-50 text-primary-700"
                    : "border-neutral-200 text-neutral-500 hover:bg-neutral-50"
                }`}
              >
                <Smartphone className="h-4 w-4" /> M-Pesa (STK Push)
              </button>
              <button
                onClick={() => setMode("manual")}
                className={`flex-1 flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium border ${
                  mode === "manual"
                    ? "border-primary-500 bg-primary-50 text-primary-700"
                    : "border-neutral-200 text-neutral-500 hover:bg-neutral-50"
                }`}
              >
                <Banknote className="h-4 w-4" /> Cash / Bank / Cheque
              </button>
            </div>

            <Card>
              <CardContent className="pt-5 space-y-4">
                {errorMsg && (
                  <div className="flex items-start gap-2 text-sm text-error bg-error-light/40 border border-error/20 rounded-md px-3 py-2">
                    <AlertTriangle className="h-4 w-4 mt-0.5 shrink-0" />
                    <span>{errorMsg}</span>
                  </div>
                )}

                <div>
                  <label className="text-xs font-medium text-neutral-600 block mb-1">Amount (KES)</label>
                  <Input
                    type="number"
                    min="1"
                    placeholder="0.00"
                    value={amount}
                    onChange={(e) => setAmount(e.target.value)}
                  />
                </div>

                {mode === "mpesa" ? (
                  <div>
                    <label className="text-xs font-medium text-neutral-600 block mb-1">Phone number</label>
                    <Input
                      placeholder="07XXXXXXXX"
                      value={phone}
                      onChange={(e) => setPhone(e.target.value)}
                    />
                    <p className="text-xs text-neutral-400 mt-1">
                      The payer will get an STK prompt on this number.
                    </p>
                  </div>
                ) : (
                  <>
                    <div>
                      <label className="text-xs font-medium text-neutral-600 block mb-1">Method</label>
                      <select
                        value={paymentMethod}
                        onChange={(e) => setPaymentMethod(e.target.value as typeof paymentMethod)}
                        className="w-full h-9 px-3 text-sm border border-neutral-200 rounded-md bg-white"
                      >
                        <option value="cash">Cash</option>
                        <option value="bank_transfer">Bank Transfer</option>
                        <option value="cheque">Cheque</option>
                      </select>
                    </div>

                    {paymentMethod === "bank_transfer" && (
                      <div>
                        <label className="text-xs font-medium text-neutral-600 block mb-1">Bank reference</label>
                        <Input value={bankReference} onChange={(e) => setBankReference(e.target.value)} />
                      </div>
                    )}

                    {paymentMethod === "cheque" && (
                      <div>
                        <label className="text-xs font-medium text-neutral-600 block mb-1">Cheque number</label>
                        <Input value={chequeNumber} onChange={(e) => setChequeNumber(e.target.value)} />
                      </div>
                    )}

                    {outstandingInvoices.length > 0 && (
                      <div>
                        <label className="text-xs font-medium text-neutral-600 block mb-1">
                          Apply to invoice (optional)
                        </label>
                        <select
                          value={invoiceId}
                          onChange={(e) => setInvoiceId(e.target.value)}
                          className="w-full h-9 px-3 text-sm border border-neutral-200 rounded-md bg-white"
                        >
                          <option value="">Don&apos;t allocate — record only</option>
                          {outstandingInvoices.map((inv) => (
                            <option key={inv.id} value={inv.id}>
                              {inv.invoice_number} — balance {formatKES(inv.balance)}
                            </option>
                          ))}
                        </select>
                      </div>
                    )}

                    <div>
                      <label className="text-xs font-medium text-neutral-600 block mb-1">Notes (optional)</label>
                      <Input value={notes} onChange={(e) => setNotes(e.target.value)} />
                    </div>
                  </>
                )}

                <Button
                  className="w-full"
                  disabled={submitting}
                  onClick={mode === "mpesa" ? submitMpesa : submitManual}
                >
                  {submitting ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" /> Processing…
                    </>
                  ) : mode === "mpesa" ? (
                    "Send STK Push"
                  ) : (
                    "Record Payment"
                  )}
                </Button>
              </CardContent>
            </Card>
          </>
        )}

        {/* --- Stage: waiting for Daraja callback --- */}
        {stage === "stk_pending" && (
          <Card>
            <CardContent className="py-10 flex flex-col items-center text-center gap-3">
              <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
              <p className="text-sm font-medium text-neutral-800">
                Prompt sent to {phone}
              </p>
              <p className="text-xs text-neutral-500 max-w-xs">
                Waiting for the payer to enter their M-Pesa PIN and complete the payment.
                This page will update automatically.
              </p>
            </CardContent>
          </Card>
        )}

        {/* --- Stage: success --- */}
        {stage === "success" && (
          <Card>
            <CardContent className="py-10 flex flex-col items-center text-center gap-3">
              <CheckCircle2 className="h-10 w-10 text-success" />
              <p className="text-sm font-medium text-neutral-800">Payment recorded</p>
              {receiptNumber && (
                <p className="text-xs text-neutral-500">Receipt {receiptNumber}</p>
              )}
              <div className="flex gap-2 mt-2">
                <Button variant="outline" size="sm" onClick={resetAll}>
                  Record another payment
                </Button>
                <Button
                  size="sm"
                  onClick={() => { window.location.href = "/finance"; }}
                >
                  Back to Finance
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* --- Stage: error --- */}
        {stage === "error" && (
          <Card>
            <CardContent className="py-10 flex flex-col items-center text-center gap-3">
              <AlertTriangle className="h-8 w-8 text-error" />
              <p className="text-sm font-medium text-neutral-800">{errorMsg || "Something went wrong."}</p>
              <Button variant="outline" size="sm" onClick={() => setStage("form")}>
                Try again
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </DashboardLayout>
  );
}