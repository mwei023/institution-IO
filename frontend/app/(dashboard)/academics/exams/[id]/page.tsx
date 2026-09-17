"use client";
import React, { useEffect, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import { AlertTriangle, Loader2, CheckCircle2 } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { examsAPI, academicsAPI } from "@/lib/api";

interface SittingView {
  sitting: {
    id: string; exam_paper_id: string; exam_date: string;
    start_time: string; end_time: string;
    room_name: string | null; room_capacity: number | null;
    candidate_count: number | null; seating_note?: string | null; status: string;
  };
  paper: { id: string; paper_name: string; paper_code: string | null; duration_minutes: number } | null;
  groups: Array<{ id: string; class_id: string | null; class_name: string | null; label: string | null }>;
  invigilators: Array<{ staff_id: string; staff_name: string | null; role: string }>;
  capacity_status: string;
}

interface Issue { code: string; severity: string; message: string; sitting_id?: string }
interface Validation { valid: boolean; blocking: Issue[]; warnings: Issue[]; sittings_count: number }

type ViewTab = "master" | "candidate" | "paper" | "room" | "invigilator";

const groupLabel = (g: SittingView["groups"][number]) =>
  g.class_name ?? g.label ?? (g.class_id ? g.class_id.slice(0, 8) : "—");

export default function ExamSessionDetailPage() {
  const params = useParams();
  const sessionId = params.id as string;

  const [session, setSession] = useState<any>(null);
  const [papers, setPapers] = useState<any[]>([]);
  const [views, setViews] = useState<SittingView[]>([]);
  const [validation, setValidation] = useState<Validation | null>(null);
  const [tab, setTab] = useState<ViewTab>("master");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // form state
  const [paperName, setPaperName] = useState("");
  const [duration, setDuration] = useState("120");
  const [groupPaperId, setGroupPaperId] = useState("");
  const [groupClassId, setGroupClassId] = useState("");
  const [groupLabelText, setGroupLabelText] = useState("");
  const [sitPaperId, setSitPaperId] = useState("");
  const [sitDate, setSitDate] = useState("");
  const [sitStart, setSitStart] = useState("");
  const [sitEnd, setSitEnd] = useState("");
  const [sitRoom, setSitRoom] = useState("");
  const [sitCapacity, setSitCapacity] = useState("");
  const [sitCandidates, setSitCandidates] = useState("");
  const [editSeats, setEditSeats] = useState<Record<string, { capacity: string; candidates: string }>>({});
  const [classes, setClasses] = useState<any[]>([]);
  const [filterClassId, setFilterClassId] = useState("");
  const [filterPaperId, setFilterPaperId] = useState("");
  const [filterRoom, setFilterRoom] = useState("");
  const [filterStaff, setFilterStaff] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [sessRes, papersRes, masterRes, classesRes] = await Promise.all([
        examsAPI.getSession(sessionId),
        examsAPI.listPapers(sessionId),
        examsAPI.master(sessionId),
        academicsAPI.listClasses(),
      ]);
      setSession(sessRes.data?.data);
      const plist = papersRes.data?.data ?? [];
      setPapers(plist);
      setViews(masterRes.data?.data ?? []);
      setClasses(classesRes.data?.data ?? []);
      if (plist.length > 0) {
        setGroupPaperId((v) => v || plist[0].id);
        setSitPaperId((v) => v || plist[0].id);
        setFilterPaperId((v) => v || plist[0].id);
      }
      const clist = classesRes.data?.data ?? [];
      if (clist.length > 0) setFilterClassId((v) => v || clist[0].id);
    } catch {
      setError("Couldn't load this examination session.");
    } finally {
      setLoading(false);
    }
  }, [sessionId]);

  useEffect(() => {
    load();
  }, [load]);

  const refreshViews = async (nextTab: ViewTab = tab) => {
    try {
      let res;
      if (nextTab === "candidate" && filterClassId) res = await examsAPI.candidateView(sessionId, filterClassId);
      else if (nextTab === "paper" && filterPaperId) res = await examsAPI.paperView(sessionId, filterPaperId);
      else if (nextTab === "room" && filterRoom) res = await examsAPI.roomView(sessionId, filterRoom);
      else if (nextTab === "invigilator" && filterStaff) res = await examsAPI.invigilatorView(sessionId, filterStaff);
      else res = await examsAPI.master(sessionId);
      setViews(res.data?.data ?? []);
    } catch {
      setError("Couldn't load this timetable view.");
    }
  };

  const switchTab = (t: ViewTab) => {
    setTab(t);
    refreshViews(t);
  };

  const action = async (fn: () => Promise<any>, okMsg?: string) => {
    setBusy(true);
    setError(null);
    try {
      await fn();
      await load();
      const v = await examsAPI.master(sessionId);
      setViews(v.data?.data ?? []);
      if (okMsg) setError(null);
    } catch (err: any) {
      setError(err?.response?.data?.error?.message ?? "Action failed.");
    } finally {
      setBusy(false);
    }
  };

  const isPublished = session?.status === "published";
  const draftBanner = isPublished
    ? "Published — editing is locked. Reopen to draft to make changes (version will increment)."
    : "Draft — edit freely, then validate and publish.";

  return (
    <DashboardLayout title="Exam Session">
      <div className="space-y-6">
        <div className="page-header">
          <h1 className="page-title">{session?.name ?? "Examination session"}</h1>
          <p className="page-subtitle">
            {session ? `${session.exam_type} · ${session.start_date} → ${session.end_date} · ${session.status} · v${session.version}` : "Loading…"}
          </p>
          {session && <p className="mt-1 text-xs text-neutral-500">{draftBanner}</p>}
        </div>

        {error && (
          <p className="text-sm text-error flex items-center gap-1.5">
            <AlertTriangle className="h-4 w-4" /> {error}
          </p>
        )}

        {loading ? (
          <div className="p-8 text-center text-sm text-neutral-400">Loading…</div>
        ) : (
          <>
            {!isPublished && (
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
                <Card>
                  <CardHeader className="pb-3"><CardTitle>Add paper</CardTitle></CardHeader>
                  <CardContent>
                    <input value={paperName} onChange={(e) => setPaperName(e.target.value)}
                      placeholder="Paper name (e.g. Mathematics P1)"
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm mb-2" />
                    <input type="number" min={1} value={duration} onChange={(e) => setDuration(e.target.value)}
                      placeholder="Duration (minutes)"
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm mb-3" />
                    <Button disabled={busy || !paperName || !duration}
                      onClick={() => action(() => examsAPI.createPaper(sessionId, {
                        paper_name: paperName, duration_minutes: parseInt(duration, 10),
                      })).then(() => setPaperName(""))}>
                      Add paper
                    </Button>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-3"><CardTitle>Assign candidate group</CardTitle></CardHeader>
                  <CardContent>
                    <select value={groupPaperId} onChange={(e) => setGroupPaperId(e.target.value)}
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm mb-2">
                      {papers.map((p) => <option key={p.id} value={p.id}>{p.paper_name}</option>)}
                    </select>
                    <select value={groupClassId} onChange={(e) => setGroupClassId(e.target.value)}
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm mb-2">
                      <option value="">Class…</option>
                      {classes.map((c) => (
                        <option key={c.id} value={c.id}>{c.name} {c.stream ?? ""}</option>
                      ))}
                    </select>
                    <input value={groupLabelText} onChange={(e) => setGroupLabelText(e.target.value)}
                      placeholder="Or programme/level label (e.g. Form 4 cohort)"
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm mb-3" />
                    <Button disabled={busy || !groupPaperId || (!groupClassId && !groupLabelText)}
                      onClick={() => action(() => examsAPI.addPaperGroup(groupPaperId, {
                        class_id: groupClassId || undefined,
                        label: groupLabelText || undefined,
                      }))}>
                      Assign group
                    </Button>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-3"><CardTitle>Schedule sitting</CardTitle></CardHeader>
                  <CardContent>
                    <select value={sitPaperId} onChange={(e) => setSitPaperId(e.target.value)}
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm mb-2">
                      {papers.map((p) => <option key={p.id} value={p.id}>{p.paper_name}</option>)}
                    </select>
                    <div className="grid grid-cols-3 gap-2 mb-2">
                      <input type="date" value={sitDate} onChange={(e) => setSitDate(e.target.value)}
                        className="rounded-md border border-neutral-200 px-2 py-2 text-sm" />
                      <input type="time" value={sitStart} onChange={(e) => setSitStart(e.target.value)}
                        className="rounded-md border border-neutral-200 px-2 py-2 text-sm" />
                      <input type="time" value={sitEnd} onChange={(e) => setSitEnd(e.target.value)}
                        className="rounded-md border border-neutral-200 px-2 py-2 text-sm" />
                    </div>
                    <input value={sitRoom} onChange={(e) => setSitRoom(e.target.value)}
                      placeholder="Room (optional)"
                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm mb-2" />
                    <div className="grid grid-cols-2 gap-2 mb-3">
                      <input type="number" min={1} value={sitCapacity} onChange={(e) => setSitCapacity(e.target.value)}
                        placeholder="Room capacity"
                        className="rounded-md border border-neutral-200 px-3 py-2 text-sm" />
                      <input type="number" min={1} value={sitCandidates} onChange={(e) => setSitCandidates(e.target.value)}
                        placeholder="Candidates"
                        className="rounded-md border border-neutral-200 px-3 py-2 text-sm" />
                    </div>
                    <Button disabled={busy || !sitPaperId || !sitDate || !sitStart || !sitEnd}
                      onClick={() => action(() => examsAPI.createSitting(sessionId, {
                        exam_paper_id: sitPaperId, exam_date: sitDate,
                        start_time: sitStart, end_time: sitEnd,
                        room_name: sitRoom || undefined,
                        room_capacity: sitCapacity ? parseInt(sitCapacity, 10) : undefined,
                        candidate_count: sitCandidates ? parseInt(sitCandidates, 10) : undefined,
                      })).then(() => { setSitCapacity(""); setSitCandidates(""); })}>
                      Schedule
                    </Button>
                  </CardContent>
                </Card>
              </div>
            )}

            <Card>
              <CardHeader className="pb-3">
                <CardTitle>Validation & publication</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2 mb-4">
                  <Button disabled={busy} onClick={() => action(async () => {
                    const res = await examsAPI.validate(sessionId);
                    setValidation(res.data?.data);
                  })}>
                    {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : "Validate draft"}
                  </Button>
                  {!isPublished && (
                    <Button disabled={busy}
                      onClick={async () => {
                        if (!window.confirm("Publish this timetable? Published schedules are locked.")) return;
                        await action(() => examsAPI.publish(sessionId));
                      }}>
                      Publish
                    </Button>
                  )}
                  {isPublished && (
                    <Button disabled={busy} onClick={() => action(() => examsAPI.reopen(sessionId))}>
                      Reopen to draft (new version)
                    </Button>
                  )}
                </div>
                {validation ? (
                  <div className="space-y-2 text-sm">
                    <p className="flex items-center gap-1.5">
                      {validation.valid
                        ? <><CheckCircle2 className="h-4 w-4 text-green-600" /> No blocking errors — safe to publish ({validation.sittings_count} sittings).</>
                        : <><AlertTriangle className="h-4 w-4 text-error" /> {validation.blocking.length} blocking error(s) — cannot publish.</>}
                    </p>
                    {validation.blocking.map((b, i) => (
                      <p key={`b${i}`} className="text-error">⛔ [{b.code}] {b.message}</p>
                    ))}
                    {validation.warnings.map((w, i) => (
                      <p key={`w${i}`} className="text-amber-700">⚠️ [{w.code}] {w.message}</p>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-neutral-400">Run validation to check candidate, room, invigilator and capacity conflicts.</p>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle>Timetable views (all derive from the canonical schedule)</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap items-center gap-2 mb-4 text-sm">
                  {(["master", "candidate", "paper", "room", "invigilator"] as ViewTab[]).map((t) => (
                    <button key={t} onClick={() => switchTab(t)}
                      className={`px-3 py-1.5 rounded-md border ${tab === t ? "bg-neutral-800 text-white" : "text-neutral-600"}`}>
                      {t}
                    </button>
                  ))}
                  {tab === "candidate" && (
                    <select value={filterClassId} onChange={(e) => { setFilterClassId(e.target.value); }}
                      onBlur={() => refreshViews("candidate")}
                      className="rounded-md border border-neutral-200 px-2 py-1.5">
                      {classes.map((c) => <option key={c.id} value={c.id}>{c.name} {c.stream ?? ""}</option>)}
                    </select>
                  )}
                  {tab === "paper" && (
                    <select value={filterPaperId} onChange={(e) => setFilterPaperId(e.target.value)}
                      onBlur={() => refreshViews("paper")}
                      className="rounded-md border border-neutral-200 px-2 py-1.5">
                      {papers.map((p) => <option key={p.id} value={p.id}>{p.paper_name}</option>)}
                    </select>
                  )}
                  {tab === "room" && (
                    <input value={filterRoom} onChange={(e) => setFilterRoom(e.target.value)}
                      onBlur={() => refreshViews("room")} placeholder="Room name"
                      className="rounded-md border border-neutral-200 px-2 py-1.5" />
                  )}
                  {tab === "invigilator" && (
                    <input value={filterStaff} onChange={(e) => setFilterStaff(e.target.value)}
                      onBlur={() => refreshViews("invigilator")} placeholder="Staff UUID"
                      className="rounded-md border border-neutral-200 px-2 py-1.5" />
                  )}
                  <Button onClick={() => refreshViews()}>Apply</Button>
                </div>
                {views.length === 0 ? (
                  <p className="text-sm text-neutral-400 p-4 text-center">No sittings in this view yet.</p>
                ) : (
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-neutral-100">
                        <th className="text-left px-4 py-2 text-xs font-medium text-neutral-500 uppercase">Date / time</th>
                        <th className="text-left px-4 py-2 text-xs font-medium text-neutral-500 uppercase">Paper</th>
                        <th className="text-left px-4 py-2 text-xs font-medium text-neutral-500 uppercase">Groups</th>
                        <th className="text-left px-4 py-2 text-xs font-medium text-neutral-500 uppercase">Room</th>
                        <th className="text-left px-4 py-2 text-xs font-medium text-neutral-500 uppercase">Invigilators</th>
                        <th className="text-left px-4 py-2 text-xs font-medium text-neutral-500 uppercase">Capacity</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-neutral-50">
                      {views.map((v) => (
                        <tr key={v.sitting.id} className="hover:bg-neutral-50 text-sm">
                          <td className="px-4 py-2 whitespace-nowrap">{v.sitting.exam_date} {v.sitting.start_time}–{v.sitting.end_time}</td>
                          <td className="px-4 py-2 font-medium">{v.paper?.paper_name ?? "—"}</td>
                          <td className="px-4 py-2 text-neutral-600">{(v.groups ?? []).map(groupLabel).join(", ") || "—"}</td>
                          <td className="px-4 py-2">{v.sitting.room_name ?? "—"}</td>
                          <td className="px-4 py-2 text-neutral-600">
                            {(v.invigilators ?? []).map((a) => `${a.staff_name ?? a.staff_id?.slice(0, 8) ?? "—"} (${a.role})`).join(", ") || "—"}
                          </td>
                          <td className="px-4 py-2">
                            <span className={`inline-block rounded px-2 py-0.5 text-xs ${
                              v.capacity_status === "exceeded" ? "bg-red-100 text-red-800"
                              : v.capacity_status === "verified" ? "bg-green-100 text-green-800"
                              : "bg-amber-100 text-amber-800"}`}>
                              {v.capacity_status}
                            </span>
                            <span className="ml-2 text-xs text-neutral-500">
                              {v.sitting.candidate_count ?? "—"}/{v.sitting.room_capacity ?? "—"}
                            </span>
                            {!isPublished && (
                              <span className="mt-1 flex items-center gap-1">
                                <input
                                  type="number" min={1} placeholder="Cap."
                                  title="Room capacity"
                                  value={editSeats[v.sitting.id]?.capacity ?? String(v.sitting.room_capacity ?? "")}
                                  onChange={(e) => setEditSeats((prev) => ({
                                    ...prev,
                                    [v.sitting.id]: {
                                      capacity: e.target.value,
                                      candidates: prev[v.sitting.id]?.candidates ?? String(v.sitting.candidate_count ?? ""),
                                    },
                                  }))}
                                  className="w-20 rounded border border-neutral-200 px-1.5 py-1 text-xs" />
                                <input
                                  type="number" min={1} placeholder="Cand."
                                  title="Candidate count"
                                  value={editSeats[v.sitting.id]?.candidates ?? String(v.sitting.candidate_count ?? "")}
                                  onChange={(e) => setEditSeats((prev) => ({
                                    ...prev,
                                    [v.sitting.id]: {
                                      capacity: prev[v.sitting.id]?.capacity ?? String(v.sitting.room_capacity ?? ""),
                                      candidates: e.target.value,
                                    },
                                  }))}
                                  className="w-20 rounded border border-neutral-200 px-1.5 py-1 text-xs" />
                                <Button
                                  disabled={busy}
                                  onClick={() => {
                                    const cur = editSeats[v.sitting.id] ?? {
                                      capacity: String(v.sitting.room_capacity ?? ""),
                                      candidates: String(v.sitting.candidate_count ?? ""),
                                    };
                                    action(() => examsAPI.updateSitting(v.sitting.id, {
                                      room_capacity: cur.capacity ? parseInt(cur.capacity, 10) : null,
                                      candidate_count: cur.candidates ? parseInt(cur.candidates, 10) : null,
                                    }));
                                  }}>
                                  Save
                                </Button>
                              </span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </DashboardLayout>
  );
}
