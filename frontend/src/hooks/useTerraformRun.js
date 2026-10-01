import { useEffect, useRef, useState } from "react";
import { TerraformRunsApi } from "../lib/endpoints";

const POLL_INTERVAL_MS = 2500;
const TERMINAL_STATUSES = new Set(["succeeded", "failed"]);

export function useTerraformRun(projectId, runId) {
  const [run, setRun] = useState(null);
  const [error, setError] = useState(null);
  const intervalRef = useRef(null);

  useEffect(() => {
    setRun(null);
    setError(null);

    if (!projectId || !runId) {
      return undefined;
    }

    let cancelled = false;

    async function poll() {
      try {
        const latest = await TerraformRunsApi.get(projectId, runId);
        if (cancelled) return;

        setRun(latest);

        if (TERMINAL_STATUSES.has(latest.status) && intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      } catch (caughtError) {
        if (!cancelled) {
          setError(caughtError);
          if (intervalRef.current) {
            clearInterval(intervalRef.current);
            intervalRef.current = null;
          }
        }
      }
    }

    poll();
    intervalRef.current = setInterval(poll, POLL_INTERVAL_MS);

    return () => {
      cancelled = true;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [projectId, runId]);

  const isRunning = run && !TERMINAL_STATUSES.has(run.status);

  return { run, error, isRunning };
}
