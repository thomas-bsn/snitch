import { useEffect, useState } from "react";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { buttonClass } from "../../components/buttons";

export function Avatar({ initials }: { initials: string }) {
  const [version, setVersion] = useState(0);
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    const update = () => {
      setFailed(false);
      setVersion(Date.now());
    };
    window.addEventListener("snitch-avatar", update);
    return () => window.removeEventListener("snitch-avatar", update);
  }, []);
  return failed ? (
    <span aria-hidden="true">{initials}</span>
  ) : (
    <img
      className="size-full rounded-full object-cover"
      alt=""
      src={`/api/account/avatar?v=${version}`}
      onError={() => setFailed(true)}
    />
  );
}

export function AvatarEditor() {
  const { language } = useI18n();
  const { session, user } = useSession();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const fr = language === "fr";
  async function save(file?: File) {
    if (file && file.size > 5 * 1024 * 1024) {
      setMessage(fr ? "La limite est de 5 Mo." : "The limit is 5 MB.");
      return;
    }
    setBusy(true);
    setMessage("");
    try {
      const response = await fetch("/api/account/avatar", {
        method: file ? "PUT" : "DELETE",
        headers: {
          "X-CSRF-Token": session?.csrf_token ?? "",
          ...(file ? { "Content-Type": file.type } : {}),
        },
        body: file,
      });
      if (!response.ok) throw new Error();
      window.dispatchEvent(new Event("snitch-avatar"));
      setMessage(
        fr ? "Photo de profil mise à jour." : "Profile photo updated.",
      );
    } catch {
      setMessage(
        fr
          ? "Impossible de modifier la photo. Choisissez une image JPEG ou PNG (5 Mo, 16 mégapixels maximum)."
          : "Could not update the photo. Choose a JPEG or PNG image (up to 5 MB and 16 megapixels).",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-3">
      <h3 className="font-medium">
        {fr ? "Photo de profil" : "Profile photo"}
      </h3>
      <div className="flex size-16 items-center justify-center rounded-full bg-avatar-bg text-accent-ink">
        <Avatar initials={user?.username.slice(0, 2).toUpperCase() ?? ""} />
      </div>
      <label className="flex flex-col gap-2">
        {fr
          ? "Choisir une photo (JPEG ou PNG)"
          : "Choose a photo (JPEG or PNG)"}
        <input
          type="file"
          accept="image/png,image/jpeg"
          disabled={busy}
          onChange={(e) => {
            const file = e.target.files?.[0];
            e.target.value = "";
            if (file) void save(file);
          }}
        />
      </label>
      <p className="text-sm text-mute">
        {fr
          ? "L’image sera recadrée au centre en carré."
          : "The image will be cropped to a centred square."}
      </p>
      <div>
        <button
          className={buttonClass("secondary")}
          disabled={busy}
          onClick={() => void save()}
        >
          {fr ? "Retirer la photo" : "Remove photo"}
        </button>
      </div>
      <p aria-live="polite" className="text-sm">
        {message}
      </p>
    </div>
  );
}
