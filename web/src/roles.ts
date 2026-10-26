import type { components } from "./api.gen";

export type Role = components["schemas"]["User"]["role"];

// The one place of the interface that decides whether a role may use the administration:
// Administrator and Super-administrator are treated alike for everything that is not reserved.
export function isAdministrator(role: Role): boolean {
  return role === "admin" || role === "super_admin";
}

// Reserved to the owner of the installation: Discord, registration approval, master key export
// and the accounts of Administrators.
export function isSuperAdministrator(role: Role): boolean {
  return role === "super_admin";
}

export function roleLabel(role: Role): string {
  if (role === "super_admin") return "Super-administrateur";
  if (role === "admin") return "Administrateur";
  return "Membre";
}
