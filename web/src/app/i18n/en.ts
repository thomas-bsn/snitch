// English texts. This file is the reference: fr.ts must have exactly the same keys, which
// TypeScript enforces. Placeholders look like {name}.

import { errorsEn } from "./errors.en";

export const en = {
  app: {
    name: "Snitch",
    skipToContent: "Skip to content",
    loading: "Loading…",
    previousInterface: "Previous interface",
  },
  nav: {
    label: "Main navigation",
    library: "Library",
    search: "Search",
    people: "People",
    import: "Import",
  },
  header: {
    searchLabel: "Search the transcripts",
    searchPlaceholder: "Search what was said…",
    import: "Import",
    accountMenu: "Account menu",
  },
  menu: {
    account: "Account settings",
    administration: "Administration",
    theme: "Theme",
    themeLight: "Light",
    themeDark: "Dark",
    themeSystem: "System",
    language: "Language",
    signOut: "Sign out",
  },
  roles: {
    admin: "Administrator",
    user: "User",
  },
  language: {
    en: "English",
    fr: "Français",
  },
  auth: {
    username: "Username",
    password: "Password",
    passwordHint: "12 to 256 characters.",
    usernameHint: "3 to 64 characters.",
    login: {
      title: "Sign in",
      lead: "Use your Snitch account.",
      submit: "Sign in",
      submitting: "Signing in…",
      toRegister: "Create an account",
    },
    register: {
      title: "Request an account",
      lead: "Choose a username and a password.",
      submit: "Request access",
      submitting: "Sending…",
      toLogin: "Back to sign in",
    },
    discord: {
      button: "Continue with Discord",
      hint: "You will be redirected to Discord. Only your public identity is requested.",
    },
    languageChoice: "Language",
  },
  notices: {
    registeredPending:
      "Your request was sent. An administrator has to approve it before you can sign in.",
    registeredActive: "Account created. You can sign in.",
    pending: "Your Discord sign-up is waiting for an administrator's approval.",
    discordLinked: "Your Discord identity is linked to your account.",
    dismiss: "Dismiss this message",
  },
  connection: {
    title: "Can't reach Snitch",
    body: "The server did not answer. Check that it is running, then try again.",
    retry: "Try again",
  },
  installation: {
    title: "Installation is not finished",
    body: "Snitch has no administrator yet. The guided installation is not available in this interface yet: finish it in the previous interface, with the installation code printed in the server logs.",
    open: "Open the previous interface",
  },
  account: {
    tabsLabel: "Account settings",
    tabs: { profile: "Profile", voice: "My voice" },
    profile: {
      identityTitle: "Account",
      username: "Username",
      role: "Role",
      discordTitle: "Discord",
      discordBody:
        "Link your Discord identity to this account, so that you can sign in with Discord. The link is made to this account only, never by matching usernames. This opens Discord.",
      discordLink: "Link Discord",
      discordLinking: "Opening Discord…",
      discordDisabled:
        "Discord sign-in is not enabled on this instance. An administrator can enable it.",
      preferencesTitle: "Preferences",
      language: "Language",
      languageHint:
        "Kept in your account, so that it follows you on your other devices.",
      languageSaved: "Language saved.",
    },
    voice: {
      title: "My voice",
      loading: "Loading your voice…",
      noPersonTitle: "No voice profile",
      noPerson:
        "No person is linked to your account, so Snitch holds no voice profile for you. An administrator links a person to an account.",
      identity: "You are the person {name}.",
      samples:
        "{count} validated voice sample(s) serve as the reference for your voice.",
      deleted: "Your voice profile is deleted: Snitch no longer recognizes you.",
      clipsTitle: "Clips where you are identified",
      clipsLabel: "My clips",
      noClips: "No clip visible to you.",
      deleteTitle: "Delete my voice profile",
      deleteEffects:
        "Deleting your voice profile erases your validated voice samples, your voice fingerprints and every attribution made to your name. Your voices become unknown again and nobody can attribute a voice to you afterwards. Your clips stay in place. This cannot be undone.",
      deleteButton: "Delete my voice profile",
      confirmTitle: "Delete your voice profile?",
      confirmBody:
        "Your samples, your fingerprints and your attributions will be erased, and your voices will become unknown again. Your clips stay in place. This cannot be undone.",
      confirm: "Delete my voice profile",
      deleting: "Deleting…",
      keep: "Keep my profile",
      done: "Voice profile deleted: {samples} sample(s), {fingerprints} fingerprint(s) and {attributions} attribution(s) erased.",
    },
  },
  pages: {
    comingSoon:
      "This screen is being rebuilt. The previous interface still offers it in the meantime.",
    library: { title: "Library" },
    search: { title: "Search", queryLabel: "Searching for “{query}”." },
    people: { title: "People" },
    import: { title: "Import" },
    account: { title: "Account settings" },
    admin: { title: "Administration" },
    forbidden: {
      title: "Administrators only",
      body: "Your account cannot open this screen.",
    },
    notFound: {
      title: "Page not found",
      body: "This address does not match any screen.",
      back: "Back to the library",
    },
  },
  errors: errorsEn,
  errorFallback: "Something went wrong ({code}). Try again.",
};
