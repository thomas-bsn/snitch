import { expect, test, type Page } from "@playwright/test";
import { prepareDiarizedClip } from "./voice-fixture";

// A second user, signed in apart, corrects the first segment of the only clip behind
// the first user's back.
async function bobCorrectsFirstSegment(
  page: Page,
  baseURL: string,
  version: number,
  text: string,
): Promise<void> {
  const clipId = await page.evaluate(async () => {
    const [clips, me] = await Promise.all([
      fetch("/api/clips?page_size=100").then((response) => response.json()),
      fetch("/api/auth/me").then((response) => response.json()),
    ]);
    await fetch("/api/admin/settings", {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": me.csrf_token,
        Origin: location.origin,
      },
      body: JSON.stringify({ client_id: "", registration_approval: false }),
    });
    return clips.items[0].id as number;
  });
  const context = await page.context().browser()!.newContext({ baseURL });
  try {
    const headers = { Origin: baseURL };
    const credentials = { username: "bob", password: "synthetic-password-123" };
    // The registration is refused the second time (already registered): ignored.
    await context.request.post("/api/auth/register", {
      data: credentials,
      headers,
    });
    await context.request.post("/api/auth/login", {
      data: credentials,
      headers,
    });
    const me = await (await context.request.get("/api/auth/me")).json();
    const response = await context.request.put(
      `/api/clips/${clipId}/segments/0/text`,
      {
        data: { text, expected_version: version },
        headers: { ...headers, "X-CSRF-Token": me.csrf_token },
      },
    );
    expect(response.status()).toBe(200);
  } finally {
    await context.close();
  }
}

test("deux utilisateurs corrigent le même segment : le second voit le conflit et reprend ou abandonne", async ({
  page,
  baseURL,
}) => {
  await prepareDiarizedClip(page);
  await page
    .getByRole("button", { name: "Actualiser la transcription" })
    .click();
  const transcript = page.getByRole("region", {
    name: "Transcription horodatée",
  });
  await expect(transcript).toContainText("Bien joué");
  const editor = transcript.getByRole("textbox", {
    name: "Texte du segment 1",
  });
  const conflict = transcript.getByRole("group", {
    name: "Conflit sur le segment 1",
  });

  // The administrator starts editing, then bob saves the same segment first.
  await transcript
    .getByRole("button", { name: "Modifier le texte du segment 1" })
    .click();
  await editor.fill("Version de l’administrateur");
  await bobCorrectsFirstSegment(page, baseURL!, 0, "Version de Bob");
  await transcript
    .getByRole("button", { name: "Enregistrer le texte du segment 1" })
    .click();
  await expect(conflict).toContainText("Ce segment a été modifié par bob le");
  await expect(conflict).toContainText("Version actuelle : « Version de Bob »");
  // Nothing was overwritten: the draft is kept and bob's text is the effective one.
  await expect(editor).toHaveValue("Version de l’administrateur");
  await expect(transcript).toContainText("Corrigé par bob le");

  // Resuming keeps the draft on top of the current version.
  await conflict
    .getByRole("button", { name: "Reprendre ma modification du segment 1" })
    .click();
  await expect(conflict).toHaveCount(0);
  await expect(editor).toHaveValue("Version de l’administrateur");
  await transcript
    .getByRole("button", { name: "Enregistrer le texte du segment 1" })
    .click();
  await expect(transcript.getByText("Correction enregistrée.")).toBeVisible();
  await expect(transcript).toContainText("Version de l’administrateur");
  await expect(transcript).toContainText("Corrigé par administrator le");

  // A second conflict can be abandoned: the draft is dropped, bob's text stays.
  await transcript
    .getByRole("button", { name: "Modifier le texte du segment 1" })
    .click();
  await editor.fill("Brouillon abandonné");
  await bobCorrectsFirstSegment(page, baseURL!, 2, "Seconde version de Bob");
  await transcript
    .getByRole("button", { name: "Enregistrer le texte du segment 1" })
    .click();
  await expect(conflict).toContainText("Seconde version de Bob");
  await conflict
    .getByRole("button", { name: "Abandonner ma modification du segment 1" })
    .click();
  await expect(conflict).toHaveCount(0);
  await expect(editor).toHaveCount(0);
  await expect(transcript).toContainText("Seconde version de Bob");
  await expect(transcript).not.toContainText("Brouillon abandonné");
  await expect(transcript).toContainText("Corrigé par bob le");
});
