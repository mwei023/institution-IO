"use client";
import React, { useEffect, useState, useCallback } from "react";
import {
  Loader2, AlertTriangle, Plus, CheckCircle2, ChevronRight,
} from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { academicsAPI, cbcAPI } from "@/lib/api";

interface ClassOption { id: string; name: string; stream: string | null }
interface LearningAreaOption { id: string; name: string; code: string | null; is_compulsory: boolean }
interface CbcComponent {
  id: string; name: string; weight: number; max_marks: number;
}
interface CbcEntry {
  id: string; component_id: string; student_id: string;
  student_name: string; marks: number | null; is_absent: boolean;
}

type Step = "select" | "components" | "gradebook";

export default function TeacherCbcGradesPage() {
  const [step, setStep] = useState<Step>("select");

  // Step 1: selectors
  const [classes, setClasses] = useState<ClassOption[]>([]);
  const [learningAreas, setLearningAreas] = useState<LearningAreaOption[]>([]);
  const [terms, setTerms] = useState<{ id: string; name: string }[]>([]);
  const [selectedClass, setSelectedClass] = useState("");
  const [selectedLearningArea, setSelectedLearningArea] = useState("");
  const [selectedTerm, setSelectedTerm] = useState("");
  const [selectLoading, setSelectLoading] = useState(true);

  // Inline "new learning area" form — mirrors the 844 gradebook's
  // inline "new component" pattern. No separate admin page for
  // learning areas, same as 844 has none for subjects.
  const [showNewLA, setShowNewLA] = useState(false);
  const [newLAName, setNewLAName] = useState("");
  const [newLACode, setNewLACode] = useState("");
  const [newLACompulsory, setNewLACompulsory] = useState(true);
  const [creatingLA, setCreatingLA] = useState(false);
  const [createLAError, setCreateLAError] = useState<string | null>(null);

  // Step 2: components
  const [components, setComponents] = useState<CbcComponent[]>([]);
  const [compLoading, setCompLoading] = useState(false);
  const [selectedComponent, setSelectedComponent] = useState<CbcComponent | null>(null);

  const [showNewComp, setShowNewComp] = useState(false);
  const [newCompName, setNewCompName] = useState("");
  const [newCompWeight, setNewCompWeight] = useState("");
  const [newCompMax, setNewCompMax] = useState("100");
  const [creatingComp, setCreatingComp] = useState(false);
  const [createCompError, setCreateCompError] = useState<string | null>(null);

  // Step 3: gradebook
  const [entries, setEntries] = useState<CbcEntry[]>([]);
  const [marksMap, setMarksMap] = useState<Record<string, string>>({});
  const [absentMap, setAbsentMap] = useState<Record<string, boolean>>({});
  const [gradebookLoading, setGradebookLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);

  const [error, setError] = useState<string | null>(null);

 useEffect(() => {
  Promise.all([
    academicsAPI.listClasses(),
    cbcAPI.listLearningAreas(),
    academicsAPI.listTerms(),
  ])
    .then(([clRes, laRes, termRes]) => {
      setClasses(clRes.data?.data ?? []);
      setLearningAreas(laRes.data?.data ?? []);
      const termList = termRes.data?.data ?? [];
      setTerms(termList);
      const current = termList.find((t: any) => t.is_current);
      if (current) setSelectedTerm(current.id);
    })
    .catch(() => setError("Could not load classes, learning areas, and terms."))
    .finally(() => setSelectLoading(false));
}, []);


  const handleCreateLearningArea = async () => {
    if (!newLAName) {
      setCreateLAError("Name is required.");
      return;
    }
    setCreatingLA(true);
    setCreateLAError(null);
    try {
      const res = await cbcAPI.createLearningArea({
        name: newLAName,
        code: newLACode || undefined,
        is_compulsory: newLACompulsory,
      });
      const created = res.data?.data;
      setShowNewLA(false);
      setNewLAName(""); setNewLACode(""); setNewLACompulsory(true);
      const listRes = await cbcAPI.listLearningAreas();
      setLearningAreas(listRes.data?.data ?? []);
      if (created?.id) setSelectedLearningArea(created.id);
    } catch (err: any) {
      setCreateLAError(err?.response?.data?.error?.message ?? "Could not create learning area.");
    } finally {
      setCreatingLA(false);
    }
  };

  const loadComponents = useCallback(() => {
    if (!selectedClass || !selectedLearningArea || !selectedTerm) return;
    setCompLoading(true);
    setComponents([]);
    setSelectedComponent(null);
    setStep("components");
    cbcAPI.getComponents(selectedClass, selectedLearningArea, selectedTerm)
      .then((res) => setComponents(res.data?.data ?? []))
      .catch(() => setError("Could not load components."))
      .finally(() => setCompLoading(false));
  }, [selectedClass, selectedLearningArea, selectedTerm]);

  const loadGradebook = useCallback((component: CbcComponent) => {
    setSelectedComponent(component);
    setGradebookLoading(true);
    setSaveSuccess(false);
    setSaveError(null);
    setStep("gradebook");
    cbcAPI.getEntries(component.id)
      .then((res) => {
        const data: CbcEntry[] = res.data?.data ?? [];
        setEntries(data);
        const mMap: Record<string, string> = {};
        const aMap: Record<string, boolean> = {};
        data.forEach((e) => {
          mMap[e.student_id] = e.marks != null ? String(e.marks) : "";
          aMap[e.student_id] = e.is_absent;
        });
        setMarksMap(mMap);
        setAbsentMap(aMap);
      })
      .catch(() => setError("Could not load gradebook."))
      .finally(() => setGradebookLoading(false));
  }, []);

  const handleCreateComponent = async () => {
    if (!newCompName || !newCompWeight || !newCompMax) {
      setCreateCompError("All fields are required.");
      return;
    }
    setCreatingComp(true);
    setCreateCompError(null);
    try {
      await cbcAPI.createComponent({
        class_id: selectedClass,
        learning_area_id: selectedLearningArea,
        term_id: selectedTerm,
        name: newCompName,
        weight: parseFloat(newCompWeight),
        max_marks: parseFloat(newCompMax),
      });
      setShowNewComp(false);
      setNewCompName(""); setNewCompWeight(""); setNewCompMax("100");
      const res = await cbcAPI.getComponents(selectedClass, selectedLearningArea, selectedTerm);
      setComponents(res.data?.data ?? []);
    } catch (err: any) {
      setCreateCompError(err?.response?.data?.error?.message ?? "Could not create component.");
    } finally {
      setCreatingComp(false);
    }
  };

  const handleSaveGrades = async () => {
    if (!selectedComponent) return;
    setSaving(true);
    setSaveError(null);
    setSaveSuccess(false);
    try {
      await cbcAPI.saveEntries({
        component_id: selectedComponent.id,
        entries: entries.map((e) => ({
          student_id: e.student_id,
          marks: absentMap[e.student_id] ? null : (
            marksMap[e.student_id] !== "" ? parseFloat(marksMap[e.student_id]) : null
          ),
          is_absent: absentMap[e.student_id] ?? false,
        })),
      });
      setSaveSuccess(true);
    } catch (err: any) {
      setSaveError(err?.response?.data?.error?.message ?? "Could not save grades.");
    } finally {
      setSaving(false);
    }
  };

  const learningAreaName = learningAreas.find((l) => l.id === selectedLearningArea)?.name ?? "";
  const className = classes.find((c) => c.id === selectedClass)?.name ?? "";

  return (
    <DashboardLayout title="Grades">
      <div className="space-y-6">
        <div className="page-header">
          <div>
            <h1 className="page-title">Grades</h1>
            <p className="page-subtitle">Enter and manage CBC learning area marks</p>
          </div>
        </div>

        {error && (
          <Card className="border-error/30 bg-error-light/20">
            <CardContent className="py-3 flex items-center gap-2 text-error text-sm">
              <AlertTriangle className="h-4 w-4" /> {error}
            </CardContent>
          </Card>
        )}

        {/* Step 1 — Select class / learning area / term */}
        <Card>
          <CardHeader className="flex-row items-center justify-between pb-3">
            <CardTitle>Select Class & Learning Area</CardTitle>
            <Button size="sm" variant="outline" onClick={() => setShowNewLA(!showNewLA)}>
              <Plus className="h-3.5 w-3.5 mr-1" /> New Learning Area
            </Button>
          </CardHeader>

          {showNewLA && (
            <CardContent className="border-t border-neutral-100 pt-4">
              <div className="flex flex-wrap gap-3 items-end">
                <div className="flex-1 min-w-[160px]">
                  <label className="block text-xs font-medium text-neutral-500 mb-1">Name</label>
                  <Input
                    value={newLAName}
                    onChange={(e) => setNewLAName(e.target.value)}
                    placeholder="Mathematics"
                    className="h-8 text-sm"
                  />
                </div>
                <div className="w-32">
                  <label className="block text-xs font-medium text-neutral-500 mb-1">Code (optional)</label>
                  <Input
                    value={newLACode}
                    onChange={(e) => setNewLACode(e.target.value)}
                    placeholder="MATH"
                    className="h-8 text-sm"
                  />
                </div>
                <label className="flex items-center gap-2 text-sm text-neutral-600 pb-1.5">
                  <input
                    type="checkbox"
                    checked={newLACompulsory}
                    onChange={(e) => setNewLACompulsory(e.target.checked)}
                    className="h-4 w-4 rounded border-neutral-300 text-primary-600"
                  />
                  Compulsory
                </label>
                <Button size="sm" onClick={handleCreateLearningArea} disabled={creatingLA}>
                  {creatingLA ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Create"}
                </Button>
              </div>
              {createLAError && <p className="text-xs text-error mt-2">{createLAError}</p>}
            </CardContent>
          )}

          <CardContent className="flex flex-wrap gap-4 items-end">
            {selectLoading ? (
              <div className="flex items-center gap-2 text-sm text-neutral-500">
                <Loader2 className="h-4 w-4 animate-spin" /> Loading…
              </div>
            ) : (
              <>
                <div className="flex-1 min-w-[180px]">
                  <label className="block text-xs font-medium text-neutral-500 uppercase tracking-wide mb-1.5">Class</label>
                  <select
                    className="w-full h-9 rounded-md border border-neutral-200 bg-white px-3 text-sm text-neutral-800 focus:outline-none focus:ring-2 focus:ring-primary-200"
                    value={selectedClass}
                    onChange={(e) => { setSelectedClass(e.target.value); setStep("select"); }}
                  >
                    <option value="">Select class…</option>
                    {classes.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}{c.stream ? ` ${c.stream}` : ""}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="flex-1 min-w-[180px]">
                  <label className="block text-xs font-medium text-neutral-500 uppercase tracking-wide mb-1.5">Learning Area</label>
                  <select
                    className="w-full h-9 rounded-md border border-neutral-200 bg-white px-3 text-sm text-neutral-800 focus:outline-none focus:ring-2 focus:ring-primary-200"
                    value={selectedLearningArea}
                    onChange={(e) => { setSelectedLearningArea(e.target.value); setStep("select"); }}
                  >
                    <option value="">Select learning area…</option>
                    {learningAreas.map((l) => (
                      <option key={l.id} value={l.id}>{l.name}</option>
                    ))}
                  </select>
                  {learningAreas.length === 0 && (
                    <p className="text-xs text-neutral-400 mt-1">
                      None yet — use &ldquo;New Learning Area&rdquo; above.
                    </p>
                  )}
                </div>

                <Button
                  onClick={loadComponents}
                  disabled={!selectedClass || !selectedLearningArea}
                >
                  View Components
                  <ChevronRight className="h-4 w-4 ml-1" />
                </Button>
              </>
            )}
          </CardContent>
        </Card>

        {/* Step 2 — Components */}
        {(step === "components" || step === "gradebook") && (
          <Card>
            <CardHeader className="flex-row items-center justify-between pb-3">
              <CardTitle>{className} · {learningAreaName} — Assessments</CardTitle>
              <Button size="sm" variant="outline" onClick={() => setShowNewComp(!showNewComp)}>
                <Plus className="h-3.5 w-3.5 mr-1" /> New Component
              </Button>
            </CardHeader>

            {showNewComp && (
              <CardContent className="border-t border-neutral-100 pt-4">
                <div className="flex flex-wrap gap-3 items-end">
                  <div className="flex-1 min-w-[160px]">
                    <label className="block text-xs font-medium text-neutral-500 mb-1">Name (e.g. End Term)</label>
                    <Input
                      value={newCompName}
                      onChange={(e) => setNewCompName(e.target.value)}
                      placeholder="End Term"
                      className="h-8 text-sm"
                    />
                  </div>
                  <div className="w-24">
                    <label className="block text-xs font-medium text-neutral-500 mb-1">Weight %</label>
                    <Input
                      type="number" min="1" max="100"
                      value={newCompWeight}
                      onChange={(e) => setNewCompWeight(e.target.value)}
                      placeholder="100"
                      className="h-8 text-sm"
                    />
                  </div>
                  <div className="w-24">
                    <label className="block text-xs font-medium text-neutral-500 mb-1">Max Marks</label>
                    <Input
                      type="number" min="1"
                      value={newCompMax}
                      onChange={(e) => setNewCompMax(e.target.value)}
                      placeholder="100"
                      className="h-8 text-sm"
                    />
                  </div>
                  <Button size="sm" onClick={handleCreateComponent} disabled={creatingComp}>
                    {creatingComp ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Create"}
                  </Button>
                </div>
                {createCompError && <p className="text-xs text-error mt-2">{createCompError}</p>}
              </CardContent>
            )}

            <CardContent className="p-0">
              {compLoading ? (
                <div className="py-8 text-center text-sm text-neutral-500 flex items-center justify-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading assessments…
                </div>
              ) : components.length === 0 ? (
                <div className="py-8 text-center text-sm text-neutral-400">
                  No assessments yet. Create one above.
                </div>
              ) : (
                <div className="divide-y divide-neutral-50">
                  {components.map((comp) => (
                    <button
                      key={comp.id}
                      onClick={() => loadGradebook(comp)}
                      className={`w-full flex items-center justify-between px-5 py-3.5 text-left transition-colors hover:bg-neutral-50 ${
                        selectedComponent?.id === comp.id ? "bg-primary-50" : ""
                      }`}
                    >
                      <div>
                        <p className="text-sm font-medium text-neutral-800">{comp.name}</p>
                        <p className="text-xs text-neutral-500 mt-0.5">
                          Weight: {comp.weight}% · Out of {comp.max_marks}
                        </p>
                      </div>
                      <ChevronRight className="h-4 w-4 text-neutral-300" />
                    </button>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {/* Step 3 — Gradebook */}
        {step === "gradebook" && selectedComponent && (
          <Card>
            <CardHeader className="flex-row items-center justify-between pb-3">
              <div>
                <CardTitle>{selectedComponent.name}</CardTitle>
                <p className="text-xs text-neutral-500 mt-0.5">
                  Out of {selectedComponent.max_marks} marks
                </p>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {gradebookLoading ? (
                <div className="py-8 text-center text-sm text-neutral-500 flex items-center justify-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading gradebook…
                </div>
              ) : (
                <>
                  <div className="flex items-center gap-4 px-5 py-2 bg-neutral-50 border-b border-neutral-100">
                    <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide flex-1">Student</p>
                    <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide w-28 text-center">Marks</p>
                    <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide w-20 text-center">Absent</p>
                  </div>

                  <div className="divide-y divide-neutral-50">
                    {entries.map((entry) => {
                      const isAbsent = absentMap[entry.student_id] ?? false;
                      const marksVal = marksMap[entry.student_id] ?? "";
                      const pct = !isAbsent && marksVal !== ""
                        ? (parseFloat(marksVal) / selectedComponent.max_marks) * 100
                        : null;

                      return (
                        <div key={entry.student_id} className="flex items-center gap-4 px-5 py-3">
                          <div className="flex-1">
                            <p className="text-sm font-medium text-neutral-800">{entry.student_name}</p>
                            {pct !== null && (
                              <p className="text-xs text-neutral-400 mt-0.5">{pct.toFixed(1)}%</p>
                            )}
                          </div>
                          <div className="w-28">
                            <Input
                              type="number"
                              min="0"
                              max={selectedComponent.max_marks}
                              value={isAbsent ? "" : marksVal}
                              disabled={isAbsent}
                              onChange={(e) =>
                                setMarksMap((prev) => ({ ...prev, [entry.student_id]: e.target.value }))
                              }
                              placeholder={`0–${selectedComponent.max_marks}`}
                              className="h-8 text-sm text-center disabled:opacity-40"
                            />
                          </div>
                          <div className="w-20 flex justify-center">
                            <input
                              type="checkbox"
                              checked={isAbsent}
                              onChange={(e) =>
                                setAbsentMap((prev) => ({ ...prev, [entry.student_id]: e.target.checked }))
                              }
                              className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500 cursor-pointer"
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>

                  {saveError && (
                    <div className="px-5 py-3 border-t border-neutral-100 flex items-center gap-2 text-error text-sm">
                      <AlertTriangle className="h-4 w-4" /> {saveError}
                    </div>
                  )}
                  {saveSuccess && (
                    <div className="px-5 py-3 border-t border-neutral-100 flex items-center gap-2 text-success-dark text-sm">
                      <CheckCircle2 className="h-4 w-4" /> Grades saved successfully.
                    </div>
                  )}

                  <div className="px-5 py-4 border-t border-neutral-100 flex justify-end">
                    <Button onClick={handleSaveGrades} disabled={saving}>
                      {saving ? (
                        <span className="inline-flex items-center gap-2">
                          <Loader2 className="h-4 w-4 animate-spin" /> Saving…
                        </span>
                      ) : "Save Grades"}
                    </Button>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </DashboardLayout>
  );
}

function setSubjects(arg0: any) {
  throw new Error("Function not implemented.");
}
