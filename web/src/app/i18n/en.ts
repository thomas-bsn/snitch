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
