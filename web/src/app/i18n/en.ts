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
    superAdmin: "Super-administrator",
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
  setup: {
    title: "Set up Snitch",
    lead: "Create the Super-administrator account first. Everything else is optional, and nothing starts without your say-so.",
    stepOf: "Step {step} of {total}",
    stepsLabel: "Setup steps",
    steps: {
      code: "Installation code",
      admin: "Super-administrator",
      discord: "Discord",
      semantic: "Semantic search",
      worker: "First Worker",
      import: "Import videos",
      summary: "Summary",
    },
    tags: {
      optional: "Optional",
      skipped: "Skipped",
      done: "Done",
      todo: "To do",
    },
    where: {
      discord: "Administration, Users and access",
      semantic: "Administration, Semantic index",
      worker: "Administration, Workers",
      import: "the Import page",
    },
    back: "Back",
    continue: "Continue",
    skip: "Skip for now",
    saveContinue: "Save and continue",
    saving: "Saving…",
    code: {
      title: "Enter the installation code",
      lead: "This proves you run the server. The code is never shown in this interface.",
      where: "Read it in the server logs",
      whereHint:
        "Run this command and copy the code from the line that gives the address and the code. It is valid for 15 minutes and works once.",
      label: "Installation code",
      invalid:
        "This code is wrong or has expired. Read the logs again and copy the code of the latest line. If 15 minutes have passed, restart the application (docker compose restart app) to get a new one.",
      checking: "Checking…",
    },
    admin: {
      title: "Create the Super-administrator",
      lead: "This account owns the installation: it cannot be demoted, deleted or closed. You can add other administrators later.",
      confirm: "Confirm the password",
      mismatch: "The two passwords do not match.",
      submit: "Create the account",
      submitting: "Creating…",
      codeLost:
        "The installation code is no longer valid. Go back, read the logs again and enter the new code: what you typed here is kept.",
    },
    discord: {
      title: "Sign in with Discord",
      lead: "Let members sign in with Discord. You use your own Discord application: nothing is shared with anyone else. Local accounts keep working either way.",
      appId: "Application ID",
      secret: "Application secret",
      secretHint: "Stored encrypted. It is never shown again.",
      secretKept: "Leave empty to keep the current secret.",
      returnUrl: "Return address to register in Discord",
      returnUrlHint:
        "Add this address to the redirects of your Discord application.",
      already: "Discord is already configured.",
      ownerOnly:
        "Only the Super-administrator can configure Discord. Skip this step, and ask them to do it from the administration.",
    },
    semantic: {
      title: "Semantic search",
      lead: "Search by meaning, not only by exact words. Full-text search works without it, and nothing is indexed until you ask.",
      status: "Status",
      active: "Active · {model}",
      inactive: "Not configured",
      howTo:
        "The embeddings service is part of the deployment, but its model is chosen in the deployment variables. To enable it, set the model, make it available, then restart the application.",
      howToHint:
        "Replace the model with the one you prefer. You can also do this later.",
    },
    worker: {
      title: "Create your first Worker",
      lead: "A Worker transcribes clips and analyzes voices on your own machine. Give its Whisper server and your Hugging Face token: Snitch prepares its deployment files.",
    },
    import: {
      title: "Import your first videos",
      lead: "Send clips from the browser; an interrupted transfer can be resumed.",
      nothing:
        "Nothing is scanned or processed unless you ask. The wizard only takes you to the Import page.",
      open: "Open the Import page",
    },
    summary: {
      title: "Summary",
      lead: "Here is where the installation stands. What you skipped can be finished later.",
      leadDone: "Everything is configured.",
      caption: "State of each setup step",
      finishIn: "Finish it in {where}",
      banner:
        "A “Finish the setup” banner will remind you of what is left until you close it.",
      open: "Open the library",
    },
    banner: {
      label: "Setup",
      title: "Finish the setup",
      body: "Optional steps are still to do ({count}). Resume where you left off.",
      resume: "Resume the setup",
      dismiss: "Hide this banner for good",
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
