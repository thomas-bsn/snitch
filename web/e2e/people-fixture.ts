import { adminApi } from "./library-fixture";

// Synthetic directory of Persons for the journeys of the People list: everything goes
// through the API, with deterministic names, and no real data.

/** The name of the n-th generated Person: "Personne 0001", "Personne 0002"… */
export function personName(n: number) {
  return `Personne ${String(n).padStart(4, "0")}`;
}

/** Creates `names` as Persons, by batches, and returns their ids in the same order. */
export async function createPersons(names: string[]): Promise<number[]> {
  const { api, post } = await adminApi();
  const ids: number[] = [];
  for (let start = 0; start < names.length; start += 50) {
    const batch = await Promise.all(
      names
        .slice(start, start + 50)
        .map((display_name) => post("/api/persons", { display_name })),
    );
    for (const person of batch) ids.push((person as { id: number }).id);
  }
  await api.dispose();
  return ids;
}

/** Links the account of `username` to the Person (administrator only). */
export async function linkAccount(personId: number, username: string) {
  const { api, put } = await adminApi();
  await put(`/api/admin/persons/${personId}/account`, { username });
  await api.dispose();
}
