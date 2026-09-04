"use client";
import React, { useEffect, useState } from "react";
import { Plus, AlertTriangle, ChevronDown, ChevronUp, CheckCircle2 } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { financeAPI, academicsAPI } from "@/lib/api";
import { formatKES } from "@/lib/utils";

interface FeeComponent {
  id: string;
  name: string;
  amount: number;
  term_number: number | null;
  is_mandatory: boolean;
  due_date: string | null;
}

interface FeeStructure {
  id: string;
  academic_year_id: string;
  class_id: string | null;
  name: string;
  is_active: boolean;
  components?: FeeComponent[];
}

interface AcademicYear {
  id: string;
  name?: string;
  label?: string;
  year?: string;
}

interface SchoolClass {
  id: string;
  name: string;
  stream?: string;
  level?: number;
}

const selectClass =
  "w-full rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-800 " +
  "focus:outline-none focus:ring-2 focus:ring-primary-500/30 focus:border-primary-500 disabled:opacity-50";

function ayLabel(ay: AcademicYear): string {
  return ay.name ?? ay.label ?? ay.year ?? ay.id;
}

function classLabel(c: SchoolClass): string {
  return c.stream ? `${c.name} ${c.stream}` : c.name;
}

export default function FeeStructuresPage() {
  const [structures, setStructures] = useState<FeeStructure[] | null>(null);
  const [structuresError, setStructuresError] = useState(false);
  const [structuresLoading, setStructuresLoading] = useState(true);

  const [academicYears, setAcademicYears] = useState<AcademicYear[]>([]);
  const [classes, setClasses] = useState<SchoolClass[]>([]);

  const [filterYear, setFilterYear] = useState<string>("");

  // Create-structure form
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newYearId, setNewYearId] = useState("");
  const [newClassId, setNewClassId] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  // Expanded structure + its add-component form
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [compName, setCompName] = useState("");
  const [compAmount, setCompAmount] = useState("");
  const [compTerm, setCompTerm] = useState("");
  const [compMandatory, setCompMandatory] = useState(true);
  const [addingComponent, setAddingComponent] = useState(false);
  const [componentError, setComponentError] = useState<string | null>(null);

  const loadStructures = (academicYearId?: string) => {
    setStructuresLoading(true);
    setStructuresError(false);
    financeAPI.listFeeStructures(academicYearId || undefined)
      .then((res) => setStructures(res.data?.data ?? []))
      .catch(() => setStructuresError(true))
      .finally(() => setStructuresLoading(false));
  };

  useEffect(() => {
    loadStructures();
    academicsAPI.listAcademicYears()
      .then((res) => setAcademicYears(res.data?.data ?? []))
      .catch(() => setAcademicYears([]));
    academicsAPI.listClasses()
      .then((res) => setClasses(res.data?.data ?? []))
      .catch(() => setClasses([]));
  }, []);

  useEffect(() => {
    loadStructures(filterYear);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterYear]);

  const handleCreate = async () => {
    setCreateError(null);
    if (!newName.trim() || !newYearId) {
      setCreateError("Name and academic year are required.");
      return;
    }
    setCreating(true);
    try {
      await financeAPI.createFeeStructure({
        name: newName.trim(),
        academic_year_id: newYearId,
        class_id: newClassId || undefined,
      });
      setNewName("");
      setNewClassId("");
      setShowCreate(false);
      loadStructures(filterYear);
    } catch (err: any) {
      setCreateError(err?.response?.data?.error?.message ?? "Could not create fee structure.");
    } finally {
      setCreating(false);
    }
  };

  const handleAddComponent = async (structureId: string) => {
    setComponentError(null);
    const amount = parseFloat(compAmount);
    if (!compName.trim() || !amount || amount <= 0) {
      setComponentError("Name and a positive amount are required.");
      return;
    }
    setAddingComponent(true);
    try {
      await financeAPI.addFeeComponent(structureId, {
        name: compName.trim(),
        amount,
        term_number: compTerm ? parseInt(compTerm, 10) : undefined,
        is_mandatory: compMandatory,
      });
      setCompName("");
      setCompAmount("");
      setCompTerm("");
      setCompMandatory(true);
      loadStructures(filterYear);
    } catch (err: any) {
      setComponentError(err?.response?.data?.error?.message ?? "Could not add component.");
    } finally {
      setAddingComponent(false);
    }
  };

  return (
    <DashboardLayout title="Fee Structures">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Fee Structures</h1>
            <p className="page-subtitle">Set up tuition, boarding, transport and other fee components per class</p>
          </div>
          <div className="flex gap-2">
            <select
              className={selectClass + " w-48"}
              value={filterYear}
              onChange={(e) => setFilterYear(e.target.value)}
            >
              <option value="">All academic years</option>
              {academicYears.map((ay) => (
                <option key={ay.id} value={ay.id}>{ayLabel(ay)}</option>
              ))}
            </select>
            <Button size="sm" onClick={() => setShowCreate((v) => !v)}>
              <Plus className="h-4 w-4 mr-1" />
              New Fee Structure
            </Button>
          </div>
        </div>

        {/* Create form */}
        {showCreate && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>New Fee Structure</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {createError && (
                <div className="text-sm text-error flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4" /> {createError}
                </div>
              )}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div>
                  <label className="text-xs font-medium text-neutral-500 mb-1 block">Name</label>
                  <Input
                    placeholder="e.g. Form 1 East — 2026 Fees"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-xs font-medium text-neutral-500 mb-1 block">Academic Year</label>
                  <select className={selectClass} value={newYearId} onChange={(e) => setNewYearId(e.target.value)}>
                    <option value="">Select year…</option>
                    {academicYears.map((ay) => (
                      <option key={ay.id} value={ay.id}>{ayLabel(ay)}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="text-xs font-medium text-neutral-500 mb-1 block">Class (optional)</label>
                  <select className={selectClass} value={newClassId} onChange={(e) => setNewClassId(e.target.value)}>
                    <option value="">All classes</option>
                    {classes.map((c) => (
                      <option key={c.id} value={c.id}>{classLabel(c)}</option>
                    ))}
                  </select>
                </div>
              </div>
              <div className="flex gap-2 pt-1">
                <Button size="sm" onClick={handleCreate} disabled={creating}>
                  {creating ? "Creating…" : "Create Fee Structure"}
                </Button>
                <Button size="sm" variant="outline" onClick={() => setShowCreate(false)}>
                  Cancel
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Structures list */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle>All Fee Structures</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {structuresLoading ? (
              <div className="px-5 py-8 text-center text-sm text-neutral-500">Loading…</div>
            ) : structuresError ? (
              <div className="px-5 py-8 text-center text-sm text-error flex items-center justify-center gap-2">
                <AlertTriangle className="h-4 w-4" />
                Could not load fee structures. Try refreshing.
              </div>
            ) : (structures ?? []).length === 0 ? (
              <div className="px-5 py-8 text-center text-sm text-neutral-500">
                No fee structures yet. Create one above to start billing a class.
              </div>
            ) : (
              <div className="divide-y divide-neutral-50">
                {(structures ?? []).map((fs) => {
                  const isExpanded = expandedId === fs.id;
                  const total = (fs.components ?? []).reduce((s, c) => s + c.amount, 0);
                  return (
                    <div key={fs.id}>
                      <button
                        className="w-full flex items-center justify-between px-5 py-4 text-left hover:bg-neutral-50"
                        onClick={() => {
                          setExpandedId(isExpanded ? null : fs.id);
                          setComponentError(null);
                        }}
                      >
                        <div>
                          <p className="text-sm font-medium text-neutral-800">{fs.name}</p>
                          <p className="text-xs text-neutral-400 mt-0.5">
                            {(fs.components ?? []).length} component{(fs.components ?? []).length === 1 ? "" : "s"}
                            {total > 0 && <> · {formatKES(total)} total</>}
                            {!fs.is_active && <span className="text-error"> · inactive</span>}
                          </p>
                        </div>
                        {isExpanded ? (
                          <ChevronUp className="h-4 w-4 text-neutral-400" />
                        ) : (
                          <ChevronDown className="h-4 w-4 text-neutral-400" />
                        )}
                      </button>

                      {isExpanded && (
                        <div className="px-5 pb-5 space-y-4 bg-neutral-25">
                          {/* Existing components */}
                          {(fs.components ?? []).length > 0 && (
                            <table className="w-full text-sm">
                              <thead>
                                <tr className="text-xs text-neutral-500 uppercase tracking-wide">
                                  <th className="text-left py-1.5">Component</th>
                                  <th className="text-left py-1.5">Term</th>
                                  <th className="text-left py-1.5">Mandatory</th>
                                  <th className="text-right py-1.5">Amount</th>
                                </tr>
                              </thead>
                              <tbody>
                                {fs.components!.map((c) => (
                                  <tr key={c.id} className="border-t border-neutral-100">
                                    <td className="py-1.5">{c.name}</td>
                                    <td className="py-1.5 text-neutral-500">{c.term_number ?? "Any"}</td>
                                    <td className="py-1.5 text-neutral-500">
                                      {c.is_mandatory ? (
                                        <CheckCircle2 className="h-3.5 w-3.5 text-success" />
                                      ) : (
                                        "—"
                                      )}
                                    </td>
                                    <td className="py-1.5 text-right font-medium">{formatKES(c.amount)}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          )}

                          {/* Add component form */}
                          <div className="rounded-md border border-neutral-200 bg-white p-3 space-y-2">
                            <p className="text-xs font-medium text-neutral-500">Add a component</p>
                            {componentError && (
                              <div className="text-xs text-error flex items-center gap-1.5">
                                <AlertTriangle className="h-3.5 w-3.5" /> {componentError}
                              </div>
                            )}
                            <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                              <Input
                                placeholder="e.g. Tuition"
                                value={compName}
                                onChange={(e) => setCompName(e.target.value)}
                                className="text-sm"
                              />
                              <Input
                                placeholder="Amount (KES)"
                                type="number"
                                value={compAmount}
                                onChange={(e) => setCompAmount(e.target.value)}
                                className="text-sm"
                              />
                              <Input
                                placeholder="Term # (optional)"
                                type="number"
                                value={compTerm}
                                onChange={(e) => setCompTerm(e.target.value)}
                                className="text-sm"
                              />
                              <label className="flex items-center gap-1.5 text-xs text-neutral-600 px-1">
                                <input
                                  type="checkbox"
                                  checked={compMandatory}
                                  onChange={(e) => setCompMandatory(e.target.checked)}
                                />
                                Mandatory
                              </label>
                            </div>
                            <Button
                              size="sm"
                              onClick={() => handleAddComponent(fs.id)}
                              disabled={addingComponent}
                            >
                              {addingComponent ? "Adding…" : "Add Component"}
                            </Button>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}