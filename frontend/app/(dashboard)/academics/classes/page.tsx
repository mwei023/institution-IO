"use client";
import React, { useEffect, useState, useCallback } from "react";
import { Plus, UserCheck, AlertTriangle, Loader2 } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { academicsAPI } from "@/lib/api";
import { useInstitutionStore } from "@/lib/institutionStore";

interface ClassRow {
  id: string;
  name: string;
  stream: string | null;
  level: number;
  class_teacher_id: string | null;
  class_teacher: string | null;
  capacity: number;
  enrollment: number;
}

interface TeacherOption {
  id: string;
  full_name: string;
}

interface AcademicYearOption {
  id: string;
  name: string;
  is_current: boolean;
}

export default function ClassesAdminPage() {
  const { context } = useInstitutionStore();

  const [classes, setClasses] = useState<ClassRow[]>([]);
  const [teachers, setTeachers] = useState<TeacherOption[]>([]);
  const [years, setYears] = useState<AcademicYearOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create-class form state
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [stream, setStream] = useState("");
  const [level, setLevel] = useState("");
  const [capacity, setCapacity] = useState("");
  const [academicYearId, setAcademicYearId] = useState("");
  const [creating, setCreating] = useState(false);

  // Assign-teacher inline state
  const [assigningClassId, setAssigningClassId] = useState<string | null>(null);
  const [assignTeacherId, setAssignTeacherId] = useState("");
  const [assigning, setAssigning] = useState(false);

  const loadAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [classesRes, teachersRes, yearsRes] = await Promise.all([
        academicsAPI.listClasses(),
        academicsAPI.listTeachers(),
        academicsAPI.listAcademicYears(),
      ]);
      setClasses(classesRes.data?.data ?? []);
      setTeachers(teachersRes.data?.data ?? []);
      const yearList: AcademicYearOption[] = yearsRes.data?.data ?? [];
      setYears(yearList);
      const current = yearList.find((y) => y.is_current);
      if (current) setAcademicYearId(current.id);
    } catch {
      setError("Couldn't load classes. Try refreshing.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  const handleCreate = async () => {
    if (!name || !level || !capacity || !academicYearId) return;
    setCreating(true);
    setError(null);
    try {
      await academicsAPI.createClass({
        academic_year_id: academicYearId,
        name,
        stream: stream || undefined,
        level: parseInt(level, 10),
        capacity: parseInt(capacity, 10),
      });
      setName("");
      setStream("");
      setLevel("");
      setCapacity("");
      setShowCreate(false);
      await loadAll();
    } catch (err: any) {
      setError(err?.response?.data?.error?.message ?? "Couldn't create class.");
    } finally {
      setCreating(false);
    }
  };

  const handleAssign = async (classId: string) => {
    if (!assignTeacherId) return;
    setAssigning(true);
    try {
      await academicsAPI.assignClassTeacher(classId, assignTeacherId);
      setAssigningClassId(null);
      setAssignTeacherId("");
      await loadAll();
    } catch {
      setError("Couldn't assign class teacher.");
    } finally {
      setAssigning(false);
    }
  };

  return (
    <DashboardLayout title="Classes">
      <div className="space-y-6">
        <div className="page-header flex items-center justify-between">
          <div>
            <h1 className="page-title">Classes</h1>
            <p className="page-subtitle">
              {context ? `${context.curriculum_type === "cbc" ? "CBC" : "8-4-4"} · ` : ""}
              Manage classes, streams, and class teacher assignments
            </p>
          </div>
          <Button onClick={() => setShowCreate((v) => !v)}>
            <Plus className="h-4 w-4 mr-2" /> New class
          </Button>
        </div>

        {error && (
          <p className="text-sm text-error flex items-center gap-1.5">
            <AlertTriangle className="h-4 w-4" /> {error}
          </p>
        )}

        {showCreate && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Create a class</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Name</label>
                  <input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Grade 7 / Form 3"
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Stream</label>
                  <input
                    value={stream}
                    onChange={(e) => setStream(e.target.value)}
                    placeholder="A"
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Level</label>
                  <input
                    type="number"
                    value={level}
                    onChange={(e) => setLevel(e.target.value)}
                    placeholder="7"
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Capacity</label>
                  <input
                    type="number"
                    value={capacity}
                    onChange={(e) => setCapacity(e.target.value)}
                    placeholder="40"
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div className="col-span-2">
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Academic year</label>
                  <select
                    value={academicYearId}
                    onChange={(e) => setAcademicYearId(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  >
                    <option value="">Choose&hellip;</option>
                    {years.map((y) => (
                      <option key={y.id} value={y.id}>
                        {y.name} {y.is_current ? "(current)" : ""}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <p className="mt-3 text-xs text-neutral-400">
                Class teacher can be assigned now or later — a class without one shows as needing setup, and ordinary teachers can&apos;t take daily attendance for it until it&apos;s assigned.
              </p>
              <div className="mt-4">
                <Button onClick={handleCreate} disabled={creating || !name || !level || !capacity || !academicYearId}>
                  {creating ? <><Loader2 className="h-4 w-4 mr-2 animate-spin" /> Creating&hellip;</> : "Create class"}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardContent className="p-0">
            {loading ? (
              <div className="p-8 text-center text-sm text-neutral-400">Loading classes&hellip;</div>
            ) : classes.length === 0 ? (
              <div className="p-8 text-center text-sm text-neutral-400">No classes yet. Create the first one above.</div>
            ) : (
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-100">
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Class</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Level</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Class Teacher</th>
                    <th className="text-left px-6 py-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">Enrollment</th>
                    <th className="px-6 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-50">
                  {classes.map((c) => (
                    <tr key={c.id} className="hover:bg-neutral-50">
                      <td className="px-6 py-3.5 text-sm font-medium text-neutral-800">
                        {c.name} {c.stream ?? ""}
                      </td>
                      <td className="px-6 py-3.5 text-sm text-neutral-600">{c.level}</td>
                      <td className="px-6 py-3.5 text-sm">
                        {c.class_teacher ? (
                          <span className="text-neutral-800">{c.class_teacher}</span>
                        ) : (
                          <span className="inline-flex items-center gap-1 text-warning">
                            <AlertTriangle className="h-3.5 w-3.5" /> Needs class teacher
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-3.5 text-sm text-neutral-600">
                        {c.enrollment} / {c.capacity}
                      </td>
                      <td className="px-6 py-3.5 text-right">
                        {assigningClassId === c.id ? (
                          <div className="flex items-center gap-2 justify-end">
                            <select
                              value={assignTeacherId}
                              onChange={(e) => setAssignTeacherId(e.target.value)}
                              className="rounded-md border border-neutral-200 px-2 py-1.5 text-sm"
                            >
                              <option value="">Choose teacher&hellip;</option>
                              {teachers.map((t) => (
                                <option key={t.id} value={t.id}>{t.full_name}</option>
                              ))}
                            </select>
                            <Button size="sm" disabled={assigning || !assignTeacherId} onClick={() => handleAssign(c.id)}>
                              {assigning ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Save"}
                            </Button>
                            <Button size="sm" variant="ghost" onClick={() => { setAssigningClassId(null); setAssignTeacherId(""); }}>
                              Cancel
                            </Button>
                          </div>
                        ) : (
                          <Button size="sm" variant="outline" onClick={() => setAssigningClassId(c.id)}>
                            <UserCheck className="h-4 w-4 mr-1.5" />
                            {c.class_teacher ? "Reassign" : "Assign teacher"}
                          </Button>
                        )}
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