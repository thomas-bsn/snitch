from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.get_clip_analysis_deletion_preview_response_200 import GetClipAnalysisDeletionPreviewResponse200
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    id: int,
    *,
    steps: str | Unset = UNSET,
    all_: bool | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["steps"] = steps

    params["all"] = all_


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/clips/{id}/analysis/deletion-preview".format(id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | GetClipAnalysisDeletionPreviewResponse200:
    if response.status_code == 200:
        response_200 = GetClipAnalysisDeletionPreviewResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | GetClipAnalysisDeletionPreviewResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    steps: str | Unset = UNSET,
    all_: bool | Unset = UNSET,

) -> Response[Error | GetClipAnalysisDeletionPreviewResponse200]:
    """  Administrateurs. Calcul préalable des pertes d'une suppression d'analyse (segments, corrections,
    voix, voix validées, échantillons, Personnes touchées).

    Args:
        id (int):
        steps (str | Unset):
        all_ (bool | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetClipAnalysisDeletionPreviewResponse200]
     """


    kwargs = _get_kwargs(
        id=id,
steps=steps,
all_=all_,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    steps: str | Unset = UNSET,
    all_: bool | Unset = UNSET,

) -> Error | GetClipAnalysisDeletionPreviewResponse200 | None:
    """  Administrateurs. Calcul préalable des pertes d'une suppression d'analyse (segments, corrections,
    voix, voix validées, échantillons, Personnes touchées).

    Args:
        id (int):
        steps (str | Unset):
        all_ (bool | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetClipAnalysisDeletionPreviewResponse200
     """


    return sync_detailed(
        id=id,
client=client,
steps=steps,
all_=all_,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    steps: str | Unset = UNSET,
    all_: bool | Unset = UNSET,

) -> Response[Error | GetClipAnalysisDeletionPreviewResponse200]:
    """  Administrateurs. Calcul préalable des pertes d'une suppression d'analyse (segments, corrections,
    voix, voix validées, échantillons, Personnes touchées).

    Args:
        id (int):
        steps (str | Unset):
        all_ (bool | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetClipAnalysisDeletionPreviewResponse200]
     """


    kwargs = _get_kwargs(
        id=id,
steps=steps,
all_=all_,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    steps: str | Unset = UNSET,
    all_: bool | Unset = UNSET,

) -> Error | GetClipAnalysisDeletionPreviewResponse200 | None:
    """  Administrateurs. Calcul préalable des pertes d'une suppression d'analyse (segments, corrections,
    voix, voix validées, échantillons, Personnes touchées).

    Args:
        id (int):
        steps (str | Unset):
        all_ (bool | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetClipAnalysisDeletionPreviewResponse200
     """


    return (await asyncio_detailed(
        id=id,
client=client,
steps=steps,
all_=all_,

    )).parsed
