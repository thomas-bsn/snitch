from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.clip_analysis_run import ClipAnalysisRun





T = TypeVar("T", bound="ClipAnalysisResponse")



@_attrs_define
class ClipAnalysisResponse:
    """
        Attributes:
            run (ClipAnalysisRun | None):
     """

    run: ClipAnalysisRun | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_analysis_run import ClipAnalysisRun # noqa: PLC0415
        run: dict[str, Any] | None
        if isinstance(self.run, ClipAnalysisRun):
            run = self.run.to_dict()
        else:
            run = self.run


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "run": run,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_analysis_run import ClipAnalysisRun # noqa: PLC0415
        d = dict(src_dict)
        def _parse_run(data: object) -> ClipAnalysisRun | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                run_type_1 = ClipAnalysisRun.from_dict(data)



                return run_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(ClipAnalysisRun | None, data)

        run = _parse_run(d.pop("run"))


        clip_analysis_response = cls(
            run=run,
        )


        clip_analysis_response.additional_properties = d
        return clip_analysis_response

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
