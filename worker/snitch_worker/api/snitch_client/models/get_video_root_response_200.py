from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.source import Source





T = TypeVar("T", bound="GetVideoRootResponse200")



@_attrs_define
class GetVideoRootResponse200:
    """
        Attributes:
            root (None | Source):
            worker_available (bool): Un Worker connecté peut prendre les fichiers demandés. Sinon ils restent en attente.
     """

    root: None | Source
    worker_available: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.source import Source # noqa: PLC0415
        root: dict[str, Any] | None
        if isinstance(self.root, Source):
            root = self.root.to_dict()
        else:
            root = self.root

        worker_available = self.worker_available


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "root": root,
            "worker_available": worker_available,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.source import Source # noqa: PLC0415
        d = dict(src_dict)
        def _parse_root(data: object) -> None | Source:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                root_type_1 = Source.from_dict(data)



                return root_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Source, data)

        root = _parse_root(d.pop("root"))


        worker_available = d.pop("worker_available")

        get_video_root_response_200 = cls(
            root=root,
            worker_available=worker_available,
        )

        return get_video_root_response_200
