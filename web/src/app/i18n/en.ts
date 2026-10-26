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
  admin: {
    title: "Administration",
    tabsLabel: "Administration sections",
    subTabsLabel: "Subsections",
    tabs: {
      workers: "Workers",
      queue: "Task queue",
      folders: "Folders",
      transcription: "Transcription",
      voice: "Voice processing",
      semantic: "Semantic index",
      users: "Users and access",
      exports: "Exports",
    },
    subTabs: {
      model: "Model",
      vocabulary: "Vocabulary",
      members: "Members",
      signups: "Sign-ups",
    },
  },
  workers: {
    title: "Workers",
    lead: "A Worker is a machine that transcribes and processes voices for the library.",
    add: "Add a Worker",
    loading: "Loading the Workers…",
    loadFailed: "The Workers could not be loaded.",
    empty: "No Worker yet. Add your first Worker to start transcribing.",
    open: "Open {name}",
    columns: {
      name: "Name",
      status: "Status",
      lastPulse: "Last pulse",
      models: "Models",
    },
    status: {
      online: "Online",
      offline: "Offline",
      revoked: "Revoked",
    },
    never: "Never",
    notDeclared: "Not declared",
    models: {
      transcription: "Transcription",
      diarization: "Diarization",
      fingerprint: "Voice fingerprint",
    },
    detail: {
      back: "All Workers",
      notFound: "This Worker does not exist.",
      activity: "Activity",
      lastPulse: "Last pulse",
      concurrency: "Concurrency",
      concurrencyValue: "{count} at a time",
      created: "Created",
      recentTasks: "Recent tasks",
      noTasks: "This Worker has not done any task yet.",
      taskColumns: {
        kind: "Task",
        clip: "Clip",
        outcome: "Result",
        when: "When",
      },
      taskKinds: {
        transcription: "Transcription",
        diarization: "Diarization",
        fingerprint: "Voice fingerprints",
        other: "Other",
      },
      outcomes: {
        started: "Started",
        completed: "Completed",
        retry: "Will retry",
        failed: "Failed",
      },
      models: "Models in use",
      modelsNone: "Not declared. Start the Worker to see its models.",
      deployment: "Deployment files",
      deploymentLead:
        "Put both files in one folder on the machine, then run docker compose up -d.",
      tokenHidden:
        "The token is never shown again. The files below hold a placeholder: generate a new token to get working files.",
      regenerate: "Generate a new token…",
      regenerateTitle: "Generate a new token?",
      regenerateBody:
        "The current token stops working immediately. You will have to put the new files on the Worker's machine.",
      regenerateConfirm: "Generate",
      newToken:
        "New token generated. Copy the files now: the token will not be shown again.",
      revoke: "Revoke",
      revokeBody:
        "A revoked Worker can no longer take tasks. Its history stays.",
      revokeButton: "Revoke this Worker…",
      revokeTitle: "Revoke {name}?",
      revokeConfirm: "Revoke",
      revokeConfirmBody:
        "This Worker will stop receiving tasks. You cannot undo this.",
      revokedNotice:
        "This Worker was revoked on {date}. It cannot take tasks any more.",
    },
    create: {
      title: "Add a Worker",
      lead: "Tell Snitch where this Worker finds its Whisper server. Snitch writes the deployment files for its machine.",
      name: "Worker name",
      nameHint:
        "1 to 64 characters. Letters, digits, spaces, dots, dashes and underscores.",
      snitch: "Snitch address",
      snitchHint: "Filled in by Snitch: the address the Worker will call.",
      insecure:
        "This address is not HTTPS and the Worker refuses plain HTTP. Open Snitch over HTTPS, or fix SNITCH_API_URL in the .env file.",
      whisper: "Whisper server address",
      whisperHint:
        "Any server that answers POST /v1/audio/transcriptions. On the gaming PC itself: http://host.docker.internal:8000",
      model: "Whisper model name",
      modelHint:
        "The model for the whole library. The first Worker sets it; you can change it later in Administration, Transcription.",
      language: "Language",
      languageHint: "One language for the whole library.",
      libraryModel: "Library model",
      libraryModelHint:
        "Set when the first Worker was created, shared by every Worker. Change it in Administration, Transcription.",
      hf: "Hugging Face token",
      hfHint:
        "Accept the pyannote conditions on Hugging Face first, then paste a read-only token. It only serves to download the voice models on the first start. Snitch never receives it and does not keep it: it is written into the .env file below, in your browser.",
      submit: "Create the Worker",
      submitting: "Creating…",
      cancel: "Cancel",
      confirmTitle: "Use this model for the whole library?",
      confirmBody:
        "This model and this language will be used by every future Worker, even with a different Whisper server. You can change them later in Administration, Transcription.",
      confirmBack: "Back",
      confirmSubmit: "Confirm and create",
      shownOnce:
        "Copy both files now. The Worker token is shown only once and Snitch cannot show it again.",
      secret:
        "The .env file holds the Worker token and your Hugging Face token: never share it, never commit it.",
      hfMissing:
        "No Hugging Face token was given: fill in HF_TOKEN by hand in the .env file.",
      whisperMissing:
        "No Whisper address was given: fill in TRANSCRIPTION_API_URL by hand in the .env file.",
      done: "I copied the files",
    },
    files: {
      tabsLabel: "Deployment files",
      env: ".env",
      compose: "compose.yaml",
      copy: "Copy",
      copied: "Copied",
      copyFailed: "The browser refused to copy: select the text instead.",
      download: "Download",
      fileLabel: "Content of {file}",
    },
    dialog: {
      cancel: "Cancel",
    },
  },
  errors: errorsEn,
  errorFallback: "Something went wrong ({code}). Try again.",
};
