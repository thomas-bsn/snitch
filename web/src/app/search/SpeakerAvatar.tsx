import { UserRound } from "lucide-react";
import { Avatar } from "../people/PersonParts";
import type { SearchResult } from "./searchApi";

/** The Person's picture when the voice is identified, a neutral silhouette otherwise. */
export function SpeakerAvatar({
  person,
  size = "md",
}: {
  person: SearchResult["person"];
  size?: "xs" | "md" | "lg" | "record";
}) {
  if (person) {
    return (
      <Avatar name={person.display_name} personId={person.id} size={size} />
    );
  }
  const box =
    size === "xs"
      ? "size-6"
      : size === "lg"
        ? "size-14"
        : size === "record"
          ? "size-12"
          : "size-10";
  return (
    <span
      aria-hidden="true"
      className={`inline-flex shrink-0 items-center justify-center rounded-full bg-side text-mute ${box}`}
    >
      <UserRound
        className={size === "xs" ? "size-3.5" : "size-5"}
        aria-hidden="true"
      />
    </span>
  );
}
