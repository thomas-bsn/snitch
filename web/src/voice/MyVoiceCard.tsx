import React, { useCallback, useEffect, useState } from "react";
import {
  voiceApi,
  voiceChecked,
  type CsrfHeaders,
  type Person,
} from "./client";
import "./voice.css";

type MyClip = { id: number; original_name: string; recorded_at: string };

// The signed-in account's own Person: where their voice is identified (only clips
// their visibility allows) and the deletion of their voice profile.
export function MyVoiceCard({
  csrf,
}: {
  csrf: CsrfHeaders;
}): React.ReactElement | null {
  const [person, setPerson] = useState<Person | null | undefined>(undefined);
  const [clips, setClips] = useState<MyClip[]>([]);
  const [message, setMessage] = useState("");
  const [confirming, setConfirming] = useState(false);

  const load = useCallback(async () => {
    const mine = await voiceApi.GET("/api/me/person");
    if (!mine.data) {
      setPerson(null);
      return;
    }
    setPerson(mine.data);
    const list = await voiceApi.GET("/api/clips", {
      params: { query: { person_id: mine.data.id } },
    });
    setClips(list.data?.items ?? []);
  }, []);

  useEffect(() => {
    load().catch(() => setPerson(null));
  }, [load]);

  async function erase() {
    if (!person) return;
    try {
      const report = voiceChecked(
        await voiceApi.DELETE("/api/persons/{id}/voice-profile", {
          params: { header: csrf, path: { id: person.id } },
        }),
      );
      setConfirming(false);
      setMessage(
        `Profil vocal supprimé : ${report.samples} échantillon(s), ${report.fingerprints} empreinte(s), ${report.attributions} attribution(s) effacé(s).`,
      );
      await load();
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    }
  }

  if (!person) return null;
  const deleted = person.voice_profile_deleted_at;
  return (
    <section className="card" id="my-voice" aria-label="Ma voix">
      <h2>Ma voix</h2>
      <p>
        Vous êtes la Personne <strong>{person.display_name}</strong>.{" "}
        {deleted
          ? "Votre profil vocal est supprimé : Snitch ne vous reconnaît plus."
          : `${person.validated_sample_count} échantillon(s) vocal(aux) validé(s) servent de référence.`}
      </p>
      <h3>Clips où vous êtes identifié(e)</h3>
      {clips.length === 0 ? (
        <p className="hint">Aucun clip visible pour vous.</p>
      ) : (
        <ul aria-label="Mes clips">
          {clips.map((clip) => (
            <li key={clip.id}>
              {clip.original_name} ·{" "}
              {new Date(clip.recorded_at).toLocaleDateString("fr-FR")}
            </li>
          ))}
        </ul>
      )}
      {!deleted && !confirming && (
        <button className="secondary" onClick={() => setConfirming(true)}>
          Supprimer mon profil vocal
        </button>
      )}
      {!deleted && confirming && (
        <div aria-live="polite">
          <p>
            Vos échantillons, vos empreintes et vos attributions seront effacés,
            et vos voix redeviendront inconnues. Vos clips restent en place.
            Cette action est définitive.
          </p>
          <button onClick={erase}>Confirmer la suppression</button>{" "}
          <button className="secondary" onClick={() => setConfirming(false)}>
            Garder mon profil
          </button>
        </div>
      )}
      <p aria-live="polite">{message}</p>
    </section>
  );
}
