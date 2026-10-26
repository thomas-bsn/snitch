from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.segment_conflict import SegmentConflict
from ...models.segment_correction import SegmentCorrection
from ...models.segment_restore_input import SegmentRestoreInput
from typing import cast



def _get_kwargs(
    id: int,
    index: int,
    *,
    body: SegmentRestoreInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/clips/{id}/segments/{index}/restore".format(id=quote(str(id), safe=""),index=quote(str(index), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | SegmentConflict | SegmentCorrection:
    if response.status_code == 200:
        response_200 = SegmentCorrection.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = SegmentConflict.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | SegmentConflict | SegmentCorrection]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,
    body: SegmentRestoreInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | SegmentConflict | SegmentCorrection]:
    """  Utilisateurs connectés, pour un clip visible. Restaure le texte d'origine (original=true) ou le
    texte produit par une entrée antérieure de ce segment (event_id) ; exactement l'un des deux. La
    restauration est une correction : version incrémentée, segment à réindexer, nouvelle entrée
    d'historique de type restore ; aucune entrée passée n'est modifiée. Une entrée de locuteur ne se
    restaure pas ici.

    Args:
        id (int):
        index (int):
        origin (str):
        x_csrf_token (str):
        body (SegmentRestoreInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SegmentConflict | SegmentCorrection]
     """


    kwargs = _get_kwargs(
        id=id,
index=index,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,
    body: SegmentRestoreInput,
    origin: str,
    x_csrf_token: str,

) -> Error | SegmentConflict | SegmentCorrection | None:
    """  Utilisateurs connectés, pour un clip visible. Restaure le texte d'origine (original=true) ou le
    texte produit par une entrée antérieure de ce segment (event_id) ; exactement l'un des deux. La
    restauration est une correction : version incrémentée, segment à réindexer, nouvelle entrée
    d'historique de type restore ; aucune entrée passée n'est modifiée. Une entrée de locuteur ne se
    restaure pas ici.

    Args:
        id (int):
        index (int):
        origin (str):
        x_csrf_token (str):
        body (SegmentRestoreInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SegmentConflict | SegmentCorrection
     """


    return sync_detailed(
        id=id,
index=index,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,
    body: SegmentRestoreInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | SegmentConflict | SegmentCorrection]:
    """  Utilisateurs connectés, pour un clip visible. Restaure le texte d'origine (original=true) ou le
    texte produit par une entrée antérieure de ce segment (event_id) ; exactement l'un des deux. La
    restauration est une correction : version incrémentée, segment à réindexer, nouvelle entrée
    d'historique de type restore ; aucune entrée passée n'est modifiée. Une entrée de locuteur ne se
    restaure pas ici.

    Args:
        id (int):
        index (int):
        origin (str):
        x_csrf_token (str):
        body (SegmentRestoreInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SegmentConflict | SegmentCorrection]
     """


    kwargs = _get_kwargs(
        id=id,
index=index,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    index: int,
    *,
    client: AuthenticatedClient,
    body: SegmentRestoreInput,
    origin: str,
    x_csrf_token: str,

) -> Error | SegmentConflict | SegmentCorrection | None:
    """  Utilisateurs connectés, pour un clip visible. Restaure le texte d'origine (original=true) ou le
    texte produit par une entrée antérieure de ce segment (event_id) ; exactement l'un des deux. La
    restauration est une correction : version incrémentée, segment à réindexer, nouvelle entrée
    d'historique de type restore ; aucune entrée passée n'est modifiée. Une entrée de locuteur ne se
    restaure pas ici.

    Args:
        id (int):
        index (int):
        origin (str):
        x_csrf_token (str):
        body (SegmentRestoreInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SegmentConflict | SegmentCorrection
     """


    return (await asyncio_detailed(
        id=id,
index=index,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
