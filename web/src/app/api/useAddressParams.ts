import { useCallback } from "react";
import { useSearchParams } from "react-router";

/**
 * The parameters of the address, with a way to change them. A change is built from the
 * address as it is when it applies, not from the last render: the router's state can trail
 * the address by a render, and a change made from it would undo the one just before (a mode
 * then a filter, a filter then a clear).
 */
export function useAddressParams() {
  const [params, setParams] = useSearchParams();
  const update = useCallback(
    (edit: (next: URLSearchParams) => void) => {
      const next = new URLSearchParams(window.location.search);
      edit(next);
      setParams(next);
    },
    [setParams],
  );
  return { params, update };
}
