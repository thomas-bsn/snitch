import { useEffect, useState } from "react";
import { useLocation } from "react-router";
import { fetchToProcess } from "../people/peopleApi";

/**
 * The number of voices to decide (unknown + ambiguous), from the "To process" API, for the
 * badge of the People entry. Read again when the address changes; a failed read hides it.
 */
export function useToProcessCount(): number | undefined {
  const { pathname, search } = useLocation();
  const [count, setCount] = useState<number>();
  useEffect(() => {
    let current = true;
    void fetchToProcess("all", 1, 1).then((result) => {
      if (!current) return;
      setCount(
        result.ok
          ? result.value.totals.unknown + result.value.totals.ambiguous
          : undefined,
      );
    });
    return () => {
      current = false;
    };
  }, [pathname, search]);
  return count;
}
