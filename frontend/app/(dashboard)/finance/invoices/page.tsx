"use client";
import React, { useEffect, useState } from "react";
import { Receipt, AlertTriangle, CheckCircle2, ExternalLink } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { financeAPI, academicsAPI } from "@/lib/api";

interface FeeStructure {
  id: string;
  name: string;
  is_active: boolean;
}

interface SchoolClass {
  id: string;
  name: string;
  stream?: string;
}

interface Term {
  id: string;
  name?: string;
  term_number?: number;
}

interface InvoiceSkip {
  student_id: string;
  reason: string;
}

interface GenerateResult {
  class_id: string;
  term_id: string;
  total_students: number;
  invoices_created: number;
  skipped: InvoiceSkip[];
}

const selectClass =
  "w-full rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-800 " +
  "focus:outline-none focus:ring-2 focus:ring-primary-500/30 focus:border-primary-500 disabled:opacity-50";

function classLabel(c: SchoolClass): string {
  return c.stream ? `${c.name} ${c.stream}` : c.name;
}

function termLabel(t: Term): string {
  return t.name ?? (t.term_number ? `Term ${t.term_number}` : t.id);
}

export default function InvoicesPage() {
  const [structures, setStructures] = useState<FeeStructure[]>([]);
  const [classes, setClasses] = useState<SchoolClass[]>([]);
  const [terms, setTerms] = useState<Term[]>([]);
  const [loadError, setLoadError] = useState(false);

  const [feeStructureId, setFeeStructureId] = useState("");
  const [classId, setClassId] = useState("");
  const [termId, setTermId] = useState("");
  const [termNumber, setTermNumber] = useState("1");

  const [generating, setGenerating] = useState(false);
  const [genError, setGenError] = useState<string | null>(null);
  const [result, setResult] = useState<GenerateResult | null>(null);

  useEffect(() => {
    Promise.all([
      financeAPI.listFeeStructures(),
      academicsAPI.listClasses(),
      academicsAPI.listTerms(),
    ])
      .then(([fsRes, classRes, termRes]) => {
        setStructures(fsRes.data?.data ?? []);
        setClasses(classRes.data?.data ?? []);
        setTerms(termRes.data?.data ?? []);
      })
      .catch(() => setLoadError(true));
  }, []);

  const handleGenerate = async () => {
    setGenError(null);
    setResult(null);
    if (!feeStructureId || !classId || !termId || !termNumber) {
      setGenError("Fee structure, class, term and term number are all required.");
      return;
    }
    setGenerating(true);
    try {
      const res = await financeAPI.generateInvoicesForClass({
        fee_structure_id: feeStructureId,
        class_id: classId,
        term_id: termId,
        term_number: parseInt(termNumber, 10),
      });
      setResult(res.data?.data);
    } catch (err: any) {
      setGenError(err?.response?.data?.error?.message ?? "Could not generate invoices.");
    } finally {
      setGenerating(false);
    }
  };

  return (
    <DashboardLayout title="Generate Invoices">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Generate Invoices</h1>
            <p className="page-subtitle">Bill an entire class in one go, from a fee structure</p>
          </div>
        </div>

        {loadError && (
          <Card className="border-error/30 bg-error-light/30">
            <CardContent className="py-4 flex items-center gap-2 text-error text-sm">
              <AlertTriangle className="h-4 w-4" />
              Could not load fee structures, classes, or terms. Try refreshing.
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader className="pb-3">
            <CardTitle>Generate Invoices for a Class</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {genError && (
              <div className="text-sm text-error flex items-center gap-2">
                <AlertTriangle className="h-4 w-4" /> {genError}
              </div>
            )}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-medium text-neutral-500 mb-1 block">Fee Structure</label>
                <select className={selectClass} value={feeStructureId} onChange={(e) => setFeeStructureId(e.target.value)}>
                  <option value="">Select fee structure…</option>
                  {structures.map((fs) => (
                    <option key={fs.id} value={fs.id}>{fs.name}{!fs.is_active ? " (inactive)" : ""}</option>
                  ))}
                </select>
                {structures.length === 0 && !loadError && (
                  <p className="text-xs text-neutral-400 mt-1">
                    No fee structures yet — create one under Fee Structures first.
                  </p>
                )}
              </div>
              <div>
                <label className="text-xs font-medium text-neutral-500 mb-1 block">Class</label>
                <select className={selectClass} value={classId} onChange={(e) => setClassId(e.target.value)}>
                  <option value="">Select class…</option>
                  {classes.map((c) => (
                    <option key={c.id} value={c.id}>{classLabel(c)}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-neutral-500 mb-1 block">Term</label>
                <select className={selectClass} value={termId} onChange={(e) => setTermId(e.target.value)}>
                  <option value="">Select term…</option>
                  {terms.map((t) => (
                    <option key={t.id} value={t.id}>{termLabel(t)}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-neutral-500 mb-1 block">
                  Term Number
                  <span className="font-normal text-neutral-400"> (filters which components apply)</span>
                </label>
                <select className={selectClass} value={termNumber} onChange={(e) => setTermNumber(e.target.value)}>
                  <option value="1">Term 1</option>
                  <option value="2">Term 2</option>
                  <option value="3">Term 3</option>
                </select>
              </div>
            </div>
            <Button onClick={handleGenerate} disabled={generating}>
              <Receipt className="h-4 w-4 mr-1.5" />
              {generating ? "Generating…" : "Generate Invoices"}
            </Button>
            <p className="text-xs text-neutral-400">
              Safe to re-run — students who already have an invoice for this term are skipped, not double-billed.
            </p>
          </CardContent>
        </Card>

        {result && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Result</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div className="rounded-lg p-4 bg-neutral-50">
                  <p className="text-2xl font-semibold text-neutral-800">{result.total_students}</p>
                  <p className="text-sm text-neutral-500 mt-0.5">Students in class</p>
                </div>
                <div className="rounded-lg p-4 bg-success-light">
                  <p className="text-2xl font-semibold text-success-dark">{result.invoices_created}</p>
                  <p className="text-sm text-success-dark mt-0.5">Invoices created</p>
                </div>
                <div className="rounded-lg p-4 bg-warning-light">
                  <p className="text-2xl font-semibold text-warning-dark">{result.skipped.length}</p>
                  <p className="text-sm text-warning-dark mt-0.5">Skipped</p>
                </div>
              </div>

              {result.skipped.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-neutral-500 mb-2">Skipped students</p>
                  <ul className="text-sm divide-y divide-neutral-50 border border-neutral-100 rounded-md">
                    {result.skipped.map((s) => (
                      <li key={s.student_id} className="px-3 py-2 flex justify-between">
                        <span className="text-neutral-600">{s.student_id}</span>
                        <span className="text-neutral-400">{s.reason}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {result.invoices_created > 0 && (
                <div className="flex items-center gap-2 text-sm text-success">
                  <CheckCircle2 className="h-4 w-4" />
                  Check individual balances on the
                  <a href="/finance" className="font-medium underline inline-flex items-center gap-1">
                    Finance overview <ExternalLink className="h-3 w-3" />
                  </a>
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </DashboardLayout>
  );
}