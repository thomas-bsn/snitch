import type { User } from "./api/client";

// The only place the interface decides what a role may do. The role model will grow (a
// super-administrator above the administrator): every check goes through these
// functions so that growing it changes this file only.

/** True for the people who can use the administration screens. */
export function isAdministrator(user: Pick<User, "role">): boolean {
  return user.role === "admin";
}

/** Text key of the role shown in the profile menu. */
export function roleLabelKey(
  user: Pick<User, "role">,
): "roles.admin" | "roles.user" {
  return isAdministrator(user) ? "roles.admin" : "roles.user";
}
