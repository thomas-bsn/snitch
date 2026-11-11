from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_calibration_active_mode import VoiceCalibrationActiveMode
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_calibration_mode import VoiceCalibrationMode
  from ..models.voice_fingerprint_group import VoiceFingerprintGroup
  from ..models.voice_group_thresholds import VoiceGroupThresholds





T = TypeVar("T", bound="VoiceCalibration")



@_attrs_define
class VoiceCalibration:
    """ Calibration des seuils du groupe d'empreintes actif (ADR 0064), les deux modes de score côte à côte.

        Attributes:
            group (None | VoiceFingerprintGroup): Groupe actif ; null tant qu'aucune empreinte n'a été produite.
            thresholds (None | VoiceGroupThresholds): Seuils appliqués au groupe pour le mode de score actif ; null :
                suggestions seulement, aucune Attribution automatique.
            history (list[VoiceGroupThresholds]): Vingt dernières versions, la plus récente d'abord.
            active_mode (VoiceCalibrationActiveMode):
            min_tests (int):
            min_persons (int):
            modes (list[VoiceCalibrationMode]):
     """

    group: None | VoiceFingerprintGroup
    thresholds: None | VoiceGroupThresholds
    history: list[VoiceGroupThresholds]
    active_mode: VoiceCalibrationActiveMode
    min_tests: int
    min_persons: int
    modes: list[VoiceCalibrationMode]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_calibration_mode import VoiceCalibrationMode # noqa: PLC0415
        from ..models.voice_fingerprint_group import VoiceFingerprintGroup # noqa: PLC0415
        from ..models.voice_group_thresholds import VoiceGroupThresholds # noqa: PLC0415
        group: dict[str, Any] | None
        if isinstance(self.group, VoiceFingerprintGroup):
            group = self.group.to_dict()
        else:
            group = self.group

        thresholds: dict[str, Any] | None
        if isinstance(self.thresholds, VoiceGroupThresholds):
            thresholds = self.thresholds.to_dict()
        else:
            thresholds = self.thresholds

        history = []
        for history_item_data in self.history:
            history_item = history_item_data.to_dict()
            history.append(history_item)



        active_mode = self.active_mode.value

        min_tests = self.min_tests

        min_persons = self.min_persons

        modes = []
        for modes_item_data in self.modes:
            modes_item = modes_item_data.to_dict()
            modes.append(modes_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "group": group,
            "thresholds": thresholds,
            "history": history,
            "active_mode": active_mode,
            "min_tests": min_tests,
            "min_persons": min_persons,
            "modes": modes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_calibration_mode import VoiceCalibrationMode # noqa: PLC0415
        from ..models.voice_fingerprint_group import VoiceFingerprintGroup # noqa: PLC0415
        from ..models.voice_group_thresholds import VoiceGroupThresholds # noqa: PLC0415
        d = dict(src_dict)
        def _parse_group(data: object) -> None | VoiceFingerprintGroup:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                group_type_1 = VoiceFingerprintGroup.from_dict(data)



                return group_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | VoiceFingerprintGroup, data)

        group = _parse_group(d.pop("group"))


        def _parse_thresholds(data: object) -> None | VoiceGroupThresholds:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                thresholds_type_1 = VoiceGroupThresholds.from_dict(data)



                return thresholds_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | VoiceGroupThresholds, data)

        thresholds = _parse_thresholds(d.pop("thresholds"))


        history = []
        _history = d.pop("history")
        for history_item_data in (_history):
            history_item = VoiceGroupThresholds.from_dict(history_item_data)



            history.append(history_item)


        active_mode = VoiceCalibrationActiveMode(d.pop("active_mode"))




        min_tests = d.pop("min_tests")

        min_persons = d.pop("min_persons")

        modes = []
        _modes = d.pop("modes")
        for modes_item_data in (_modes):
            modes_item = VoiceCalibrationMode.from_dict(modes_item_data)



            modes.append(modes_item)


        voice_calibration = cls(
            group=group,
            thresholds=thresholds,
            history=history,
            active_mode=active_mode,
            min_tests=min_tests,
            min_persons=min_persons,
            modes=modes,
        )


        voice_calibration.additional_properties = d
        return voice_calibration

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
