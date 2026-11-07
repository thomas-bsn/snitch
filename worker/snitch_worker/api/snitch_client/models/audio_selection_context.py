from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_plan_input import AudioPlanInput
  from ..models.audio_selection_diagnostic import AudioSelectionDiagnostic
  from ..models.audio_track_inventory import AudioTrackInventory





T = TypeVar("T", bound="AudioSelectionContext")



@_attrs_define
class AudioSelectionContext:
    """
        Attributes:
            enabled (bool):
            version (str):
            state_version (str):
            inventory (list[AudioTrackInventory]): Pistes audio réelles du clip, avec leur indice ffprobe ; vide quand la
                sélection est désactivée.
            diagnostics (list[AudioSelectionDiagnostic]):
            input_ (AudioPlanInput | Unset):
     """

    enabled: bool
    version: str
    state_version: str
    inventory: list[AudioTrackInventory]
    diagnostics: list[AudioSelectionDiagnostic]
    input_: AudioPlanInput | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_plan_input import AudioPlanInput # noqa: PLC0415
        from ..models.audio_selection_diagnostic import AudioSelectionDiagnostic # noqa: PLC0415
        from ..models.audio_track_inventory import AudioTrackInventory # noqa: PLC0415
        enabled = self.enabled

        version = self.version

        state_version = self.state_version

        inventory = []
        for inventory_item_data in self.inventory:
            inventory_item = inventory_item_data.to_dict()
            inventory.append(inventory_item)



        diagnostics = []
        for diagnostics_item_data in self.diagnostics:
            diagnostics_item = diagnostics_item_data.to_dict()
            diagnostics.append(diagnostics_item)



        input_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.input_, Unset):
            input_ = self.input_.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "version": version,
            "state_version": state_version,
            "inventory": inventory,
            "diagnostics": diagnostics,
        })
        if input_ is not UNSET:
            field_dict["input"] = input_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_plan_input import AudioPlanInput # noqa: PLC0415
        from ..models.audio_selection_diagnostic import AudioSelectionDiagnostic # noqa: PLC0415
        from ..models.audio_track_inventory import AudioTrackInventory # noqa: PLC0415
        d = dict(src_dict)
        enabled = d.pop("enabled")

        version = d.pop("version")

        state_version = d.pop("state_version")

        inventory = []
        _inventory = d.pop("inventory")
        for inventory_item_data in (_inventory):
            inventory_item = AudioTrackInventory.from_dict(inventory_item_data)



            inventory.append(inventory_item)


        diagnostics = []
        _diagnostics = d.pop("diagnostics")
        for diagnostics_item_data in (_diagnostics):
            diagnostics_item = AudioSelectionDiagnostic.from_dict(diagnostics_item_data)



            diagnostics.append(diagnostics_item)


        _input_ = d.pop("input", UNSET)
        input_: AudioPlanInput | Unset
        if isinstance(_input_,  Unset):
            input_ = UNSET
        else:
            input_ = AudioPlanInput.from_dict(_input_)




        audio_selection_context = cls(
            enabled=enabled,
            version=version,
            state_version=state_version,
            inventory=inventory,
            diagnostics=diagnostics,
            input_=input_,
        )

        return audio_selection_context
