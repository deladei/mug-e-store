// src/components/ui/ErrorBoundary.tsx

"use client";

import React from "react";
import { Button } from "./Button";

interface Props {
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    // In production, send to your error tracking service (Sentry, etc.)
    console.error("ErrorBoundary caught:", error, info);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;

      return (
        <div className="min-h-[50vh] flex flex-col items-center justify-center text-center px-4 space-y-4">
          <span className="text-5xl">☕</span>
          <div>
            <p className="font-bold text-stone-800 dark:text-stone-200 text-lg">
              Something went wrong
            </p>
            <p className="text-sm text-stone-500 dark:text-stone-400 mt-1">
              An unexpected error occurred. Please try refreshing the page.
            </p>
          </div>
          <Button
            variant="outline"
            onClick={() => {
              this.setState({ hasError: false, error: null });
              window.location.reload();
            }}
          >
            Refresh page
          </Button>
        </div>
      );
    }

    return this.props.children;
  }
}