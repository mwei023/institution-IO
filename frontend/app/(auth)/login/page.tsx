"use client";
import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { GraduationCap, Eye, EyeOff, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { authAPI } from "@/lib/api";
import { useAuthStore } from "@/lib/store";
import { roleDashboardPath } from "@/lib/utils";

const loginSchema = z.object({
  institutionSlug: z.string().min(1, "Institution is required"),
  email: z.string().min(1, "Email or phone is required"),
  password: z.string().min(8, "Password must be at least 8 characters"),
});

type LoginForm = z.infer<typeof loginSchema>;

const cardVariants = {
  initial: { opacity: 0, scale: 0.97 },
  animate: { opacity: 1, scale: 1, transition: { duration: 0.18, ease: "easeOut" } },
};

export default function LoginPage() {
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { setAuth } = useAuthStore();
  const router = useRouter();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: { institutionSlug: "upendo" },
  });

  const onSubmit = async (data: LoginForm) => {
    setError(null);
    try {
      const response = await authAPI.login(data.email, data.password);
      const result = response.data?.data;

      if (result?.mfa_required) {
        // Store MFA token temporarily and redirect to MFA page
        sessionStorage.setItem("mfa_token", result.mfa_token);
        router.push("/mfa");
        return;
      }

      if (result?.access_token && result?.user) {
        setAuth(result.user, result.access_token, data.institutionSlug);
        const dashboardPath = roleDashboardPath[result.user.role] ?? "/principal";
        router.push(dashboardPath);
      } else {
        setError("Unexpected response from server. Please try again.");
      }
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { code?: string; message?: string } } } };
      const code = axiosErr?.response?.data?.error?.code;
      switch (code) {
        case "INVALID_CREDENTIALS":
          setError("Incorrect email or password. Please try again.");
          break;
        case "ACCOUNT_LOCKED":
          setError("Your account is temporarily locked after too many failed attempts. Try again in 30 minutes.");
          break;
        case "ACCOUNT_INACTIVE":
          setError("Your account has been deactivated. Please contact your administrator.");
          break;
        case "INVALID_INSTITUTION":
          setError("Institution not found. Please check the institution code.");
          break;
        default:
          setError("Something went wrong. Please try again or contact support.");
      }
    }
  };

  return (
    <motion.div
      variants={cardVariants}
      initial="initial"
      animate="animate"
      className="w-full max-w-md"
    >
      <div className="bg-white rounded-xl shadow-modal border border-neutral-100 overflow-hidden">
        {/* Header */}
        <div className="bg-primary-600 px-8 py-8 text-center">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-white/20 mb-4">
            <GraduationCap className="h-7 w-7 text-white" />
          </div>
          <h1 className="text-xl font-semibold text-white">Institution OS</h1>
          <p className="text-primary-200 text-sm mt-1">School Management Platform</p>
        </div>

        {/* Form */}
        <div className="px-8 py-8">
          <h2 className="text-lg font-semibold text-neutral-800 mb-1">Sign in to your account</h2>
          <p className="text-sm text-neutral-500 mb-6">Enter your credentials to continue</p>

          {error && (
            <motion.div
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              className="flex items-start gap-3 p-3 rounded-md bg-error-light border border-error/20 mb-5"
            >
              <AlertCircle className="h-4 w-4 text-error mt-0.5 flex-shrink-0" />
              <p className="text-sm text-error-dark">{error}</p>
            </motion.div>
          )}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Institution */}
            <div>
              <label className="field-label">
                Institution Code <span className="text-error">*</span>
              </label>
              <Input
                {...register("institutionSlug")}
                placeholder="e.g. upendo"
                error={!!errors.institutionSlug}
              />
              {errors.institutionSlug && (
                <p className="field-error">{errors.institutionSlug.message}</p>
              )}
              <p className="text-xs text-neutral-400 mt-1">
                Your school&apos;s unique identifier
              </p>
            </div>

            {/* Email/Phone */}
            <div>
              <label className="field-label">
                Email or Phone <span className="text-error">*</span>
              </label>
              <Input
                {...register("email")}
                type="text"
                placeholder="teacher@school.ac.ke or 0712345678"
                autoComplete="username"
                error={!!errors.email}
              />
              {errors.email && <p className="field-error">{errors.email.message}</p>}
            </div>

            {/* Password */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="field-label mb-0">
                  Password <span className="text-error">*</span>
                </label>
                <button
                  type="button"
                  className="text-xs text-primary-600 hover:text-primary-700 font-medium"
                  onClick={() => {/* TODO: forgot password */}}
                >
                  Forgot password?
                </button>
              </div>
              <div className="relative">
                <Input
                  {...register("password")}
                  type={showPassword ? "text" : "password"}
                  placeholder="Enter your password"
                  autoComplete="current-password"
                  error={!!errors.password}
                  className="pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.password && <p className="field-error">{errors.password.message}</p>}
            </div>

            <Button type="submit" className="w-full mt-2" size="lg" loading={isSubmitting}>
              Sign In
            </Button>
          </form>
        </div>

        {/* Demo credentials */}
        <div className="px-8 pb-6">
          <div className="rounded-md bg-neutral-50 border border-neutral-200 p-3">
            <p className="text-xs font-medium text-neutral-600 mb-2">Demo credentials</p>
            <div className="space-y-1 text-xs text-neutral-500">
              <p><span className="font-medium">Principal:</span> principal@upendo.ac.ke</p>
              <p><span className="font-medium">Teacher:</span> teacher.odhiambo@upendo.ac.ke</p>
              <p><span className="font-medium">Parent:</span> parent.amara@gmail.com</p>
              <p><span className="font-medium">Password:</span> Admin1234!</p>
            </div>
          </div>
        </div>
      </div>

      <p className="text-center text-xs text-neutral-400 mt-4">
        Institution OS · Secure School Management
      </p>
    </motion.div>
  );
}
