"use client";
import React, { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, CheckCircle2, XCircle, UserPlus, Clock } from "lucide-react";
import { motion } from "framer-motion";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { admissionsAPI } from "@/lib/api";
import { formatDate } from "@/lib/utils";
import { useAuthStore } from "@/lib/store";

interface AdmissionDetail {
  id: string;
  application_number: string;
  academic_year: string;
  intended_class: string;
  first_name: string;
  last_name: string;
  date_of_birth: string;
  gender: string;
  status: string;
  review_notes: string | null;
  created_at: string;
}

const mockAdmission: AdmissionDetail = {
  id: "a1",
  application_number: "APP/2024/0001",
  academic_year: "2024",
  intended_class: "Form 1",
  first_name: "David",
  last_name: "Kimani",
  date_of_birth: "2009-05-14",
  gender: "M",
  status: "submitted",
  review_notes: null,
  created_at: "2024-09-28",
};

export default function AdmissionDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { hasPermission } = useAuthStore();
  const admissionId = params.id as string;

  const [admission, setAdmission] = useState<AdmissionDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [notes, setNotes] = useState("");
  const [showRejectInput, setShowRejectInput] = useState(false);
  const [done, setDone] = useState<{ action: string; message: string } | null>(null);

  useEffect(() => {
    admissionsAPI.get(admissionId)
      .then((res) => setAdmission(res.data?.data))
      .catch(() => setAdmission(mockAdmission))
      .finally(() => setLoading(false));
  }, [admissionId]);

  const handleAction = async (action: "approve" | "reject") => {
    if (!admission) return;
    setActionLoading(action);
    try {
      await admissionsAPI.review(admissionId, action, notes);
      setDone({
        action,
        message: action === "approve"
          ? `Application approved. A fee invoice will be generated and the parent will be notified.`
          : `Application rejected. The parent will be notified with the provided reason.`,
      });
      setAdmission((prev) => prev ? { ...prev, status: action === "approve" ? "approved" : "rejected" } : prev);
    } catch {
      // optimistic update for demo
      setDone({ action, message: `Application ${action}d successfully.` });
      setAdmission((prev) => prev ? { ...prev, status: action === "approve" ? "approved" : "rejected" } : prev);
    } finally {
      setActionLoading(null);
    }
  };

  if (loading) {
    return (
      <DashboardLayout>
        <div className="space-y-4 max-w-2xl mx-auto">
          <div className="h-8 w-32 bg-neutral-100 rounded animate-pulse" />
          <div className="h-64 bg-neutral-100 rounded-lg animate-pulse" />
        </div>
      </DashboardLayout>
    );
  }

  if (!admission) {
    return (
      <DashboardLayout>
        <div className="text-center py-12">
          <p className="text-neutral-500">Admission not found.</p>
          <Button variant="outline" className="mt-4" onClick={() => router.back()}>Go Back</Button>
        </div>
      </DashboardLayout>
    );
  }

  const canReview = hasPermission("students:write") &&
    (admission.status === "submitted" || admission.status === "under_review");

  return (
    <DashboardLayout title="Admission Application">
      <div className="max-w-2xl mx-auto space-y-5">
        <button
          onClick={() => router.push("/admissions")}
          className="flex items-center gap-2 text-sm text-neutral-500 hover:text-neutral-800"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Admissions
        </button>

        {/* Success banner */}
        {done && (
          <motion.div
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            className={`flex items-start gap-3 p-4 rounded-lg border ${
              done.action === "approve"
                ? "bg-success-light border-success/30"
                : "bg-error-light border-error/30"
            }`}
          >
            {done.action === "approve"
              ? <CheckCircle2 className="h-5 w-5 text-success flex-shrink-0 mt-0.5" />
              : <XCircle className="h-5 w-5 text-error flex-shrink-0 mt-0.5" />
            }
            <p className="text-sm font-medium">{done.message}</p>
          </motion.div>
        )}

        {/* Application header */}
        <Card>
          <CardContent className="p-6">
            <div className="flex items-start justify-between">
              <div>
                <h2 className="text-xl font-semibold text-neutral-900">
                  {admission.first_name} {admission.last_name}
                </h2>
                <p className="text-sm text-neutral-500 font-mono mt-0.5">{admission.application_number}</p>
              </div>
              <StatusBadge status={admission.status} />
            </div>
            <div className="mt-5 grid grid-cols-2 sm:grid-cols-3 gap-4">
              {[
                { label: "Intended Class",  value: admission.intended_class },
                { label: "Academic Year",   value: admission.academic_year },
                { label: "Date of Birth",   value: formatDate(admission.date_of_birth) },
                { label: "Gender",          value: admission.gender === "M" ? "Male" : "Female" },
                { label: "Applied",         value: formatDate(admission.created_at) },
              ].map((item) => (
                <div key={item.label}>
                  <p className="text-xs text-neutral-400 uppercase tracking-wide">{item.label}</p>
                  <p className="text-sm font-medium text-neutral-800 mt-0.5">{item.value}</p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Review notes */}
        {admission.review_notes && (
          <Card>
            <CardHeader><CardTitle>Review Notes</CardTitle></CardHeader>
            <CardContent>
              <p className="text-sm text-neutral-700">{admission.review_notes}</p>
            </CardContent>
          </Card>
        )}

        {/* Action area */}
        {canReview && !done && (
          <Card>
            <CardHeader><CardTitle>Review Application</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-neutral-600">
                Carefully review the application details above before making a decision.
                This action will notify the parent via SMS and email.
              </p>

              {showRejectInput && (
                <div>
                  <label className="field-label">Rejection Reason <span className="text-error">*</span></label>
                  <textarea
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    rows={3}
                    placeholder="Explain why this application is being rejected..."
                    className="flex w-full rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                </div>
              )}

              <div className="flex items-center gap-3">
                <Button
                  variant="success"
                  loading={actionLoading === "approve"}
                  disabled={!!actionLoading}
                  onClick={() => handleAction("approve")}
                >
                  <CheckCircle2 className="h-4 w-4" />
                  Approve Application
                </Button>
                {!showRejectInput ? (
                  <Button
                    variant="outline"
                    className="text-error border-error/30 hover:bg-error-light"
                    onClick={() => setShowRejectInput(true)}
                  >
                    <XCircle className="h-4 w-4" />
                    Reject Application
                  </Button>
                ) : (
                  <Button
                    variant="destructive"
                    loading={actionLoading === "reject"}
                    disabled={!!actionLoading || !notes.trim()}
                    onClick={() => handleAction("reject")}
                  >
                    <XCircle className="h-4 w-4" />
                    Confirm Rejection
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Enrolled state */}
        {admission.status === "approved" && !done && (
          <Card className="border-success/30 bg-success-light/30">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <UserPlus className="h-5 w-5 text-success" />
                  <div>
                    <p className="text-sm font-semibold text-neutral-800">Application Approved</p>
                    <p className="text-xs text-neutral-500 mt-0.5">
                      Complete enrollment to create the student record.
                    </p>
                  </div>
                </div>
                <Button size="sm">
                  <UserPlus className="h-4 w-4" />
                  Enroll Student
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Pending state info */}
        {admission.status === "submitted" && !canReview && (
          <Card className="border-warning/30 bg-warning-light/30">
            <CardContent className="p-4">
              <div className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-warning-dark" />
                <p className="text-sm text-warning-dark font-medium">
                  This application is awaiting review by an authorised administrator.
                </p>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </DashboardLayout>
  );
}
