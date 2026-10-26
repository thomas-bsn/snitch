import type { User } from "./api/client";

// The only place the interface decides what a role may do: every check goes through these
// functions so that changing the role model changes this file only.

/** True for the people who can use the administration screens (Administrator and Super-administrator alike). */
export function isAdministrator(user: Pick<User, "role">): boolean {
  return user.role === "admin" || user.role === "super_admin";
}

/** True for the owner of the installation: Discord, registration approval, master key, Administrator accounts. */
export function isSuperAdministrator(user: Pick<User, "role">): boolean {
  return user.role === "super_admin";
}

/** Text key of the role shown in the profile menu. */
export function roleLabelKey(
  user: Pick<User, "role">,
): "roles.superAdmin" | "roles.admin" | "roles.user" {
  if (isSuperAdministrator(user)) return "roles.superAdmin";
  return isAdministrator(user) ? "roles.admin" : "roles.user";
}
