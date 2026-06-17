"use client";
import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { ArrowLeft, CheckCircle2 } from "lucide-react";
import { motion } from "framer-motion";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { admissionsAPI } from "@/lib/api";

const schema = z.object({
  academic_year:          z.string().min(4, "Required"),
  intended_class:         z.string().min(1, "Required"),
  first_name:             z.string().min(1, "Required").max(100),
  last_name:              z.string().min(1, "Required").max(100),
  other_names:            z.string().optional(),
  date_of_birth:          z.string().min(1, "Required"),
  gender:                 z.enum(["M", "F", "O"], { required_error: "Required" }),
  birth_certificate_number: z.string().optional(),
  nationality:            z.string().optional(),
  previous_school:        z.string().optional(),
  blood_group:            z.string().optional(),
  medical_conditions:     z.string().optional(),
  allergies:              z.string().optional(),
  special_needs:          z.boolean().default(false),
  boarding_required:      z.boolean().default(false),
  transport_required:     z.boolean().default(false),
  parent_name:            z.string().min(1, "Required"),
  parent_relationship:    z.string().min(1, "Required"),
  parent_phone:           z.string().min(10, "Valid phone required"),
  parent_email:           z.string().email("Valid email required").optional().or(z.literal("")),
});

type FormData = z.infer<typeof schema>;

const STEPS = ["Student Details", "Medical & Boarding", "Parent/Guardian"] as const;

export default function NewAdmissionPage() {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [submitted, setSubmitted] = useState(false);
  const [createdRef, setCreatedRef] = useState("");

  const {
    register,
    handleSubmit,
    trigger,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      academic_year: new Date().getFullYear().toString(),
      nationality: "Kenyan",
      special_needs: false,
      boarding_required: false,
      transport_required: false,
    },
  });

  const nextStep = async () => {
    const fieldsPerStep: Array<Array<keyof FormData>> = [
      ["first_name", "last_name", "date_of_birth", "gender", "intended_class", "academic_year"],
      [],
      ["parent_name", "parent_relationship", "parent_phone"],
    ];
    const valid = await trigger(fieldsPerStep[step]);
    if (valid) setStep((s) => s + 1);
  };

  const onSubmit = async (data: FormData) => {
    try {
      const res = await admissionsAPI.create(data);
      setCreatedRef(res.data?.data?.application_number ?? "APP/NEW");
      setSubmitted(true);
    } catch (err: unknown) {
      console.error(err);
    }
  };

  if (submitted) {
    return (
      <DashboardLayout title="Admission Submitted">
        <motion.div
          initial={{ opacity: 0, scale: 0.97 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.18 }}
          className="max-w-lg mx-auto text-center py-16"
        >
          <div className="w-16 h-16 rounded-full bg-success-light flex items-center justify-center mx-auto mb-5">
            <CheckCircle2 className="h-8 w-8 text-success" />
          </div>
          <h2 className="text-xl font-semibold text-neutral-900">Application Submitted</h2>
          <p className="text-neutral-500 mt-2">
            Reference number: <span className="font-mono font-semibold text-neutral-800">{createdRef}</span>
          </p>
          <p className="text-sm text-neutral-400 mt-1">
            The application is now pending review by the Principal.
          </p>
          <div className="flex justify-center gap-3 mt-8">
            <Button variant="outline" onClick={() => router.push("/admissions")}>
              View All Admissions
            </Button>
            <Button onClick={() => { setSubmitted(false); setStep(0); }}>
              New Application
            </Button>
          </div>
        </motion.div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout title="New Admission">
      <div className="max-w-2xl mx-auto space-y-5">
        {/* Back */}
        <button
          onClick={() => router.back()}
          className="flex items-center gap-2 text-sm text-neutral-500 hover:text-neutral-800"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Admissions
        </button>

        {/* Step indicator */}
        <div className="flex items-center gap-0">
          {STEPS.map((label, i) => (
            <React.Fragment key={label}>
              <div className="flex items-center gap-2">
                <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold
                  ${i < step ? "bg-success text-white" :
                    i === step ? "bg-primary-600 text-white" :
                    "bg-neutral-100 text-neutral-500"}`}
                >
                  {i < step ? "✓" : i + 1}
                </div>
                <span className={`text-sm font-medium hidden sm:block ${i === step ? "text-neutral-800" : "text-neutral-400"}`}>
                  {label}
                </span>
              </div>
              {i < STEPS.length - 1 && (
                <div className={`flex-1 h-0.5 mx-3 ${i < step ? "bg-success" : "bg-neutral-200"}`} />
              )}
            </React.Fragment>
          ))}
        </div>

        <form onSubmit={handleSubmit(onSubmit)}>
          {/* Step 1 — Student Details */}
          {step === 0 && (
            <Card>
              <CardHeader><CardTitle>Student Details</CardTitle></CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="field-label">First Name <span className="text-error">*</span></label>
                    <Input {...register("first_name")} placeholder="e.g. Amara" error={!!errors.first_name} />
                    {errors.first_name && <p className="field-error">{errors.first_name.message}</p>}
                  </div>
                  <div>
                    <label className="field-label">Last Name <span className="text-error">*</span></label>
                    <Input {...register("last_name")} placeholder="e.g. Odhiambo" error={!!errors.last_name} />
                    {errors.last_name && <p className="field-error">{errors.last_name.message}</p>}
                  </div>
                </div>
                <div>
                  <label className="field-label">Other Names</label>
                  <Input {...register("other_names")} placeholder="Middle name (optional)" />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="field-label">Date of Birth <span className="text-error">*</span></label>
                    <Input {...register("date_of_birth")} type="date" error={!!errors.date_of_birth} />
                    {errors.date_of_birth && <p className="field-error">{errors.date_of_birth.message}</p>}
                  </div>
                  <div>
                    <label className="field-label">Gender <span className="text-error">*</span></label>
                    <select
                      {...register("gender")}
                      className="flex h-9 w-full rounded-md border border-neutral-200 bg-white px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                    >
                      <option value="">Select gender</option>
                      <option value="M">Male</option>
                      <option value="F">Female</option>
                    </select>
                    {errors.gender && <p className="field-error">{errors.gender.message}</p>}
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="field-label">Intended Class <span className="text-error">*</span></label>
                    <select
                      {...register("intended_class")}
                      className="flex h-9 w-full rounded-md border border-neutral-200 bg-white px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                    >
                      <option value="">Select class</option>
                      {["Form 1","Form 2","Form 3","Form 4"].map((c) => (
                        <option key={c} value={c}>{c}</option>
                      ))}
                    </select>
                    {errors.intended_class && <p className="field-error">{errors.intended_class.message}</p>}
                  </div>
                  <div>
                    <label className="field-label">Academic Year <span className="text-error">*</span></label>
                    <Input {...register("academic_year")} placeholder="2024" />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="field-label">Birth Certificate No.</label>
                    <Input {...register("birth_certificate_number")} placeholder="Optional" />
                  </div>
                  <div>
                    <label className="field-label">Nationality</label>
                    <Input {...register("nationality")} placeholder="Kenyan" />
                  </div>
                </div>
                <div>
                  <label className="field-label">Previous School</label>
                  <Input {...register("previous_school")} placeholder="Name of last school attended" />
                </div>
              </CardContent>
            </Card>
          )}

          {/* Step 2 — Medical & Boarding */}
          {step === 1 && (
            <Card>
              <CardHeader><CardTitle>Medical & Boarding</CardTitle></CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="field-label">Blood Group</label>
                  <select
                    {...register("blood_group")}
                    className="flex h-9 w-full rounded-md border border-neutral-200 bg-white px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                  >
                    <option value="">Unknown / Not tested</option>
                    {["A+","A-","B+","B-","O+","O-","AB+","AB-"].map((bg) => (
                      <option key={bg} value={bg}>{bg}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="field-label">Known Medical Conditions</label>
                  <textarea
                    {...register("medical_conditions")}
                    rows={2}
                    placeholder="e.g. Asthma, Diabetes, Epilepsy — or leave blank if none"
                    className="flex w-full rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                </div>
                <div>
                  <label className="field-label">Known Allergies</label>
                  <textarea
                    {...register("allergies")}
                    rows={2}
                    placeholder="e.g. Penicillin, Nuts — or leave blank if none"
                    className="flex w-full rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                </div>
                <div className="space-y-3 pt-2">
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input type="checkbox" {...register("special_needs")} className="rounded border-neutral-300" />
                    <span className="text-sm text-neutral-700">Student has special educational needs</span>
                  </label>
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input type="checkbox" {...register("boarding_required")} className="rounded border-neutral-300" />
                    <span className="text-sm text-neutral-700">Boarding required (will live at school)</span>
                  </label>
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input type="checkbox" {...register("transport_required")} className="rounded border-neutral-300" />
                    <span className="text-sm text-neutral-700">School bus transport required</span>
                  </label>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Step 3 — Parent/Guardian */}
          {step === 2 && (
            <Card>
              <CardHeader><CardTitle>Parent / Guardian Details</CardTitle></CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="field-label">Full Name <span className="text-error">*</span></label>
                    <Input {...register("parent_name")} placeholder="Parent or guardian name" error={!!errors.parent_name} />
                    {errors.parent_name && <p className="field-error">{errors.parent_name.message}</p>}
                  </div>
                  <div>
                    <label className="field-label">Relationship <span className="text-error">*</span></label>
                    <select
                      {...register("parent_relationship")}
                      className="flex h-9 w-full rounded-md border border-neutral-200 bg-white px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                    >
                      <option value="">Select relationship</option>
                      {["father","mother","guardian","grandparent","uncle","aunt","sibling","other"].map((r) => (
                        <option key={r} value={r} className="capitalize">{r}</option>
                      ))}
                    </select>
                    {errors.parent_relationship && <p className="field-error">{errors.parent_relationship.message}</p>}
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="field-label">Phone Number <span className="text-error">*</span></label>
                    <Input {...register("parent_phone")} placeholder="+254712345678" error={!!errors.parent_phone} />
                    {errors.parent_phone && <p className="field-error">{errors.parent_phone.message}</p>}
                  </div>
                  <div>
                    <label className="field-label">Email Address</label>
                    <Input {...register("parent_email")} type="email" placeholder="Optional" error={!!errors.parent_email} />
                    {errors.parent_email && <p className="field-error">{errors.parent_email.message}</p>}
                  </div>
                </div>
                <div className="rounded-md bg-neutral-50 border border-neutral-200 p-3 text-sm text-neutral-500">
                  A parent portal account will be created automatically when the student is enrolled. Login details will be sent via SMS.
                </div>
              </CardContent>
            </Card>
          )}

          {/* Navigation */}
          <div className="flex justify-between mt-5">
            <Button
              type="button"
              variant="outline"
              onClick={() => step === 0 ? router.back() : setStep((s) => s - 1)}
            >
              {step === 0 ? "Cancel" : "Back"}
            </Button>
            {step < STEPS.length - 1 ? (
              <Button type="button" onClick={nextStep}>
                Next: {STEPS[step + 1]}
              </Button>
            ) : (
              <Button type="submit" loading={isSubmitting}>
                Submit Application
              </Button>
            )}
          </div>
        </form>
      </div>
    </DashboardLayout>
  );
}
