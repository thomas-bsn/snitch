import { useI18n } from "./i18n";
import {
  durationText,
  scoreText,
  secondsText,
  videoCountText,
} from "./numberFormats";

/** The number formats of numberFormats.ts, in the language of the interface. */
export function useNumberFormats() {
  const { t, language } = useI18n();
  return {
    videoCount: (count: number) => videoCountText(count, language, t),
    duration: durationText,
    seconds: (value: number) => secondsText(value, language),
    score: (value: number) => scoreText(value, language),
  };
}
