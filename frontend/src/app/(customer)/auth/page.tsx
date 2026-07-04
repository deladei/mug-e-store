// src/app/(customer)/auth/page.tsx

"use client";

import { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { motion, AnimatePresence } from "framer-motion";
import { Eye, EyeOff, ArrowLeft } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { toast } from "@/hooks/useToast";
import { authService } from "@/services/auth.service";
import { API_ERROR_CODES } from "@/types";
import {
  loginSchema,
  registerSchema,
  passwordResetRequestSchema,
  passwordResetConfirmSchema,
  LoginFormValues,
  RegisterFormValues,
  PasswordResetRequestValues,
  PasswordResetConfirmValues,
} from "@/lib/validators/auth.schema";
import { ROUTES } from "@/constants/routes";
import { cn } from "@/utils";
import axios from "axios";

// ── Tab type ──────────────────────────────────────────────────────────────────

type AuthTab = "login" | "register" | "reset-request" | "reset-confirm";

// ── Password field with show/hide toggle ──────────────────────────────────────
function PasswordInput({
  label,
  error,
  ...props
}: React.ComponentProps<typeof Input>) {
  const [show, setShow] = useState(false);
  return (
    <Input
      {...props}
      label={label}
      error={error}
      type={show ? "text" : "password"}
      rightIcon={
        <button
          type="button"
          onClick={() => setShow((s) => !s)}
          className="text-stone-400 hover:text-stone-600 dark:hover:text-stone-300 cursor-pointer"
        >
          {show ? <EyeOff size={16} /> : <Eye size={16} />}
        </button>
      }
    />
  );
}

// ── Login form ─────────────────────────────────────────────────────────────────

function LoginForm({ onSuccess }: { onSuccess: () => void }) {
  const { login } = useAuth();
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({ resolver: zodResolver(loginSchema) });

  const onSubmit = async (values: LoginFormValues) => {
    try {
      await login(values);
      toast.success("Welcome back!");
      onSuccess();
    } catch (err) {
      if (axios.isAxiosError(err)) {
        const code = err.response?.data?.error?.code;
        if (code === API_ERROR_CODES.INVALID_CREDENTIALS) {
          setError("email", { message: "Invalid email or password" });
          setError("password", { message: "Invalid email or password" });
        } else if (code === API_ERROR_CODES.RATE_LIMITED) {
          toast.error("Too many attempts — wait a moment and try again");
        } else {
          toast.error("Something went wrong — please try again");
        }
      }
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <Input
        label="Email"
        type="email"
        placeholder="kofi@example.com"
        error={errors.email?.message}
        {...register("email")}
      />
      <PasswordInput
        label="Password"
        placeholder="••••••••"
        error={errors.password?.message}
        {...register("password")}
      />
      <Button type="submit" fullWidth size="lg" isLoading={isSubmitting}>
        Sign in
      </Button>
    </form>
  );
}

// ── Register form ──────────────────────────────────────────────────────────────

function RegisterForm({ onSuccess }: { onSuccess: () => void }) {
  const { register: registerUser } = useAuth();
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormValues>({ resolver: zodResolver(registerSchema) });

  const onSubmit = async (values: RegisterFormValues) => {
    try {
      await registerUser(values);
      toast.success("Account created — welcome!");
      onSuccess();
    } catch (err) {
      if (axios.isAxiosError(err)) {
        const code = err.response?.data?.error?.code;
        if (code === API_ERROR_CODES.EMAIL_TAKEN) {
          setError("email", {
            message: "An account with this email already exists",
          });
        } else if (code === API_ERROR_CODES.VALIDATION) {
          toast.error("Please check your details and try again");
        } else if (code === API_ERROR_CODES.RATE_LIMITED) {
          toast.error("Too many attempts — wait a moment and try again");
        } else {
          toast.error("Something went wrong — please try again");
        }
      }
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <Input
        label="Full name"
        placeholder="Kofi Mensah"
        error={errors.name?.message}
        {...register("name")}
      />
      <Input
        label="Email"
        type="email"
        placeholder="kofi@example.com"
        error={errors.email?.message}
        {...register("email")}
      />
      <Input
        label="Phone"
        type="tel"
        placeholder="0244000001"
        error={errors.phone?.message}
        {...register("phone")}
      />
      <PasswordInput
        label="Password"
        placeholder="Min. 8 characters"
        error={errors.password?.message}
        {...register("password")}
      />
      <Button type="submit" fullWidth size="lg" isLoading={isSubmitting}>
        Create account
      </Button>
    </form>
  );
}

// ── Password reset request form ────────────────────────────────────────────────

function ResetRequestForm({ onSent }: { onSent: () => void }) {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<PasswordResetRequestValues>({
    resolver: zodResolver(passwordResetRequestSchema),
  });

  const onSubmit = async (values: PasswordResetRequestValues) => {
    try {
      await authService.requestPasswordReset(values);
      onSent();
    } catch {
      // Always show the same message — no account enumeration
      onSent();
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <Input
        label="Email"
        type="email"
        placeholder="kofi@example.com"
        error={errors.email?.message}
        hint="We'll send a reset link if an account exists"
        {...register("email")}
      />
      <Button type="submit" fullWidth size="lg" isLoading={isSubmitting}>
        Send reset link
      </Button>
    </form>
  );
}

// ── Password reset confirm form ────────────────────────────────────────────────

function ResetConfirmForm({
  token,
  onSuccess,
}: {
  token: string;
  onSuccess: () => void;
}) {
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<PasswordResetConfirmValues>({
    resolver: zodResolver(passwordResetConfirmSchema),
  });

  const onSubmit = async (values: PasswordResetConfirmValues) => {
    try {
      await authService.confirmPasswordReset({ token, password: values.password });
      toast.success("Password reset — please sign in with your new password");
      onSuccess();
    } catch (err) {
      if (axios.isAxiosError(err)) {
        const code = err.response?.data?.error?.code;
        if (code === API_ERROR_CODES.INVALID_TOKEN) {
          setError("password", {
            message: "This reset link is invalid or has expired — request a new one",
          });
        } else {
          toast.error("Something went wrong — please try again");
        }
      }
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <PasswordInput
        label="New password"
        placeholder="Min. 8 characters"
        error={errors.password?.message}
        {...register("password")}
      />
      <Button type="submit" fullWidth size="lg" isLoading={isSubmitting}>
        Reset password
      </Button>
    </form>
  );
}

// ── Internal Auth Component (Uses Search Params) ───────────────────────────────

function AuthContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { isAuthenticated, isLoading } = useAuth();

  const resetToken = searchParams.get("token");
  const [tab, setTab] = useState<AuthTab>("login");
  const [resetSent, setResetSent] = useState(false);

  // Safely intercept and swap tab rules once parameters load
  useEffect(() => {
    if (resetToken) {
      setTab("reset-confirm");
    }
  }, [resetToken]);

  // Redirect already-authenticated users away from this page
  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.replace(ROUTES.HOME);
    }
  }, [isAuthenticated, isLoading, router]);

  const handleAuthSuccess = () => {
    const referrer = document.referrer;
    if (referrer && referrer.includes(window.location.host)) {
      router.back();
    } else {
      router.push(ROUTES.HOME);
    }
  };

  const tabConfig: Record<
    "login" | "register",
    { label: string; key: "login" | "register" }
  > = {
    login: { label: "Sign in", key: "login" },
    register: { label: "Create account", key: "register" },
  };

  if (resetSent) {
    return (
      <div className="min-h-[70vh] flex items-center justify-center px-4">
        <div className="w-full max-w-sm text-center space-y-4">
          <div className="text-5xl">📬</div>
          <h1 className="text-xl font-bold text-stone-900 dark:text-stone-100">
            Check your inbox
          </h1>
          <p className="text-sm text-stone-500 dark:text-stone-400">
            If that address has an account, we've sent a password reset link.
            Check your email and follow the instructions.
          </p>
          <Button
            variant="ghost"
            fullWidth
            onClick={() => {
              setResetSent(false);
              setTab("login");
            }}
          >
            Back to sign in
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-[70vh] flex items-center justify-center px-4">
      <motion.div
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="w-full max-w-sm space-y-6"
      >
        {/* Header */}
        <div className="text-center space-y-1">
          <span className="text-4xl">☕</span>
          <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
            {tab === "reset-request" && "Reset password"}
            {tab === "reset-confirm" && "New password"}
            {(tab === "login" || tab === "register") && "Coffee Mug Shop"}
          </h1>
          <p className="text-sm text-stone-500 dark:text-stone-400">
            {tab === "reset-request" &&
              "Enter your email and we'll send a reset link"}
            {tab === "reset-confirm" && "Choose a new password"}
            {tab === "login" && "Sign in to place your order"}
            {tab === "register" && "Create an account to get started"}
          </p>
        </div>

        {/* Login / Register tabs */}
        {(tab === "login" || tab === "register") && (
          <div className="flex bg-stone-100 dark:bg-stone-800 rounded-xl p-1">
            {(["login", "register"] as const).map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={cn(
                  "flex-1 h-8 rounded-lg text-sm font-medium transition-all duration-150 cursor-pointer",
                  tab === t
                    ? "bg-white dark:bg-stone-900 text-stone-900 dark:text-stone-100 shadow-sm"
                    : "text-stone-500 dark:text-stone-400 hover:text-stone-700 dark:hover:text-stone-300"
                )}
              >
                {tabConfig[t].label}
              </button>
            ))}
          </div>
        )}

        {/* Back button for reset flows */}
        {(tab === "reset-request" || tab === "reset-confirm") && (
          <button
            onClick={() => setTab("login")}
            className="flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 transition-colors cursor-pointer"
          >
            <ArrowLeft size={15} />
            Back to sign in
          </button>
        )}

        {/* Form area */}
        <AnimatePresence mode="wait">
          <motion.div
            key={tab}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.15 }}
          >
            {tab === "login" && (
              <LoginForm onSuccess={handleAuthSuccess} />
            )}
            {tab === "register" && (
              <RegisterForm onSuccess={handleAuthSuccess} />
            )}
            {tab === "reset-request" && (
              <ResetRequestForm onSent={() => setResetSent(true)} />
            )}
            {tab === "reset-confirm" && resetToken && (
              <ResetConfirmForm
                token={resetToken}
                onSuccess={() => setTab("login")}
              />
            )}
          </motion.div>
        </AnimatePresence>

        {/* Forgot password link */}
        {tab === "login" && (
          <p className="text-center text-sm text-stone-500 dark:text-stone-400">
            <button
              onClick={() => setTab("reset-request")}
              className="text-amber-700 dark:text-amber-500 hover:underline font-medium cursor-pointer"
            >
              Forgot password?
            </button>
          </p>
        )}
      </motion.div>
    </div>
  );
}

// ── Main Layout Entry Point (Wrapped in Suspense) ──────────────────────────────

export default function AuthPage() {
  return (
    <Suspense 
      fallback={
        <div className="min-h-[70vh] flex items-center justify-center">
          <div className="w-8 h-8 border-4 border-amber-700 border-t-transparent rounded-full animate-spin" />
        </div>
      }
    >
      <AuthContent />
    </Suspense>
  );
}