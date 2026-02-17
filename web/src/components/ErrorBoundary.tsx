import React from "react";
import { AlertTriangle } from "lucide-react";
import { Card, CardHeader, CardTitle } from "./ui/card";
import { Button } from "./ui/button";

interface Props {
  children: React.ReactNode;
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
    console.error("ErrorBoundary caught:", error, info);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex items-center justify-center py-20">
          <Card className="max-w-md text-center">
            <CardHeader className="flex-col items-center gap-3">
              <AlertTriangle className="h-10 w-10 text-rose-300" />
              <CardTitle>Something went wrong</CardTitle>
            </CardHeader>
            <p className="mb-4 text-sm text-ink-300">
              {this.state.error?.message ?? "An unexpected error occurred."}
            </p>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => window.location.reload()}
            >
              Reload page
            </Button>
          </Card>
        </div>
      );
    }

    return this.props.children;
  }
}
