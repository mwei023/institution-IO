"use client";
import React, { useState } from "react";
import { Save, School, Bell, Shield, Palette, CreditCard } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuthStore } from "@/lib/store";

const SETTING_TABS = [
  { id: "school",          label: "School Profile",   icon: School },
  { id: "notifications",   label: "Notifications",    icon: Bell },
  { id: "fees",            label: "Fee Settings",     icon: CreditCard },
  { id: "security",        label: "Security",         icon: Shield },
] as const;

type SettingTab = typeof SETTING_TABS[number]["id"];

export default function SettingsPage() {
  const { user } = useAuthStore();
  const [activeTab, setActiveTab] = useState<SettingTab>("school");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    await new Promise((r) => setTimeout(r, 800));
    setSaving(false);
  };

  return (
    <DashboardLayout title="Settings">
      <div className="space-y-5">
        <div className="page-header">
          <div>
            <h1 className="page-title">Settings</h1>
            <p className="page-subtitle">Manage school configuration and preferences</p>
          </div>
        </div>

        <div className="flex gap-6 flex-col lg:flex-row">
          {/* Sidebar */}
          <div className="lg:w-48 flex-shrink-0">
            <nav className="space-y-0.5">
              {SETTING_TABS.map(({ id, label, icon: Icon }) => (
                <button
                  key={id}
                  onClick={() => setActiveTab(id)}
                  className={`w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium text-left transition-colors ${
                    activeTab === id
                      ? "bg-primary-50 text-primary-700"
                      : "text-neutral-600 hover:bg-neutral-100"
                  }`}
                >
                  <Icon className="h-4 w-4" />
                  {label}
                </button>
              ))}
            </nav>
          </div>

          {/* Content */}
          <div className="flex-1 space-y-4">
            {activeTab === "school" && (
              <Card>
                <CardHeader><CardTitle>School Profile</CardTitle></CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center gap-4 pb-4 border-b border-neutral-100">
                    <div className="w-16 h-16 rounded-lg bg-primary-100 flex items-center justify-center">
                      <School className="h-8 w-8 text-primary-600" />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-neutral-800">School Logo</p>
                      <p className="text-xs text-neutral-500 mt-0.5">PNG or JPG, max 2MB</p>
                      <Button size="sm" variant="outline" className="mt-2">Upload Logo</Button>
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="field-label">School Name</label>
                      <Input defaultValue="Upendo Secondary School" />
                    </div>
                    <div>
                      <label className="field-label">Short Name</label>
                      <Input defaultValue="Upendo" />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="field-label">Phone</label>
                      <Input defaultValue="+254 722 100 200" />
                    </div>
                    <div>
                      <label className="field-label">Email</label>
                      <Input defaultValue="info@upendo.ac.ke" type="email" />
                    </div>
                  </div>
                  <div>
                    <label className="field-label">Address</label>
                    <Input defaultValue="Ngong Road, Karen, Nairobi" />
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="field-label">County</label>
                      <Input defaultValue="Nairobi" />
                    </div>
                    <div>
                      <label className="field-label">Registration Number</label>
                      <Input defaultValue="TSC/SEC/2005/0042" />
                    </div>
                  </div>
                  <Button loading={saving} onClick={handleSave}>
                    <Save className="h-4 w-4" />
                    Save Changes
                  </Button>
                </CardContent>
              </Card>
            )}

            {activeTab === "notifications" && (
              <Card>
                <CardHeader><CardTitle>Notification Settings</CardTitle></CardHeader>
                <CardContent className="space-y-5">
                  {[
                    { label: "SMS Provider",    sub: "Africa's Talking API key for outbound SMS", value: "" },
                    { label: "Sender ID",       sub: "SMS sender name (max 11 characters)",        value: "UpendoSec" },
                    { label: "Email From Name", sub: "Display name for outbound emails",           value: "Upendo Secondary" },
                    { label: "Email From Address", sub: "Reply-to address",                       value: "info@upendo.ac.ke" },
                  ].map((field) => (
                    <div key={field.label}>
                      <label className="field-label">{field.label}</label>
                      <Input defaultValue={field.value} placeholder={field.label} />
                      <p className="text-xs text-neutral-400 mt-1">{field.sub}</p>
                    </div>
                  ))}
                  <div className="space-y-3 pt-2">
                    <p className="text-sm font-semibold text-neutral-700">Automatic Notifications</p>
                    {[
                      "Send SMS when student is absent",
                      "Send push notification when payment received",
                      "Send SMS for fee balance reminder (weekly)",
                      "Send push notification when report card published",
                    ].map((label) => (
                      <label key={label} className="flex items-center gap-3 cursor-pointer">
                        <input type="checkbox" defaultChecked className="rounded border-neutral-300 text-primary-600" />
                        <span className="text-sm text-neutral-700">{label}</span>
                      </label>
                    ))}
                  </div>
                  <Button loading={saving} onClick={handleSave}>
                    <Save className="h-4 w-4" />
                    Save Changes
                  </Button>
                </CardContent>
              </Card>
            )}

            {activeTab === "fees" && (
              <Card>
                <CardHeader><CardTitle>Fee Settings</CardTitle></CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="field-label">M-Pesa Paybill Number</label>
                      <Input defaultValue="400200" />
                      <p className="text-xs text-neutral-400 mt-1">Your registered Safaricom Paybill</p>
                    </div>
                    <div>
                      <label className="field-label">M-Pesa Environment</label>
                      <select className="flex h-9 w-full rounded-md border border-neutral-200 bg-white px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500">
                        <option value="sandbox">Sandbox (Testing)</option>
                        <option value="production">Production (Live)</option>
                      </select>
                    </div>
                  </div>
                  <div>
                    <label className="field-label">Late Fee Policy</label>
                    <select className="flex h-9 w-full rounded-md border border-neutral-200 bg-white px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500">
                      <option>No late fees</option>
                      <option>KES 500 per month overdue</option>
                      <option>2% per month overdue</option>
                    </select>
                  </div>
                  <Button loading={saving} onClick={handleSave}>
                    <Save className="h-4 w-4" />
                    Save Changes
                  </Button>
                </CardContent>
              </Card>
            )}

            {activeTab === "security" && (
              <Card>
                <CardHeader><CardTitle>Security Settings</CardTitle></CardHeader>
                <CardContent className="space-y-5">
                  <div className="space-y-3">
                    <p className="text-sm font-semibold text-neutral-700">Access Controls</p>
                    {[
                      { label: "Enforce MFA for Admins", sub: "Require two-factor authentication for Super Admin and Department Admin roles", checked: false },
                      { label: "Enforce MFA for Teachers", sub: "Require two-factor authentication for all teacher accounts", checked: false },
                      { label: "Session timeout (idle)", sub: "Auto-logout after 30 minutes of inactivity", checked: true },
                    ].map((setting) => (
                      <div key={setting.label} className="flex items-start gap-3 p-3 rounded-lg bg-neutral-50">
                        <input type="checkbox" defaultChecked={setting.checked} className="mt-0.5 rounded border-neutral-300 text-primary-600" />
                        <div>
                          <p className="text-sm font-medium text-neutral-800">{setting.label}</p>
                          <p className="text-xs text-neutral-500 mt-0.5">{setting.sub}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                  <div className="border-t border-neutral-100 pt-4">
                    <p className="text-sm font-semibold text-neutral-700 mb-3">Data & Privacy</p>
                    <div className="space-y-2 text-sm text-neutral-600">
                      <p>Data Protection Officer: <span className="font-medium">Not assigned</span></p>
                      <p>ODPC Registration: <span className="text-warning font-medium">Pending</span></p>
                      <p>Data retention policy: <span className="font-medium">7 years (statutory)</span></p>
                    </div>
                  </div>
                  <Button loading={saving} onClick={handleSave}>
                    <Save className="h-4 w-4" />
                    Save Changes
                  </Button>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}
