import { Fragment, useState } from "react";
import { History } from "lucide-react";
import { useHistory } from "../lib/queries";
import type { HistoryItem } from "../lib/types";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Skeleton } from "../components/ui/skeleton";
import { EmptyState } from "../components/EmptyState";
import { TimeAgo } from "../components/TimeAgo";

function formatDate(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

function resultBadge(item: HistoryItem) {
  if (item.rolled_back) return <Badge variant="warning">Rolled back</Badge>;
  if (item.success) return <Badge variant="success">Success</Badge>;
  return <Badge variant="danger">Failed</Badge>;
}

export function HistoryPage() {
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({ target_id: "", service_id: "", result: "" });
  const { data, isLoading } = useHistory(page, 20, filters);
  const [expanded, setExpanded] = useState<string | null>(null);

  const updateFilter = (key: keyof typeof filters, value: string) => {
    setPage(1);
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>History</CardTitle>
          <CardDescription>Audit update outcomes and rollback decisions.</CardDescription>
        </CardHeader>
        <div className="grid gap-3 md:grid-cols-3">
          <Input
            placeholder="Filter by target id"
            value={filters.target_id}
            onChange={(event) => updateFilter("target_id", event.target.value)}
          />
          <Input
            placeholder="Filter by service id"
            value={filters.service_id}
            onChange={(event) => updateFilter("service_id", event.target.value)}
          />
          <Input
            placeholder="Result: success | failed | rolled_back"
            value={filters.result}
            onChange={(event) => updateFilter("result", event.target.value)}
          />
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Runs</CardTitle>
          <CardDescription>Most recent 20 entries.</CardDescription>
        </CardHeader>
        {isLoading && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Service</TableHead>
                <TableHead>Result</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Completed</TableHead>
                <TableHead></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                  <TableCell><Skeleton className="h-6 w-20 rounded-full" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-12" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                  <TableCell><Skeleton className="h-6 w-16" /></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        {!isLoading && (data?.items ?? []).length === 0 && (
          <EmptyState
            icon={History}
            title="No update history"
            description="History appears after your first apply run."
          />
        )}
        {!isLoading && (data?.items ?? []).length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Service</TableHead>
                <TableHead>Result</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Completed</TableHead>
                <TableHead></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(data?.items ?? []).map((item) => (
                <Fragment key={item.service_id}>
                  <TableRow>
                    <TableCell>
                      <div className="font-semibold text-ink-100">{item.service_name}</div>
                      <div className="text-xs text-ink-400">{item.target_id}</div>
                    </TableCell>
                    <TableCell>{resultBadge(item)}</TableCell>
                    <TableCell>{item.duration_sec.toFixed(1)}s</TableCell>
                    <TableCell>
                      {item.completed_at ? (
                        <TimeAgo date={item.completed_at} />
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setExpanded(expanded === item.service_id ? null : item.service_id)}
                      >
                        Details
                      </Button>
                    </TableCell>
                  </TableRow>
                  {expanded === item.service_id && (
                    <TableRow className="bg-ink-900/80">
                      <TableCell colSpan={5}>
                        <div className="grid gap-2 text-xs text-ink-300 md:grid-cols-2">
                          <div>Old digest: {item.old_digest}</div>
                          <div>New digest: {item.new_digest}</div>
                          <div>Probes passed: {item.probes_passed}</div>
                          <div>Probes failed: {item.probes_failed}</div>
                          <div>Started: {formatDate(item.started_at)}</div>
                          <div>Error: {item.error_message || "-"}</div>
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </Fragment>
              ))}
            </TableBody>
          </Table>
        )}
        <div className="mt-4 flex items-center justify-between">
          <Button variant="secondary" size="sm" disabled={page === 1} onClick={() => setPage(page - 1)}>
            Previous
          </Button>
          <div className="text-xs text-ink-400">Page {page}</div>
          <Button
            variant="secondary"
            size="sm"
            disabled={!data?.has_more}
            onClick={() => setPage(page + 1)}
          >
            Next
          </Button>
        </div>
      </Card>
    </div>
  );
}
