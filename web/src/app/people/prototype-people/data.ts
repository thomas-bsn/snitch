// PROTOTYPE — onglet People refondu. Données factices, jamais fusionné dans main.

export type TodoVideo = {
  id: number;
  title: string;
  folder: string;
  date: string;
  duration: string;
  suggestions: number; // « C'est X ? » à confirmer
  ambiguous: number;
  unknown: number;
  groups: string[]; // groupes inconnus non nommés présents dans la vidéo
  recentlyAnalysed?: boolean;
};

export const todo: TodoVideo[] = [
  { id: 40, title: "Réunion de préparation — tournage S12", folder: "2026/novembre", date: "28 oct.", duration: "38:12", suggestions: 1, ambiguous: 1, unknown: 2, groups: ["#7"], recentlyAnalysed: true },
  { id: 38, title: "Rushes parking — prise 3", folder: "2026/octobre/parking", date: "24 oct.", duration: "12:40", suggestions: 2, ambiguous: 0, unknown: 1, groups: ["#7", "#9"] },
  { id: 35, title: "Interview Claire — making-of", folder: "2026/octobre", date: "19 oct.", duration: "22:05", suggestions: 0, ambiguous: 1, unknown: 0, groups: [] },
  { id: 31, title: "Brief équipe lumière", folder: "2026/octobre", date: "12 oct.", duration: "17:58", suggestions: 3, ambiguous: 0, unknown: 0, groups: [] },
  { id: 27, title: "Visio production — budget", folder: "2026/septembre", date: "30 sept.", duration: "51:20", suggestions: 0, ambiguous: 0, unknown: 3, groups: ["#9", "#11"] },
  { id: 22, title: "Repérages Studio B", folder: "2026/septembre", date: "21 sept.", duration: "09:12", suggestions: 1, ambiguous: 0, unknown: 0, groups: [] },
];

export const total = (v: TodoVideo) => v.suggestions + v.ambiguous + v.unknown;

export type Person = {
  id: number;
  name: string;
  account?: string;
  videos: number;
  speakingMin: number;
  samples: number;
  lastHeard: string;
  referenceQuality: "bonne" | "faible" | "aucune";
};

export const people: Person[] = [
  { id: 1, name: "Marc Lefèvre", account: "marc", videos: 34, speakingMin: 412, samples: 18, lastHeard: "28 oct.", referenceQuality: "bonne" },
  { id: 2, name: "Julie Martin", account: "julie", videos: 29, speakingMin: 305, samples: 14, lastHeard: "28 oct.", referenceQuality: "bonne" },
  { id: 3, name: "Claire Dubois", videos: 11, speakingMin: 96, samples: 5, lastHeard: "19 oct.", referenceQuality: "bonne" },
  { id: 4, name: "Thomas Roux", account: "thomas", videos: 8, speakingMin: 41, samples: 2, lastHeard: "24 oct.", referenceQuality: "faible" },
  { id: 5, name: "Hugo Bernard", videos: 3, speakingMin: 12, samples: 1, lastHeard: "30 sept.", referenceQuality: "faible" },
  { id: 6, name: "Inès Garnier", videos: 1, speakingMin: 4, samples: 0, lastHeard: "2 sept.", referenceQuality: "aucune" },
];

export type Group = { id: string; videos: string[]; speakingS: number; guess?: string };

export const groups: Group[] = [
  { id: "#7", videos: ["Réunion de préparation — tournage S12", "Rushes parking — prise 3", "Repérages Studio B"], speakingS: 214, guess: "Hugo Bernard" },
  { id: "#9", videos: ["Rushes parking — prise 3", "Visio production — budget"], speakingS: 380 },
  { id: "#11", videos: ["Visio production — budget", "Brief équipe lumière"], speakingS: 95 },
];

export const appearances = [
  { title: "Réunion de préparation — tournage S12", date: "28 oct.", speaking: "10:11", state: "validée" },
  { title: "Rushes parking — prise 3", date: "24 oct.", speaking: "3:02", state: "suggestion" },
  { title: "Brief équipe lumière", date: "12 oct.", speaking: "8:40", state: "validée" },
  { title: "Visio production — budget", date: "30 sept.", speaking: "21:05", state: "validée" },
];

/** Adresse de la fiche (prototype C) ouverte sur ses voix à traiter. */
export const ficheVoices = (id: number) => `/prototype/fiche-video?variant=C&tab=voices&clip=${id}`;
