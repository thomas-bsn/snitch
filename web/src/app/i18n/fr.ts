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
  errors: errorsFr,
  errorFallback: "Une erreur est survenue ({code}). Réessayez.",
};
