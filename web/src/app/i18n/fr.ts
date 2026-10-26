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
    discordLinked: "Votre identité Discord est liée à votre compte.",
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
  account: {
    tabsLabel: "Paramètres du compte",
    tabs: { profile: "Profil", voice: "Ma voix" },
    profile: {
      identityTitle: "Compte",
      username: "Nom d’utilisateur",
      role: "Rôle",
      discordTitle: "Discord",
      discordBody:
        "Liez votre identité Discord à ce compte pour pouvoir vous connecter avec Discord. Le lien se fait avec ce compte uniquement, jamais par correspondance de pseudo. Cette action ouvre Discord.",
      discordLink: "Lier Discord",
      discordLinking: "Ouverture de Discord…",
      discordDisabled:
        "La connexion Discord n’est pas activée sur cette instance. Un administrateur peut l’activer.",
      preferencesTitle: "Préférences",
      language: "Langue",
      languageHint:
        "Conservée dans votre compte : elle vous suit sur vos autres appareils.",
      languageSaved: "Langue enregistrée.",
    },
    voice: {
      title: "Ma voix",
      loading: "Chargement de votre voix…",
      noPersonTitle: "Pas de profil vocal",
      noPerson:
        "Aucune Personne n’est liée à votre compte : Snitch ne détient donc aucun profil vocal à votre nom. Un administrateur lie une Personne à un compte.",
      identity: "Vous êtes la Personne {name}.",
      samples:
        "{count} échantillon(s) vocal(aux) validé(s) servent de référence pour votre voix.",
      deleted:
        "Votre profil vocal est supprimé : Snitch ne vous reconnaît plus.",
      clipsTitle: "Clips où vous êtes identifié(e)",
      clipsLabel: "Mes clips",
      noClips: "Aucun clip visible pour vous.",
      deleteTitle: "Supprimer mon profil vocal",
      deleteEffects:
        "Supprimer votre profil vocal efface vos échantillons vocaux validés, vos empreintes vocales et toutes les attributions faites à votre nom. Vos voix redeviennent inconnues et plus personne ne peut vous attribuer une voix ensuite. Vos clips restent en place. Cette action est définitive.",
      deleteButton: "Supprimer mon profil vocal",
      confirmTitle: "Supprimer votre profil vocal ?",
      confirmBody:
        "Vos échantillons, vos empreintes et vos attributions seront effacés, et vos voix redeviendront inconnues. Vos clips restent en place. Cette action est définitive.",
      confirm: "Supprimer mon profil vocal",
      deleting: "Suppression…",
      keep: "Garder mon profil",
      done: "Profil vocal supprimé : {samples} échantillon(s), {fingerprints} empreinte(s) et {attributions} attribution(s) effacé(s).",
    },
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
