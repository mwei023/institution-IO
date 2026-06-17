"use client";
import React, { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, User, DollarSign, BookOpen, Activity, FileText, Edit } from "lucide-react";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
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

export default function StudentProfilePage() {
  const params = useParams();
  const router = useRouter();
  const studentId = params.id as string;

  const [student, setStudent] = useState<StudentProfile | null>(null);
  const [timeline, setTimeline] = useState<TimelineEvent[]>([]);
  const [feeAccount, setFeeAccount] = useState<{ balance: number; total_paid: number } | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>("Overview");
  const [loading, setLoading] = useState(true);

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
        // use mock
        setStudent(mockStudent);
        setTimeline(mockTimeline);
        setFeeAccount({ balance: 0, total_paid: 19500 });
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [studentId]);

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

  return (
    <DashboardLayout title={fullName}>
      <div className="space-y-5">
        {/* Back + actions */}
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

        {/* Profile header card */}
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

        {/* Tab navigation */}
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

        {/* Tab content */}
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
              <Button variant="outline" size="sm">View Full Statement</Button>
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
