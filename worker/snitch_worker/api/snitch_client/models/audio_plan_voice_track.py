from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_plan_voice_track_status import AudioPlanVoiceTrackStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_plan_voice_stake import AudioPlanVoiceStake
  from ..models.voice_step import VoiceStep





T = TypeVar("T", bound="AudioPlanVoiceTrack")



@_attrs_define
class AudioPlanVoiceTrack:
    """
        Attributes:
            audio_stream_index (int):
            status (AudioPlanVoiceTrackStatus):
            cause (str): voice_worker_unavailable (en attente d'un Worker vocal prêt), microphone_direct_attribution,
                no_speakers, cancelled_by_admin, fingerprint_missing ou le code expurgé de l'étape en échec ; vide sinon.
            reused (bool): Vrai si la diarisation déjà faite avec le même modèle a été conservée : seuls les travaux
                manquants sont repris.
            microphone (bool): Piste micro désignée : attribution directe, jamais diarisée, sans extension aux autres
                pistes.
            diarization (None | VoiceStep):
            fingerprint (None | VoiceStep):
            speaker_count (int):
            model (str):
            model_version (str): Version épinglée au premier lancement du Plan.
            stake (AudioPlanVoiceStake): Travail humain qu'une diarisation des pistes concernées rendrait obsolète. Les
                anciennes voix restent protégées jusqu'à un nouveau résultat réussi.
     """

    audio_stream_index: int
    status: AudioPlanVoiceTrackStatus
    cause: str
    reused: bool
    microphone: bool
    diarization: None | VoiceStep
    fingerprint: None | VoiceStep
    speaker_count: int
    model: str
    model_version: str
    stake: AudioPlanVoiceStake
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_plan_voice_stake import AudioPlanVoiceStake # noqa: PLC0415
        from ..models.voice_step import VoiceStep # noqa: PLC0415
        audio_stream_index = self.audio_stream_index

        status = self.status.value

        cause = self.cause

        reused = self.reused

        microphone = self.microphone

        diarization: dict[str, Any] | None
        if isinstance(self.diarization, VoiceStep):
            diarization = self.diarization.to_dict()
        else:
            diarization = self.diarization

        fingerprint: dict[str, Any] | None
        if isinstance(self.fingerprint, VoiceStep):
            fingerprint = self.fingerprint.to_dict()
        else:
            fingerprint = self.fingerprint

        speaker_count = self.speaker_count

        model = self.model

        model_version = self.model_version

        stake = self.stake.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "audio_stream_index": audio_stream_index,
            "status": status,
            "cause": cause,
            "reused": reused,
            "microphone": microphone,
            "diarization": diarization,
            "fingerprint": fingerprint,
            "speaker_count": speaker_count,
            "model": model,
            "model_version": model_version,
            "stake": stake,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_plan_voice_stake import AudioPlanVoiceStake # noqa: PLC0415
        from ..models.voice_step import VoiceStep # noqa: PLC0415
        d = dict(src_dict)
        audio_stream_index = d.pop("audio_stream_index")

        status = AudioPlanVoiceTrackStatus(d.pop("status"))




        cause = d.pop("cause")

        reused = d.pop("reused")

        microphone = d.pop("microphone")

        def _parse_diarization(data: object) -> None | VoiceStep:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                diarization_type_1 = VoiceStep.from_dict(data)



                return diarization_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | VoiceStep, data)

        diarization = _parse_diarization(d.pop("diarization"))


        def _parse_fingerprint(data: object) -> None | VoiceStep:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                fingerprint_type_1 = VoiceStep.from_dict(data)



                return fingerprint_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | VoiceStep, data)

        fingerprint = _parse_fingerprint(d.pop("fingerprint"))


        speaker_count = d.pop("speaker_count")

        model = d.pop("model")

        model_version = d.pop("model_version")

        stake = AudioPlanVoiceStake.from_dict(d.pop("stake"))




        audio_plan_voice_track = cls(
            audio_stream_index=audio_stream_index,
            status=status,
            cause=cause,
            reused=reused,
            microphone=microphone,
            diarization=diarization,
            fingerprint=fingerprint,
            speaker_count=speaker_count,
            model=model,
            model_version=model_version,
            stake=stake,
        )


        audio_plan_voice_track.additional_properties = d
        return audio_plan_voice_track

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
