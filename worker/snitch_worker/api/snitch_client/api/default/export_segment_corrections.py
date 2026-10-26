from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.export_segment_corrections_format import ExportSegmentCorrectionsFormat
from ...models.segment_correction_export import SegmentCorrectionExport
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    format_: ExportSegmentCorrectionsFormat | Unset = ExportSegmentCorrectionsFormat.JSON,
    page: int | Unset = 1,
    page_size: int | Unset = 100,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_format_: str | Unset = UNSET
    if not isinstance(format_, Unset):
        json_format_ = format_.value

    params["format"] = json_format_

    params["page"] = page

    params["page_size"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/segment-corrections/export",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | SegmentCorrectionExport:
    if response.status_code == 200:
        response_200 = SegmentCorrectionExport.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | SegmentCorrectionExport]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    format_: ExportSegmentCorrectionsFormat | Unset = ExportSegmentCorrectionsFormat.JSON,
    page: int | Unset = 1,
    page_size: int | Unset = 100,

) -> Response[Error | SegmentCorrectionExport]:
    """  Administrateurs uniquement. Liste les segments dont le texte est corrigé (jamais un segment non
    corrigé, ni un segment restauré à l'original, ni une correction obsolète d'une ancienne
    transcription), pour tous les clips, avec le texte d'origine du worker, le texte corrigé, l'auteur,
    la date et la version, pour préparer la mesure de qualité de la phase 7. Snitch ne calcule aucun
    taux d'erreur.

    Args:
        format_ (ExportSegmentCorrectionsFormat | Unset):  Default:
            ExportSegmentCorrectionsFormat.JSON.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 100.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SegmentCorrectionExport]
     """


    kwargs = _get_kwargs(
        format_=format_,
page=page,
page_size=page_size,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    format_: ExportSegmentCorrectionsFormat | Unset = ExportSegmentCorrectionsFormat.JSON,
    page: int | Unset = 1,
    page_size: int | Unset = 100,

) -> Error | SegmentCorrectionExport | None:
    """  Administrateurs uniquement. Liste les segments dont le texte est corrigé (jamais un segment non
    corrigé, ni un segment restauré à l'original, ni une correction obsolète d'une ancienne
    transcription), pour tous les clips, avec le texte d'origine du worker, le texte corrigé, l'auteur,
    la date et la version, pour préparer la mesure de qualité de la phase 7. Snitch ne calcule aucun
    taux d'erreur.

    Args:
        format_ (ExportSegmentCorrectionsFormat | Unset):  Default:
            ExportSegmentCorrectionsFormat.JSON.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 100.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SegmentCorrectionExport
     """


    return sync_detailed(
        client=client,
format_=format_,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    format_: ExportSegmentCorrectionsFormat | Unset = ExportSegmentCorrectionsFormat.JSON,
    page: int | Unset = 1,
    page_size: int | Unset = 100,

) -> Response[Error | SegmentCorrectionExport]:
    """  Administrateurs uniquement. Liste les segments dont le texte est corrigé (jamais un segment non
    corrigé, ni un segment restauré à l'original, ni une correction obsolète d'une ancienne
    transcription), pour tous les clips, avec le texte d'origine du worker, le texte corrigé, l'auteur,
    la date et la version, pour préparer la mesure de qualité de la phase 7. Snitch ne calcule aucun
    taux d'erreur.

    Args:
        format_ (ExportSegmentCorrectionsFormat | Unset):  Default:
            ExportSegmentCorrectionsFormat.JSON.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 100.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SegmentCorrectionExport]
     """


    kwargs = _get_kwargs(
        format_=format_,
page=page,
page_size=page_size,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    format_: ExportSegmentCorrectionsFormat | Unset = ExportSegmentCorrectionsFormat.JSON,
    page: int | Unset = 1,
    page_size: int | Unset = 100,

) -> Error | SegmentCorrectionExport | None:
    """  Administrateurs uniquement. Liste les segments dont le texte est corrigé (jamais un segment non
    corrigé, ni un segment restauré à l'original, ni une correction obsolète d'une ancienne
    transcription), pour tous les clips, avec le texte d'origine du worker, le texte corrigé, l'auteur,
    la date et la version, pour préparer la mesure de qualité de la phase 7. Snitch ne calcule aucun
    taux d'erreur.

    Args:
        format_ (ExportSegmentCorrectionsFormat | Unset):  Default:
            ExportSegmentCorrectionsFormat.JSON.
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 100.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SegmentCorrectionExport
     """


    return (await asyncio_detailed(
        client=client,
format_=format_,
page=page,
page_size=page_size,

    )).parsed
