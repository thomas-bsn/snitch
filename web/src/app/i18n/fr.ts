// Textes français. Mêmes clés que en.ts, vérifiées par le compilateur.

import type { Messages } from "./messages";
import { errorsFr } from "./errors.fr";

export const fr: Messages = {
  app: {
    name: "Snitch",
    skipToContent: "Aller au contenu",
    loading: "Chargement…",
    previousInterface: "Ancienne interface",
  },
  nav: {
    label: "Navigation principale",
    library: "Bibliothèque",
    search: "Recherche",
    people: "Personnes",
    import: "Importer",
  },
  header: {
    searchLabel: "Rechercher dans les transcriptions",
    searchPlaceholder: "Rechercher ce qui a été dit…",
    import: "Importer",
    accountMenu: "Menu du compte",
  },
  menu: {
    account: "Paramètres du compte",
    administration: "Administration",
    theme: "Thème",
    themeLight: "Clair",
    themeDark: "Sombre",
    themeSystem: "Système",
    language: "Langue",
    signOut: "Se déconnecter",
  },
  roles: {
    admin: "Administrateur",
    user: "Utilisateur",
  },
  language: {
    en: "English",
    fr: "Français",
  },
  auth: {
    username: "Nom d’utilisateur",
    password: "Mot de passe",
    passwordHint: "12 à 256 caractères.",
    usernameHint: "3 à 64 caractères.",
    login: {
      title: "Connexion",
      lead: "Utilisez votre compte Snitch.",
      submit: "Se connecter",
      submitting: "Connexion…",
      toRegister: "Créer un compte",
    },
    register: {
      title: "Demander un compte",
      lead: "Choisissez un nom d’utilisateur et un mot de passe.",
      submit: "Demander l’accès",
      submitting: "Envoi…",
      toLogin: "Retour à la connexion",
    },
    discord: {
      button: "Continuer avec Discord",
      hint: "Vous serez redirigé vers Discord. Seule votre identité publique est demandée.",
    },
    languageChoice: "Langue",
  },
  notices: {
    registeredPending:
      "Votre demande est envoyée. Un administrateur doit l’approuver avant que vous puissiez vous connecter.",
    registeredActive: "Compte créé. Vous pouvez vous connecter.",
    pending:
      "Votre inscription Discord attend l’approbation d’un administrateur.",
    dismiss: "Fermer ce message",
  },
  connection: {
    title: "Snitch est injoignable",
    body: "Le serveur ne répond pas. Vérifiez qu’il fonctionne, puis réessayez.",
    retry: "Réessayer",
  },
  installation: {
    title: "L’installation n’est pas terminée",
    body: "Snitch n’a pas encore d’administrateur. L’installation guidée n’est pas encore disponible dans cette interface : terminez-la dans l’ancienne interface, avec le code d’installation affiché dans les journaux du serveur.",
    open: "Ouvrir l’ancienne interface",
  },
  pages: {
    comingSoon:
      "Cet écran est en cours de refonte. L’ancienne interface le propose en attendant.",
    library: { title: "Bibliothèque" },
    search: { title: "Recherche", queryLabel: "Recherche de « {query} »." },
    people: { title: "Personnes" },
    import: { title: "Importer" },
    account: { title: "Paramètres du compte" },
    admin: { title: "Administration" },
    forbidden: {
      title: "Réservé aux administrateurs",
      body: "Votre compte ne peut pas ouvrir cet écran.",
    },
    notFound: {
      title: "Page introuvable",
      body: "Cette adresse ne correspond à aucun écran.",
      back: "Retour à la bibliothèque",
    },
  },
  admin: {
    title: "Administration",
    tabsLabel: "Sections de l’administration",
    subTabsLabel: "Sous-sections",
    tabs: {
      workers: "Workers",
      queue: "File de tâches",
      folders: "Dossiers",
      transcription: "Transcription",
      voice: "Traitement vocal",
      semantic: "Index sémantique",
      users: "Utilisateurs et accès",
      exports: "Exports",
    },
    subTabs: {
      model: "Modèle",
      vocabulary: "Vocabulaire",
      members: "Membres",
      signups: "Inscriptions",
    },
  },
  workers: {
    title: "Workers",
    lead: "Un Worker est une machine qui transcrit et traite les voix de la bibliothèque.",
    add: "Ajouter un Worker",
    loading: "Chargement des Workers…",
    loadFailed: "Les Workers n’ont pas pu être chargés.",
    empty:
      "Aucun Worker pour l’instant. Ajoutez votre premier Worker pour commencer à transcrire.",
    open: "Ouvrir {name}",
    columns: {
      name: "Nom",
      status: "État",
      lastPulse: "Dernier pouls",
      models: "Modèles",
    },
    status: {
      online: "En ligne",
      offline: "Hors ligne",
      revoked: "Révoqué",
    },
    never: "Jamais",
    notDeclared: "Non déclaré",
    models: {
      transcription: "Transcription",
      diarization: "Diarisation",
      fingerprint: "Empreinte vocale",
    },
    detail: {
      back: "Tous les Workers",
      notFound: "Ce Worker n’existe pas.",
      activity: "Activité",
      lastPulse: "Dernier pouls",
      concurrency: "Concurrence",
      concurrencyValue: "{count} à la fois",
      created: "Créé le",
      recentTasks: "Tâches récentes",
      noTasks: "Ce Worker n’a encore fait aucune tâche.",
      taskColumns: {
        kind: "Tâche",
        clip: "Clip",
        outcome: "Résultat",
        when: "Quand",
      },
      taskKinds: {
        transcription: "Transcription",
        diarization: "Diarisation",
        fingerprint: "Empreintes vocales",
        other: "Autre",
      },
      outcomes: {
        started: "Démarrée",
        completed: "Terminée",
        retry: "Nouvel essai prévu",
        failed: "Échouée",
      },
      models: "Modèles utilisés",
      modelsNone: "Non déclarés. Démarrez le Worker pour voir ses modèles.",
      deployment: "Fichiers de déploiement",
      deploymentLead:
        "Placez les deux fichiers dans un même dossier de la machine, puis lancez docker compose up -d.",
      tokenHidden:
        "Le jeton n’est jamais réaffiché. Les fichiers ci-dessous contiennent un emplacement : générez un nouveau jeton pour obtenir des fichiers utilisables.",
      regenerate: "Générer un nouveau jeton…",
      regenerateTitle: "Générer un nouveau jeton ?",
      regenerateBody:
        "Le jeton actuel cesse de fonctionner immédiatement. Vous devrez poser les nouveaux fichiers sur la machine du Worker.",
      regenerateConfirm: "Générer",
      newToken:
        "Nouveau jeton généré. Copiez les fichiers maintenant : le jeton ne sera plus affiché.",
      revoke: "Révocation",
      revokeBody:
        "Un Worker révoqué ne peut plus prendre de tâches. Son historique est conservé.",
      revokeButton: "Révoquer ce Worker…",
      revokeTitle: "Révoquer {name} ?",
      revokeConfirm: "Révoquer",
      revokeConfirmBody:
        "Ce Worker cessera de recevoir des tâches. Cette action est irréversible.",
      revokedNotice:
        "Ce Worker a été révoqué le {date}. Il ne peut plus prendre de tâches.",
    },
    create: {
      title: "Ajouter un Worker",
      lead: "Indiquez à Snitch où ce Worker trouve son serveur Whisper. Snitch écrit les fichiers de déploiement pour sa machine.",
      name: "Nom du Worker",
      nameHint:
        "1 à 64 caractères. Lettres, chiffres, espaces, points, tirets et soulignés.",
      snitch: "Adresse de Snitch",
      snitchHint: "Renseignée par Snitch : l’adresse que le Worker appellera.",
      insecure:
        "Cette adresse n’est pas en HTTPS et le Worker refuse le HTTP. Ouvrez Snitch en HTTPS, ou corrigez SNITCH_API_URL dans le fichier .env.",
      whisper: "Adresse du serveur Whisper",
      whisperHint:
        "Tout serveur qui répond à POST /v1/audio/transcriptions. Sur le PC de jeu lui-même : http://host.docker.internal:8000",
      model: "Nom du modèle Whisper",
      modelHint:
        "Le modèle de toute la bibliothèque. Le premier Worker le fixe ; vous pourrez le changer ensuite dans Administration, Transcription.",
      language: "Langue",
      languageHint: "Une seule langue pour toute la bibliothèque.",
      libraryModel: "Modèle de la bibliothèque",
      libraryModelHint:
        "Fixé à la création du premier Worker, partagé par tous les Workers. À changer dans Administration, Transcription.",
      hf: "Jeton Hugging Face",
      hfHint:
        "Acceptez d’abord les conditions de pyannote sur Hugging Face, puis collez un jeton en lecture seule. Il ne sert qu’à télécharger les modèles de voix au premier démarrage. Snitch ne le reçoit jamais et ne le conserve pas : il est écrit dans le fichier .env ci-dessous, dans votre navigateur.",
      submit: "Créer le Worker",
      submitting: "Création…",
      cancel: "Annuler",
      confirmTitle: "Utiliser ce modèle pour toute la bibliothèque ?",
      confirmBody:
        "Ce modèle et cette langue serviront à tous les futurs Workers, même avec un serveur Whisper différent. Vous pourrez les changer plus tard dans Administration, Transcription.",
      confirmBack: "Retour",
      confirmSubmit: "Confirmer et créer",
      shownOnce:
        "Copiez les deux fichiers maintenant. Le jeton du Worker n’est affiché qu’une fois et Snitch ne pourra plus le montrer.",
      secret:
        "Le fichier .env contient le jeton du Worker et votre jeton Hugging Face : ne le partagez ni ne le versionnez jamais.",
      hfMissing:
        "Aucun jeton Hugging Face n’a été donné : renseignez HF_TOKEN à la main dans le fichier .env.",
      whisperMissing:
        "Aucune adresse Whisper n’a été donnée : renseignez TRANSCRIPTION_API_URL à la main dans le fichier .env.",
      done: "J’ai copié les fichiers",
    },
    files: {
      tabsLabel: "Fichiers de déploiement",
      env: ".env",
      compose: "compose.yaml",
      copy: "Copier",
      copied: "Copié",
      copyFailed:
        "Le navigateur a refusé la copie : sélectionnez le texte à la place.",
      download: "Télécharger",
      fileLabel: "Contenu de {file}",
    },
    dialog: {
      cancel: "Annuler",
    },
  },
  errors: errorsFr,
  errorFallback: "Une erreur est survenue ({code}). Réessayez.",
};
