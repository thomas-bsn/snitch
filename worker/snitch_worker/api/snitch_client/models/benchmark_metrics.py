from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_separation_metrics import BenchmarkSeparationMetrics
  from ..models.benchmark_word_rates import BenchmarkWordRates





T = TypeVar("T", bound="BenchmarkMetrics")



@_attrs_define
class BenchmarkMetrics:
    """ Mesures d'un seul modèle ; les champs dépendent de l'étape (empreinte : eer, top1, précision, rappel… ;
    transcription : wer, cer… ; diarisation : der et ses trois composantes). Jamais comparables en score brut d'un
    modèle à l'autre. Bout en bout : cpwer (erreurs de mots avec attribution des locuteurs), wer (sans attribution) et
    der des mêmes sorties.

        Attributes:
            references (int | Unset):
            tests (int | Unset):
            persons (int | Unset):
            genuine_trials (int | Unset):
            impostor_trials (int | Unset):
            eer (float | None | Unset):
            eer_threshold (float | None | Unset):
            top1_accuracy (float | None | Unset):
            true_positives (int | Unset):
            false_positives (int | Unset):
            false_negatives (int | Unset):
            precision (float | None | Unset):
            recall (float | None | Unset):
            genuine_mean_score (float | None | Unset):
            impostor_mean_score (float | None | Unset):
            ignored_dimension (int | Unset):
            ineligible (int | Unset): Empreinte : empreintes écartées par la règle d'éligibilité du run (référence non
                contributive, ou non comparable en mode meilleur échantillon ; test non comparable).
            clips (int | Unset): Transcription : clips (et pistes) mesurés.
            reference_words (int | Unset):
            word_errors (int | Unset):
            reference_chars (int | Unset):
            character_errors (int | Unset):
            cer_skipped_clips (int | Unset): Clips trop longs pour la comparaison par caractère, laissés hors du CER.
            wer (float | None | Unset): Word Error Rate (transcription), micro-moyenne sur les clips.
            cer (float | None | Unset): Character Error Rate (transcription).
            skipped_clips (int | Unset): Diarisation : clips trop longs, laissés hors du calcul.
            reference_speech_seconds (float | Unset):
            missed_speech_seconds (float | Unset):
            false_alarm_seconds (float | Unset):
            confusion_seconds (float | Unset):
            collar_seconds (float | Unset): Tolérance autour de chaque borne de référence.
            der (float | None | Unset): Diarization Error Rate : parole manquée + fausse alarme + confusion, rapportées à la
                parole de référence.
            missed_speech_rate (float | None | Unset):
            false_alarm_rate (float | None | Unset):
            confusion_rate (float | None | Unset):
            cpwer (float | None | Unset): Bout en bout : WER avec attribution des locuteurs (cpWER), règle d'attribution par
                segment (ADR 0065).
            word_rule (BenchmarkWordRates | Unset):
            cp_word_errors (int | Unset):
            reference_speakers (int | Unset):
            model_speakers (int | Unset):
            separation (BenchmarkSeparationMetrics | Unset): Bout en bout, chaîne avec un modèle de séparation (#402, #403)
                : comptes des tâches de séparation, et cpWER et WER sans et avec séparation sur les mêmes clips, ceux dont la
                référence contient un Chevauchement.
     """

    references: int | Unset = UNSET
    tests: int | Unset = UNSET
    persons: int | Unset = UNSET
    genuine_trials: int | Unset = UNSET
    impostor_trials: int | Unset = UNSET
    eer: float | None | Unset = UNSET
    eer_threshold: float | None | Unset = UNSET
    top1_accuracy: float | None | Unset = UNSET
    true_positives: int | Unset = UNSET
    false_positives: int | Unset = UNSET
    false_negatives: int | Unset = UNSET
    precision: float | None | Unset = UNSET
    recall: float | None | Unset = UNSET
    genuine_mean_score: float | None | Unset = UNSET
    impostor_mean_score: float | None | Unset = UNSET
    ignored_dimension: int | Unset = UNSET
    ineligible: int | Unset = UNSET
    clips: int | Unset = UNSET
    reference_words: int | Unset = UNSET
    word_errors: int | Unset = UNSET
    reference_chars: int | Unset = UNSET
    character_errors: int | Unset = UNSET
    cer_skipped_clips: int | Unset = UNSET
    wer: float | None | Unset = UNSET
    cer: float | None | Unset = UNSET
    skipped_clips: int | Unset = UNSET
    reference_speech_seconds: float | Unset = UNSET
    missed_speech_seconds: float | Unset = UNSET
    false_alarm_seconds: float | Unset = UNSET
    confusion_seconds: float | Unset = UNSET
    collar_seconds: float | Unset = UNSET
    der: float | None | Unset = UNSET
    missed_speech_rate: float | None | Unset = UNSET
    false_alarm_rate: float | None | Unset = UNSET
    confusion_rate: float | None | Unset = UNSET
    cpwer: float | None | Unset = UNSET
    word_rule: BenchmarkWordRates | Unset = UNSET
    cp_word_errors: int | Unset = UNSET
    reference_speakers: int | Unset = UNSET
    model_speakers: int | Unset = UNSET
    separation: BenchmarkSeparationMetrics | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_separation_metrics import BenchmarkSeparationMetrics # noqa: PLC0415
        from ..models.benchmark_word_rates import BenchmarkWordRates # noqa: PLC0415
        references = self.references

        tests = self.tests

        persons = self.persons

        genuine_trials = self.genuine_trials

        impostor_trials = self.impostor_trials

        eer: float | None | Unset
        if isinstance(self.eer, Unset):
            eer = UNSET
        else:
            eer = self.eer

        eer_threshold: float | None | Unset
        if isinstance(self.eer_threshold, Unset):
            eer_threshold = UNSET
        else:
            eer_threshold = self.eer_threshold

        top1_accuracy: float | None | Unset
        if isinstance(self.top1_accuracy, Unset):
            top1_accuracy = UNSET
        else:
            top1_accuracy = self.top1_accuracy

        true_positives = self.true_positives

        false_positives = self.false_positives

        false_negatives = self.false_negatives

        precision: float | None | Unset
        if isinstance(self.precision, Unset):
            precision = UNSET
        else:
            precision = self.precision

        recall: float | None | Unset
        if isinstance(self.recall, Unset):
            recall = UNSET
        else:
            recall = self.recall

        genuine_mean_score: float | None | Unset
        if isinstance(self.genuine_mean_score, Unset):
            genuine_mean_score = UNSET
        else:
            genuine_mean_score = self.genuine_mean_score

        impostor_mean_score: float | None | Unset
        if isinstance(self.impostor_mean_score, Unset):
            impostor_mean_score = UNSET
        else:
            impostor_mean_score = self.impostor_mean_score

        ignored_dimension = self.ignored_dimension

        ineligible = self.ineligible

        clips = self.clips

        reference_words = self.reference_words

        word_errors = self.word_errors

        reference_chars = self.reference_chars

        character_errors = self.character_errors

        cer_skipped_clips = self.cer_skipped_clips

        wer: float | None | Unset
        if isinstance(self.wer, Unset):
            wer = UNSET
        else:
            wer = self.wer

        cer: float | None | Unset
        if isinstance(self.cer, Unset):
            cer = UNSET
        else:
            cer = self.cer

        skipped_clips = self.skipped_clips

        reference_speech_seconds = self.reference_speech_seconds

        missed_speech_seconds = self.missed_speech_seconds

        false_alarm_seconds = self.false_alarm_seconds

        confusion_seconds = self.confusion_seconds

        collar_seconds = self.collar_seconds

        der: float | None | Unset
        if isinstance(self.der, Unset):
            der = UNSET
        else:
            der = self.der

        missed_speech_rate: float | None | Unset
        if isinstance(self.missed_speech_rate, Unset):
            missed_speech_rate = UNSET
        else:
            missed_speech_rate = self.missed_speech_rate

        false_alarm_rate: float | None | Unset
        if isinstance(self.false_alarm_rate, Unset):
            false_alarm_rate = UNSET
        else:
            false_alarm_rate = self.false_alarm_rate

        confusion_rate: float | None | Unset
        if isinstance(self.confusion_rate, Unset):
            confusion_rate = UNSET
        else:
            confusion_rate = self.confusion_rate

        cpwer: float | None | Unset
        if isinstance(self.cpwer, Unset):
            cpwer = UNSET
        else:
            cpwer = self.cpwer

        word_rule: dict[str, Any] | Unset = UNSET
        if not isinstance(self.word_rule, Unset):
            word_rule = self.word_rule.to_dict()

        cp_word_errors = self.cp_word_errors

        reference_speakers = self.reference_speakers

        model_speakers = self.model_speakers

        separation: dict[str, Any] | Unset = UNSET
        if not isinstance(self.separation, Unset):
            separation = self.separation.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if references is not UNSET:
            field_dict["references"] = references
        if tests is not UNSET:
            field_dict["tests"] = tests
        if persons is not UNSET:
            field_dict["persons"] = persons
        if genuine_trials is not UNSET:
            field_dict["genuine_trials"] = genuine_trials
        if impostor_trials is not UNSET:
            field_dict["impostor_trials"] = impostor_trials
        if eer is not UNSET:
            field_dict["eer"] = eer
        if eer_threshold is not UNSET:
            field_dict["eer_threshold"] = eer_threshold
        if top1_accuracy is not UNSET:
            field_dict["top1_accuracy"] = top1_accuracy
        if true_positives is not UNSET:
            field_dict["true_positives"] = true_positives
        if false_positives is not UNSET:
            field_dict["false_positives"] = false_positives
        if false_negatives is not UNSET:
            field_dict["false_negatives"] = false_negatives
        if precision is not UNSET:
            field_dict["precision"] = precision
        if recall is not UNSET:
            field_dict["recall"] = recall
        if genuine_mean_score is not UNSET:
            field_dict["genuine_mean_score"] = genuine_mean_score
        if impostor_mean_score is not UNSET:
            field_dict["impostor_mean_score"] = impostor_mean_score
        if ignored_dimension is not UNSET:
            field_dict["ignored_dimension"] = ignored_dimension
        if ineligible is not UNSET:
            field_dict["ineligible"] = ineligible
        if clips is not UNSET:
            field_dict["clips"] = clips
        if reference_words is not UNSET:
            field_dict["reference_words"] = reference_words
        if word_errors is not UNSET:
            field_dict["word_errors"] = word_errors
        if reference_chars is not UNSET:
            field_dict["reference_chars"] = reference_chars
        if character_errors is not UNSET:
            field_dict["character_errors"] = character_errors
        if cer_skipped_clips is not UNSET:
            field_dict["cer_skipped_clips"] = cer_skipped_clips
        if wer is not UNSET:
            field_dict["wer"] = wer
        if cer is not UNSET:
            field_dict["cer"] = cer
        if skipped_clips is not UNSET:
            field_dict["skipped_clips"] = skipped_clips
        if reference_speech_seconds is not UNSET:
            field_dict["reference_speech_seconds"] = reference_speech_seconds
        if missed_speech_seconds is not UNSET:
            field_dict["missed_speech_seconds"] = missed_speech_seconds
        if false_alarm_seconds is not UNSET:
            field_dict["false_alarm_seconds"] = false_alarm_seconds
        if confusion_seconds is not UNSET:
            field_dict["confusion_seconds"] = confusion_seconds
        if collar_seconds is not UNSET:
            field_dict["collar_seconds"] = collar_seconds
        if der is not UNSET:
            field_dict["der"] = der
        if missed_speech_rate is not UNSET:
            field_dict["missed_speech_rate"] = missed_speech_rate
        if false_alarm_rate is not UNSET:
            field_dict["false_alarm_rate"] = false_alarm_rate
        if confusion_rate is not UNSET:
            field_dict["confusion_rate"] = confusion_rate
        if cpwer is not UNSET:
            field_dict["cpwer"] = cpwer
        if word_rule is not UNSET:
            field_dict["word_rule"] = word_rule
        if cp_word_errors is not UNSET:
            field_dict["cp_word_errors"] = cp_word_errors
        if reference_speakers is not UNSET:
            field_dict["reference_speakers"] = reference_speakers
        if model_speakers is not UNSET:
            field_dict["model_speakers"] = model_speakers
        if separation is not UNSET:
            field_dict["separation"] = separation

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_separation_metrics import BenchmarkSeparationMetrics # noqa: PLC0415
        from ..models.benchmark_word_rates import BenchmarkWordRates # noqa: PLC0415
        d = dict(src_dict)
        references = d.pop("references", UNSET)

        tests = d.pop("tests", UNSET)

        persons = d.pop("persons", UNSET)

        genuine_trials = d.pop("genuine_trials", UNSET)

        impostor_trials = d.pop("impostor_trials", UNSET)

        def _parse_eer(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        eer = _parse_eer(d.pop("eer", UNSET))


        def _parse_eer_threshold(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        eer_threshold = _parse_eer_threshold(d.pop("eer_threshold", UNSET))


        def _parse_top1_accuracy(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        top1_accuracy = _parse_top1_accuracy(d.pop("top1_accuracy", UNSET))


        true_positives = d.pop("true_positives", UNSET)

        false_positives = d.pop("false_positives", UNSET)

        false_negatives = d.pop("false_negatives", UNSET)

        def _parse_precision(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        precision = _parse_precision(d.pop("precision", UNSET))


        def _parse_recall(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        recall = _parse_recall(d.pop("recall", UNSET))


        def _parse_genuine_mean_score(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        genuine_mean_score = _parse_genuine_mean_score(d.pop("genuine_mean_score", UNSET))


        def _parse_impostor_mean_score(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        impostor_mean_score = _parse_impostor_mean_score(d.pop("impostor_mean_score", UNSET))


        ignored_dimension = d.pop("ignored_dimension", UNSET)

        ineligible = d.pop("ineligible", UNSET)

        clips = d.pop("clips", UNSET)

        reference_words = d.pop("reference_words", UNSET)

        word_errors = d.pop("word_errors", UNSET)

        reference_chars = d.pop("reference_chars", UNSET)

        character_errors = d.pop("character_errors", UNSET)

        cer_skipped_clips = d.pop("cer_skipped_clips", UNSET)

        def _parse_wer(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        wer = _parse_wer(d.pop("wer", UNSET))


        def _parse_cer(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        cer = _parse_cer(d.pop("cer", UNSET))


        skipped_clips = d.pop("skipped_clips", UNSET)

        reference_speech_seconds = d.pop("reference_speech_seconds", UNSET)

        missed_speech_seconds = d.pop("missed_speech_seconds", UNSET)

        false_alarm_seconds = d.pop("false_alarm_seconds", UNSET)

        confusion_seconds = d.pop("confusion_seconds", UNSET)

        collar_seconds = d.pop("collar_seconds", UNSET)

        def _parse_der(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        der = _parse_der(d.pop("der", UNSET))


        def _parse_missed_speech_rate(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        missed_speech_rate = _parse_missed_speech_rate(d.pop("missed_speech_rate", UNSET))


        def _parse_false_alarm_rate(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        false_alarm_rate = _parse_false_alarm_rate(d.pop("false_alarm_rate", UNSET))


        def _parse_confusion_rate(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        confusion_rate = _parse_confusion_rate(d.pop("confusion_rate", UNSET))


        def _parse_cpwer(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        cpwer = _parse_cpwer(d.pop("cpwer", UNSET))


        _word_rule = d.pop("word_rule", UNSET)
        word_rule: BenchmarkWordRates | Unset
        if isinstance(_word_rule,  Unset):
            word_rule = UNSET
        else:
            word_rule = BenchmarkWordRates.from_dict(_word_rule)




        cp_word_errors = d.pop("cp_word_errors", UNSET)

        reference_speakers = d.pop("reference_speakers", UNSET)

        model_speakers = d.pop("model_speakers", UNSET)

        _separation = d.pop("separation", UNSET)
        separation: BenchmarkSeparationMetrics | Unset
        if isinstance(_separation,  Unset):
            separation = UNSET
        else:
            separation = BenchmarkSeparationMetrics.from_dict(_separation)




        benchmark_metrics = cls(
            references=references,
            tests=tests,
            persons=persons,
            genuine_trials=genuine_trials,
            impostor_trials=impostor_trials,
            eer=eer,
            eer_threshold=eer_threshold,
            top1_accuracy=top1_accuracy,
            true_positives=true_positives,
            false_positives=false_positives,
            false_negatives=false_negatives,
            precision=precision,
            recall=recall,
            genuine_mean_score=genuine_mean_score,
            impostor_mean_score=impostor_mean_score,
            ignored_dimension=ignored_dimension,
            ineligible=ineligible,
            clips=clips,
            reference_words=reference_words,
            word_errors=word_errors,
            reference_chars=reference_chars,
            character_errors=character_errors,
            cer_skipped_clips=cer_skipped_clips,
            wer=wer,
            cer=cer,
            skipped_clips=skipped_clips,
            reference_speech_seconds=reference_speech_seconds,
            missed_speech_seconds=missed_speech_seconds,
            false_alarm_seconds=false_alarm_seconds,
            confusion_seconds=confusion_seconds,
            collar_seconds=collar_seconds,
            der=der,
            missed_speech_rate=missed_speech_rate,
            false_alarm_rate=false_alarm_rate,
            confusion_rate=confusion_rate,
            cpwer=cpwer,
            word_rule=word_rule,
            cp_word_errors=cp_word_errors,
            reference_speakers=reference_speakers,
            model_speakers=model_speakers,
            separation=separation,
        )


        benchmark_metrics.additional_properties = d
        return benchmark_metrics

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
