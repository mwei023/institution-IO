"use client";
import React, { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Plus, Bell, MessageSquare, Send, X, Users } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { announcementsAPI } from "@/lib/api";
import { formatDate, timeAgo } from "@/lib/utils";
import { useAuthStore } from "@/lib/store";

interface Announcement {
  id: string;
  title: string;
  body: string;
  type: string;
  audience: string;
  is_published: boolean;
  published_at: string | null;
  author_name: string;
  read_count: number;
  is_read: boolean;
  created_at: string;
}

const TABS = ["Announcements", "Messages"] as const;
type Tab = typeof TABS[number];

const mockAnnouncements: Announcement[] = [
  {
    id: "1",
    title: "Term 3 2024 Fee Deadline Reminder",
    body: "Dear Parents and Guardians, kindly note that Term 3 2024 school fees are due by 13th September 2024. Students with outstanding balances risk being sent home. Please pay via M-Pesa Paybill 400200, Account Number = your child's admission number.",
    type: "fee_notice",
    audience: "parents",
    is_published: true,
    published_at: "2024-09-26T08:00:00Z",
    author_name: "Grace Atieno",
    read_count: 312,
    is_read: true,
    created_at: "2024-09-26T07:45:00Z",
  },
  {
    id: "2",
    title: "Opening of Term 3 2024",
    body: "We welcome back all students for Term 3 2024. Reporting date is Monday 2nd September 2024. Boarders should report by 4:00 PM. Please come with all required school items.",
    type: "general",
    audience: "all",
    is_published: true,
    published_at: "2024-08-30T10:00:00Z",
    author_name: "Margaret Wanjiku",
    read_count: 698,
    is_read: true,
    created_at: "2024-08-30T09:30:00Z",
  },
  {
    id: "3",
    title: "Form 4 Mock Examination Schedule",
    body: "Form 4 Mock examinations will be held from 7th to 18th October 2024. Timetables have been posted on the school notice board. Students are advised to prepare adequately.",
    type: "academic",
    audience: "students",
    is_published: true,
    published_at: "2024-09-23T09:00:00Z",
    author_name: "Peter Kamau",
    read_count: 204,
    is_read: false,
    created_at: "2024-09-23T08:30:00Z",
  },
];

const typeColors: Record<string, string> = {
  general:    "bg-neutral-100 text-neutral-600",
  academic:   "bg-primary-50 text-primary-700",
  event:      "bg-info-light text-info-dark",
  emergency:  "bg-error-light text-error-dark",
  fee_notice: "bg-warning-light text-warning-dark",
};

export default function CommunicationsPage() {
  const { user, hasPermission } = useAuthStore();
  const [activeTab, setActiveTab] = useState<Tab>("Announcements");
  const [announcements, setAnnouncements] = useState<Announcement[]>(mockAnnouncements);
  const [showNewModal, setShowNewModal] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newBody, setNewBody]   = useState("");
  const [newType, setNewType]   = useState("general");
  const [newAudience, setNewAudience] = useState("all");
  const [sending, setSending]   = useState(false);
  const [selectedAnn, setSelectedAnn] = useState<Announcement | null>(null);

  const canCreate = hasPermission("announcements:write");

  const handleCreateAnnouncement = async () => {
    if (!newTitle.trim() || !newBody.trim()) return;
    setSending(true);
    try {
      await announcementsAPI.create({
        title: newTitle, body: newBody, type: newType,
        audience: newAudience, send_push: true, publish: true,
      });
      setShowNewModal(false);
      setNewTitle(""); setNewBody("");
    } catch {
      // optimistic add for demo
      const mock: Announcement = {
        id: Date.now().toString(), title: newTitle, body: newBody,
        type: newType, audience: newAudience, is_published: true,
        published_at: new Date().toISOString(), author_name: user?.full_name ?? "You",
        read_count: 0, is_read: true, created_at: new Date().toISOString(),
      };
      setAnnouncements((prev) => [mock, ...prev]);
      setShowNewModal(false);
      setNewTitle(""); setNewBody("");
    } finally {
      setSending(false);
    }
  };

  return (
    <DashboardLayout title="Communications">
      <div className="space-y-5">
        <div className="page-header">
          <div>
            <h1 className="page-title">Communications</h1>
            <p className="page-subtitle">Announcements and direct messages</p>
          </div>
          {canCreate && (
            <Button size="sm" onClick={() => setShowNewModal(true)}>
              <Plus className="h-4 w-4" />
              New Announcement
            </Button>
          )}
        </div>

        {/* Tabs */}
        <div className="flex items-center gap-0.5 border-b border-neutral-200">
          {TABS.map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab
                  ? "border-primary-600 text-primary-700"
                  : "border-transparent text-neutral-500 hover:text-neutral-700"
              }`}
            >
              {tab === "Announcements" ? <Bell className="h-4 w-4" /> : <MessageSquare className="h-4 w-4" />}
              {tab}
            </button>
          ))}
        </div>

        {/* Announcements tab */}
        {activeTab === "Announcements" && (
          <div className="space-y-3">
            {announcements.map((ann) => (
              <Card
                key={ann.id}
                className={`cursor-pointer hover:shadow-card-hover transition-shadow ${!ann.is_read ? "border-primary-200" : ""}`}
                onClick={() => setSelectedAnn(selectedAnn?.id === ann.id ? null : ann)}
              >
                <CardContent className="p-5">
                  <div className="flex items-start gap-3">
                    {!ann.is_read && (
                      <div className="w-2 h-2 rounded-full bg-primary-500 mt-1.5 flex-shrink-0" />
                    )}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex items-center gap-2 flex-wrap">
                          <p className="text-sm font-semibold text-neutral-800">{ann.title}</p>
                          <span className={`text-xs px-2 py-0.5 rounded-full font-medium capitalize ${typeColors[ann.type] ?? "bg-neutral-100 text-neutral-600"}`}>
                            {ann.type.replace("_"," ")}
                          </span>
                        </div>
                        <p className="text-xs text-neutral-400 whitespace-nowrap flex-shrink-0">
                          {ann.published_at ? timeAgo(ann.published_at) : "Draft"}
                        </p>
                      </div>
                      <p className={`text-sm text-neutral-600 mt-1 ${selectedAnn?.id === ann.id ? "" : "line-clamp-2"}`}>
                        {ann.body}
                      </p>
                      <div className="flex items-center gap-3 mt-2">
                        <span className="text-xs text-neutral-400">By {ann.author_name}</span>
                        <span className="text-xs text-neutral-400">·</span>
                        <span className="text-xs text-neutral-400 flex items-center gap-1">
                          <Users className="h-3 w-3" />
                          {ann.read_count} reads
                        </span>
                        <span className="text-xs text-neutral-400">·</span>
                        <span className="text-xs text-neutral-400 capitalize">
                          Audience: {ann.audience}
                        </span>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}

        {/* Messages tab */}
        {activeTab === "Messages" && (
          <Card>
            <CardContent className="p-8 text-center">
              <MessageSquare className="h-10 w-10 text-neutral-300 mx-auto mb-3" />
              <p className="text-sm font-medium text-neutral-600">Direct messaging</p>
              <p className="text-xs text-neutral-400 mt-1">Coming in Phase 2 — parent ↔ teacher messaging</p>
            </CardContent>
          </Card>
        )}
      </div>

      {/* New announcement modal */}
      <AnimatePresence>
        {showNewModal && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.15 }}
              className="fixed inset-0 z-40 bg-neutral-900/50"
              onClick={() => setShowNewModal(false)}
            />
            <motion.div
              initial={{ opacity: 0, scale: 0.97 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.15 }}
              className="fixed inset-0 z-50 flex items-center justify-center p-4"
            >
              <div className="bg-white rounded-xl shadow-modal w-full max-w-lg">
                <div className="flex items-center justify-between px-6 py-4 border-b border-neutral-100">
                  <h3 className="text-base font-semibold text-neutral-900">New Announcement</h3>
                  <button
                    onClick={() => setShowNewModal(false)}
                    className="p-1.5 rounded-md text-neutral-400 hover:bg-neutral-100"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
                <div className="p-6 space-y-4">
                  <div>
                    <label className="field-label">Title <span className="text-error">*</span></label>
                    <Input
                      value={newTitle}
                      onChange={(e) => setNewTitle(e.target.value)}
                      placeholder="Brief, clear subject line"
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="field-label">Type</label>
                      <select
                        value={newType}
                        onChange={(e) => setNewType(e.target.value)}
                        className="flex h-9 w-full rounded-md border border-neutral-200 bg-white px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                      >
                        {["general","academic","event","fee_notice","emergency"].map((t) => (
                          <option key={t} value={t} className="capitalize">{t.replace("_"," ")}</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="field-label">Audience</label>
                      <select
                        value={newAudience}
                        onChange={(e) => setNewAudience(e.target.value)}
                        className="flex h-9 w-full rounded-md border border-neutral-200 bg-white px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                      >
                        {["all","parents","teachers","students","staff"].map((a) => (
                          <option key={a} value={a} className="capitalize">{a}</option>
                        ))}
                      </select>
                    </div>
                  </div>
                  <div>
                    <label className="field-label">Message <span className="text-error">*</span></label>
                    <textarea
                      value={newBody}
                      onChange={(e) => setNewBody(e.target.value)}
                      rows={4}
                      placeholder="Write your announcement..."
                      className="flex w-full rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                  </div>
                </div>
                <div className="flex justify-end gap-3 px-6 py-4 border-t border-neutral-100">
                  <Button variant="outline" onClick={() => setShowNewModal(false)}>Cancel</Button>
                  <Button
                    loading={sending}
                    onClick={handleCreateAnnouncement}
                    disabled={!newTitle.trim() || !newBody.trim()}
                  >
                    <Send className="h-4 w-4" />
                    Publish Now
                  </Button>
                </div>
              </div>
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </DashboardLayout>
  );
}
