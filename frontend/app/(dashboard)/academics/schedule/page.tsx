"use client";
import React, { useEffect, useState, useCallback } from "react";
import { Plus, AlertTriangle, Loader2, Trash2 } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { academicsAPI } from "@/lib/api";

interface ScheduleProfile {
  id: string;
  term_id: string | null;
  name: string;
  is_default: boolean;
}

interface ScheduleSlot {
  id: string;
  profile_id: string;
  day_of_week: number;
  sequence: number;
  start_time: string;
  end_time: string;
  slot_type: "lesson" | "break" | "lunch" | "assembly" | "activity";
  is_instructional: boolean;
  is_active: boolean;
}

const DAYS = [
  { value: 1, label: "Monday" },
  { value: 2, label: "Tuesday" },
  { value: 3, label: "Wednesday" },
  { value: 4, label: "Thursday" },
  { value: 5, label: "Friday" },
  { value: 6, label: "Saturday" },
  { value: 7, label: "Sunday" },
];

const SLOT_TYPES = ["lesson", "break", "lunch", "assembly", "activity"] as const;

const TYPE_STYLES: Record<string, string> = {
  lesson: "bg-neutral-100 text-neutral-700",
  break: "bg-amber-100 text-amber-800",
  lunch: "bg-orange-100 text-orange-800",
  assembly: "bg-blue-100 text-blue-800",
  activity: "bg-purple-100 text-purple-800",
};

const dayLabel = (d: number) => DAYS.find((x) => x.value === d)?.label ?? `Day ${d}`;

export default function ScheduleConfigPage() {
  const [profiles, setProfiles] = useState<ScheduleProfile[]>([]);
  const [profileId, setProfileId] = useState("");
  const [slots, setSlots] = useState<ScheduleSlot[]>([]);

  const [loading, setLoading] = useState(true);
  const [loadingSlots, setLoadingSlots] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // New profile form
  const [showNewProfile, setShowNewProfile] = useState(false);
  const [profileName, setProfileName] = useState("");
  const [profileDefault, setProfileDefault] = useState(false);
  const [creatingProfile, setCreatingProfile] = useState(false);

  // New slot form — days is a multi-select defaulting to Mon–Fri so one
  // entry covers the whole regular week; each day stays an independent
  // row that an admin can later edit on its own to diverge.
  const [selectedDays, setSelectedDays] = useState<number[]>([1, 2, 3, 4, 5]);
  const [sequence, setSequence] = useState("");
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
  const [slotType, setSlotType] = useState<string>("lesson");
  const [instructional, setInstructional] = useState(true);
  const [creatingSlot, setCreatingSlot] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  // Copy-day form
  const [copyFrom, setCopyFrom] = useState("1");
  const [copyTo, setCopyTo] = useState<number[]>([2, 3, 4, 5]);
  const [copyOverwrite, setCopyOverwrite] = useState(false);
  const [copying, setCopying] = useState(false);

  // Inline edit form (corrections without delete + recreate)
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editStart, setEditStart] = useState("");
  const [editEnd, setEditEnd] = useState("");
  const [editType, setEditType] = useState<string>("lesson");
  const [editInstructional, setEditInstructional] = useState(true);
  const [editScope, setEditScope] = useState<"day" | "propagate">("day");
  const [editTargetDays, setEditTargetDays] = useState<number[]>([]);
  const [savingEdit, setSavingEdit] = useState(false);

  const loadProfiles = useCallback(async (selectId?: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await academicsAPI.listScheduleProfiles();
      const list: ScheduleProfile[] = res.data?.data ?? [];
      setProfiles(list);
      if (selectId) {
        setProfileId(selectId);
      } else if (list.length > 0) {
        const def = list.find((p) => p.is_default) ?? list[0];
        setProfileId((prev) => prev || def.id);
      }
    } catch {
      setError("Couldn't load schedule profiles. Try refreshing.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProfiles();
  }, [loadProfiles]);

  const loadSlots = useCallback(async () => {
    if (!profileId) {
      setSlots([]);
      return;
    }
    setLoadingSlots(true);
    setError(null);
    try {
      const res = await academicsAPI.listScheduleSlots(profileId);
      setSlots(res.data?.data ?? []);
    } catch {
      setError("Couldn't load slots for this profile.");
    } finally {
      setLoadingSlots(false);
    }
  }, [profileId]);

  useEffect(() => {
    loadSlots();
  }, [loadSlots]);

  const handleCreateProfile = async () => {
    if (!profileName.trim()) return;
    setCreatingProfile(true);
    setError(null);
    try {
      const res = await academicsAPI.createScheduleProfile({
        name: profileName.trim(),
        is_default: profileDefault || profiles.length === 0,
      });
      const created: ScheduleProfile = res.data?.data;
      setProfileName("");
      setProfileDefault(false);
      setShowNewProfile(false);
      await loadProfiles(created?.id);
    } catch (err: any) {
      setError(err?.response?.data?.error?.message ?? "Couldn't create this profile.");
    } finally {
      setCreatingProfile(false);
    }
  };

  const handleCreateSlot = async () => {
    if (!profileId || selectedDays.length === 0 || !sequence || !startTime || !endTime) return;
    setCreatingSlot(true);
    setError(null);
    setNotice(null);
    try {
      const payload = {
        sequence: parseInt(sequence, 10),
        start_time: startTime,
        end_time: endTime,
        slot_type: slotType,
        is_instructional: instructional,
      };
      if (selectedDays.length === 1) {
        await academicsAPI.createScheduleSlot(profileId, { ...payload, day_of_week: selectedDays[0] });
        setNotice(`Added to ${dayLabel(selectedDays[0])}.`);
      } else {
        const res = await academicsAPI.bulkCreateScheduleSlots(profileId, { ...payload, days: selectedDays });
        const created = res.data?.data?.created ?? [];
        const failed = res.data?.data?.failed ?? [];
        const parts = [`Added to ${created.length} day${created.length === 1 ? "" : "s"}.`];
        failed.forEach((f: { day_of_week: number; error: string }) => {
          parts.push(`${dayLabel(f.day_of_week)} skipped: ${f.error}`);
        });
        setNotice(parts.join(" "));
      }
      setSequence("");
      setStartTime("");
      setEndTime("");
      await loadSlots();
    } catch (err: any) {
      // Sequence clashes and overlaps come back as 409s with the real
      // backend message — surfaced as-is so the admin knows what to fix.
      setError(err?.response?.data?.error?.message ?? "Couldn't create this slot.");
    } finally {
      setCreatingSlot(false);
    }
  };

  const handleCopyDay = async () => {
    if (!profileId || copyTo.length === 0) return;
    setCopying(true);
    setError(null);
    setNotice(null);
    try {
      const res = await academicsAPI.copyScheduleDay(profileId, {
        from_day: parseInt(copyFrom, 10),
        to_days: copyTo,
        overwrite: copyOverwrite,
      });
      const created = res.data?.data?.created ?? [];
      const failed = res.data?.data?.failed ?? [];
      const parts = [`Copied ${dayLabel(parseInt(copyFrom, 10))} → ${created.length} block${created.length === 1 ? "" : "s"}.`];
      failed.forEach((f: { day_of_week: number; sequence: number; error: string }) => {
        parts.push(`${dayLabel(f.day_of_week)} #${f.sequence} skipped: ${f.error}`);
      });
      setNotice(parts.join(" "));
      await loadSlots();
    } catch (err: any) {
      setError(err?.response?.data?.error?.message ?? "Couldn't copy this day.");
    } finally {
      setCopying(false);
    }
  };

  const startEditing = (slot: ScheduleSlot) => {
    setEditingId(slot.id);
    setEditStart(slot.start_time);
    setEditEnd(slot.end_time);
    setEditType(slot.slot_type);
    setEditInstructional(slot.is_instructional);
    setEditScope("day");
    // Default propagate targets: Mon–Fri except the slot's own day.
    setEditTargetDays([1, 2, 3, 4, 5].filter((d) => d !== slot.day_of_week));
    setError(null);
    setNotice(null);
  };

  const handleSaveEdit = async (slot: ScheduleSlot) => {
    if (!editStart || !editEnd) return;
    setSavingEdit(true);
    setError(null);
    setNotice(null);
    try {
      const patch = {
        start_time: editStart,
        end_time: editEnd,
        slot_type: editType,
        is_instructional: editInstructional,
      };
      if (editScope === "propagate" && editTargetDays.length > 0) {
        const res = await academicsAPI.propagateScheduleSlot(slot.id, { ...patch, target_days: editTargetDays });
        const updated = res.data?.data?.updated ?? [];
        const skipped = res.data?.data?.skipped ?? [];
        const parts = [`Updated ${updated.length} day${updated.length === 1 ? "" : "s"}.`];
        skipped.forEach((f: { day_of_week: number; error: string }) => {
          parts.push(`${dayLabel(f.day_of_week)} skipped: ${f.error}`);
        });
        setNotice(parts.join(" "));
      } else {
        await academicsAPI.updateScheduleSlot(slot.id, patch);
        setNotice(`Updated ${dayLabel(slot.day_of_week)} #${slot.sequence}.`);
      }
      setEditingId(null);
      await loadSlots();
    } catch (err: any) {
      setError(err?.response?.data?.error?.message ?? "Couldn't save this change.");
    } finally {
      setSavingEdit(false);
    }
  };

  const handleToggleActive = async (slot: ScheduleSlot) => {
    setError(null);
    try {
      await academicsAPI.updateScheduleSlot(slot.id, { is_active: !slot.is_active });
      await loadSlots();
    } catch (err: any) {
      setError(err?.response?.data?.error?.message ?? "Couldn't update this slot.");
    }
  };

  const handleDelete = async (slot: ScheduleSlot) => {
    if (!window.confirm(`Delete ${dayLabel(slot.day_of_week)} sequence ${slot.sequence} (${slot.start_time}-${slot.end_time})?`)) return;
    setError(null);
    try {
      await academicsAPI.deleteScheduleSlot(slot.id);
      await loadSlots();
    } catch {
      setError("Couldn't delete this slot.");
    }
  };

  const suggestSequence = (days: number[]) => {
    const max = slots.filter((s) => days.includes(s.day_of_week)).reduce((m, s) => Math.max(m, s.sequence), 0);
    setSequence(String(max + 1));
  };

  const toggleDay = (days: number[], setDays: (d: number[]) => void, day: number) => {
    const next = days.includes(day) ? days.filter((d) => d !== day) : [...days, day].sort((a, b) => a - b);
    setDays(next);
    return next;
  };

  // Timetable grid: days down the side, periods (sequences) across the
  // top — the standard easy-to-scan layout. Cells are looked up per
  // (day, sequence); days stay independent so divergent times still show
  // correctly per cell.
  const visibleDays = DAYS.filter((d) => slots.some((s) => s.day_of_week === d.value));
  const allSequences = Array.from(new Set(slots.map((s) => s.sequence))).sort((a, b) => a - b);
  const slotAt = (day: number, seq: number) => slots.find((s) => s.day_of_week === day && s.sequence === seq);
  const rowTimeLabel = (seq: number) => {
    const times = new Set(
      visibleDays.map((d) => slotAt(d.value, seq)).filter(Boolean).map((s) => `${s!.start_time}–${s!.end_time}`)
    );
    return times.size === 1 ? [...times][0] : null;
  };
  const editingSlot = editingId ? slots.find((s) => s.id === editingId) ?? null : null;
  const editingRowDay = editingSlot?.day_of_week ?? null;

  const renderCell = (day: number, seq: number) => {
    const s = slotAt(day, seq);
    if (!s) return <span className="text-neutral-300">—</span>;
    return (
      <div className={s.is_active ? "" : "opacity-50"}>
        <div className="whitespace-nowrap text-sm font-medium text-neutral-800">{s.start_time}–{s.end_time}</div>
        <div className="mt-1">
          <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${TYPE_STYLES[s.slot_type] ?? TYPE_STYLES.lesson}`}>
            {s.slot_type}
          </span>
          {!s.is_instructional && s.slot_type === "lesson" && (
            <span className="ml-1 text-xs text-neutral-400">non-instr.</span>
          )}
        </div>
        <div className="mt-1.5 flex flex-wrap gap-x-2 gap-y-1">
          <button
            onClick={() => (editingId === s.id ? setEditingId(null) : startEditing(s))}
            className="text-xs text-neutral-500 underline"
          >
            Edit
          </button>
          <button
            onClick={() => handleToggleActive(s)}
            className="text-xs text-neutral-500 underline"
          >
            {s.is_active ? "Off" : "On"}
          </button>
          <button
            onClick={() => handleDelete(s)}
            className="text-xs text-error underline inline-flex items-center gap-0.5"
          >
            <Trash2 className="h-3 w-3" /> Del
          </button>
        </div>
      </div>
    );
  };

  return (
    <DashboardLayout title="Bell Schedule">
      <div className="space-y-6">
        <div className="page-header flex items-center justify-between">
          <div>
            <h1 className="page-title">Bell Schedule</h1>
            <p className="page-subtitle">Define when the school teaches — days, periods, breaks, lunch, assembly</p>
          </div>
          <Button onClick={() => setShowNewProfile((v) => !v)}>
            <Plus className="h-4 w-4 mr-2" /> New profile
          </Button>
        </div>

        {error && (
          <p className="text-sm text-error flex items-center gap-1.5">
            <AlertTriangle className="h-4 w-4" /> {error}
          </p>
        )}

        {notice && (
          <p className="text-sm text-success">
            {notice}
          </p>
        )}

        {showNewProfile && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>New schedule profile</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 items-end">
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Name</label>
                  <input
                    value={profileName}
                    onChange={(e) => setProfileName(e.target.value)}
                    placeholder="e.g. Regular week, Saturday schedule"
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <label className="flex items-center gap-2 text-sm text-neutral-700">
                  <input
                    type="checkbox"
                    checked={profileDefault || profiles.length === 0}
                    onChange={(e) => setProfileDefault(e.target.checked)}
                    disabled={profiles.length === 0}
                  />
                  Default profile
                  {profiles.length === 0 && <span className="text-xs text-neutral-400">(first profile is default)</span>}
                </label>
                <div>
                  <Button onClick={handleCreateProfile} disabled={creatingProfile || !profileName.trim()}>
                    {creatingProfile ? <><Loader2 className="h-4 w-4 mr-2 animate-spin" /> Creating&hellip;</> : "Create profile"}
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardContent className="pt-6">
            <label className="block text-xs font-medium text-neutral-500 mb-1.5">Schedule profile</label>
            <select
              value={profileId}
              onChange={(e) => { setProfileId(e.target.value); setNotice(null); setError(null); setEditingId(null); }}
              className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
            >
              {loading && <option>Loading&hellip;</option>}
              {profiles.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} {p.is_default ? "(default)" : ""}
                </option>
              ))}
            </select>
            {profiles.length === 0 && !loading && (
              <p className="mt-3 text-xs text-warning flex items-center gap-1.5">
                <AlertTriangle className="h-3.5 w-3.5" />
                No schedule profile yet — create one above, then add its periods, breaks and lunch blocks.
              </p>
            )}
          </CardContent>
        </Card>

        {profileId && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Add a time block</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="mb-4">
                <label className="block text-xs font-medium text-neutral-500 mb-1.5">
                  Days — one entry is created per selected day (Mon–Fri covers the regular week)
                </label>
                <div className="flex flex-wrap gap-2">
                  {DAYS.map((d) => (
                    <label key={d.value} className={`inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm cursor-pointer ${selectedDays.includes(d.value) ? "border-neutral-800 bg-neutral-100" : "border-neutral-200"}`}>
                      <input
                        type="checkbox"
                        checked={selectedDays.includes(d.value)}
                        onChange={() => {
                          const next = toggleDay(selectedDays, setSelectedDays, d.value);
                          suggestSequence(next);
                        }}
                      />
                      {d.label.slice(0, 3)}
                    </label>
                  ))}
                  <button
                    onClick={() => { setSelectedDays([1, 2, 3, 4, 5]); suggestSequence([1, 2, 3, 4, 5]); }}
                    className="text-xs text-neutral-500 underline self-center ml-1"
                  >
                    Mon–Fri
                  </button>
                  <button
                    onClick={() => { setSelectedDays([1, 2, 3, 4, 5, 6, 7]); suggestSequence([1, 2, 3, 4, 5, 6, 7]); }}
                    className="text-xs text-neutral-500 underline self-center ml-1"
                  >
                    All week
                  </button>
                </div>
              </div>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Sequence</label>
                  <input
                    type="number" min={1}
                    value={sequence}
                    onChange={(e) => setSequence(e.target.value)}
                    placeholder="Position in the day"
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Start</label>
                  <input
                    type="time" value={startTime}
                    onChange={(e) => setStartTime(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">End</label>
                  <input
                    type="time" value={endTime}
                    onChange={(e) => setEndTime(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">Type</label>
                  <select
                    value={slotType}
                    onChange={(e) => {
                      setSlotType(e.target.value);
                      setInstructional(e.target.value === "lesson");
                    }}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  >
                    {SLOT_TYPES.map((t) => (
                      <option key={t} value={t}>{t}</option>
                    ))}
                  </select>
                </div>
                <label className="flex items-center gap-2 text-sm text-neutral-700 self-end pb-2">
                  <input
                    type="checkbox" checked={instructional}
                    onChange={(e) => setInstructional(e.target.checked)}
                  />
                  Counts as teaching time
                </label>
                <div className="self-end pb-0.5">
                  <Button
                    onClick={handleCreateSlot}
                    disabled={creatingSlot || selectedDays.length === 0 || !sequence || !startTime || !endTime}
                  >
                    {creatingSlot ? <><Loader2 className="h-4 w-4 mr-2 animate-spin" /> Adding&hellip;</> : selectedDays.length > 1 ? `Add to ${selectedDays.length} days` : "Add block"}
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {profileId && slots.length > 0 && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Copy a day</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-neutral-500 mb-3">
                Built Monday only? Copy the whole day to Tuesday–Friday instead of re-entering every block.
                Days keep independent rows afterwards, so editing one day later won&apos;t touch the others.
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 items-start">
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">From</label>
                  <select
                    value={copyFrom}
                    onChange={(e) => setCopyFrom(e.target.value)}
                    className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                  >
                    {DAYS.map((d) => (
                      <option key={d.value} value={d.value}>{d.label}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-500 mb-1.5">To</label>
                  <div className="flex flex-wrap gap-2">
                    {DAYS.filter((d) => d.value !== parseInt(copyFrom, 10)).map((d) => (
                      <label key={d.value} className={`inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm cursor-pointer ${copyTo.includes(d.value) ? "border-neutral-800 bg-neutral-100" : "border-neutral-200"}`}>
                        <input
                          type="checkbox"
                          checked={copyTo.includes(d.value)}
                          onChange={() => toggleDay(copyTo, setCopyTo, d.value)}
                        />
                        {d.label.slice(0, 3)}
                      </label>
                    ))}
                  </div>
                </div>
                <div className="space-y-2">
                  <label className="flex items-center gap-2 text-sm text-neutral-700">
                    <input
                      type="checkbox" checked={copyOverwrite}
                      onChange={(e) => setCopyOverwrite(e.target.checked)}
                    />
                    Overwrite target days first
                  </label>
                  <Button onClick={handleCopyDay} disabled={copying || copyTo.length === 0}>
                    {copying ? <><Loader2 className="h-4 w-4 mr-2 animate-spin" /> Copying&hellip;</> : "Copy day"}
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {profileId && (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Time blocks</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {loadingSlots ? (
                <div className="p-8 text-center text-sm text-neutral-400">Loading blocks&hellip;</div>
              ) : slots.length === 0 ? (
                <div className="p-8 text-center text-sm text-neutral-400">
                  No time blocks yet. Add lessons, breaks and lunch above — Saturday and Sunday are opt-in days.
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[720px]">
                    <thead>
                      <tr className="border-b border-neutral-200">
                        <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500 w-28">Day</th>
                        {allSequences.map((seq) => (
                          <th key={seq} className="px-3 py-3 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500">
                            #{seq}
                            {rowTimeLabel(seq) && (
                              <span className="block font-normal normal-case text-neutral-400">{rowTimeLabel(seq)}</span>
                            )}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-neutral-100">
                      {visibleDays.map((d) => (
                        <React.Fragment key={d.value}>
                          <tr className="hover:bg-neutral-50 align-top">
                            <td className="px-4 py-3 whitespace-nowrap">
                              <div className="text-sm font-semibold text-neutral-700">{d.label}</div>
                            </td>
                            {allSequences.map((seq) => (
                              <td key={seq} className="px-3 py-3">{renderCell(d.value, seq)}</td>
                            ))}
                          </tr>
                          {editingRowDay === d.value && editingSlot && (
                            <tr className="bg-neutral-50">
                              <td colSpan={allSequences.length + 1} className="px-6 py-4">
                                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                                  <div>
                                    <label className="block text-xs font-medium text-neutral-500 mb-1">Start</label>
                                    <input
                                      type="time" value={editStart}
                                      onChange={(e) => setEditStart(e.target.value)}
                                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                                    />
                                  </div>
                                  <div>
                                    <label className="block text-xs font-medium text-neutral-500 mb-1">End</label>
                                    <input
                                      type="time" value={editEnd}
                                      onChange={(e) => setEditEnd(e.target.value)}
                                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                                    />
                                  </div>
                                  <div>
                                    <label className="block text-xs font-medium text-neutral-500 mb-1">Type</label>
                                    <select
                                      value={editType}
                                      onChange={(e) => {
                                        setEditType(e.target.value);
                                        setEditInstructional(e.target.value === "lesson");
                                      }}
                                      className="w-full rounded-md border border-neutral-200 px-3 py-2 text-sm"
                                    >
                                      {SLOT_TYPES.map((t) => (
                                        <option key={t} value={t}>{t}</option>
                                      ))}
                                    </select>
                                  </div>
                                  <label className="flex items-center gap-2 text-sm text-neutral-700 self-end pb-2">
                                    <input
                                      type="checkbox" checked={editInstructional}
                                      onChange={(e) => setEditInstructional(e.target.checked)}
                                    />
                                    Teaching time
                                  </label>
                                </div>
                                <div className="mt-3 flex flex-wrap items-center gap-2">
                                  <label className={`inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm cursor-pointer ${editScope === "day" ? "border-neutral-800 bg-neutral-100" : "border-neutral-200"}`}>
                                    <input
                                      type="radio" name={`scope-${editingSlot.id}`}
                                      checked={editScope === "day"}
                                      onChange={() => setEditScope("day")}
                                    />
                                    {dayLabel(editingSlot.day_of_week)} only
                                  </label>
                                  <label className={`inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm cursor-pointer ${editScope === "propagate" ? "border-neutral-800 bg-neutral-100" : "border-neutral-200"}`}>
                                    <input
                                      type="radio" name={`scope-${editingSlot.id}`}
                                      checked={editScope === "propagate"}
                                      onChange={() => setEditScope("propagate")}
                                    />
                                    Also update same #{editingSlot.sequence} on:
                                  </label>
                                  {editScope === "propagate" && DAYS.filter((x) => x.value !== editingSlot.day_of_week).map((x) => (
                                    <label key={x.value} className={`inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs cursor-pointer ${editTargetDays.includes(x.value) ? "border-neutral-800 bg-neutral-100" : "border-neutral-200"}`}>
                                      <input
                                        type="checkbox"
                                        checked={editTargetDays.includes(x.value)}
                                        onChange={() => toggleDay(editTargetDays, setEditTargetDays, x.value)}
                                      />
                                      {x.label.slice(0, 3)}
                                    </label>
                                  ))}
                                </div>
                                <div className="mt-3 flex gap-2">
                                  <Button
                                    onClick={() => handleSaveEdit(editingSlot)}
                                    disabled={savingEdit || !editStart || !editEnd || (editScope === "propagate" && editTargetDays.length === 0)}
                                  >
                                    {savingEdit ? <><Loader2 className="h-4 w-4 mr-2 animate-spin" /> Saving&hellip;</> : editScope === "propagate" ? `Save to ${editTargetDays.length + 1} days` : "Save"}
                                  </Button>
                                  <button
                                    onClick={() => setEditingId(null)}
                                    className="text-xs text-neutral-500 underline self-center"
                                  >
                                    Cancel
                                  </button>
                                </div>
                              </td>
                            </tr>
                          )}
                        </React.Fragment>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </DashboardLayout>
  );
}
