from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.delete_clip_analysis_body import DeleteClipAnalysisBody
from ...models.delete_clip_analysis_response_200 import DeleteClipAnalysisResponse200
from ...models.error import Error
from typing import cast



def _get_kwargs(
    id: int,
    *,
    body: DeleteClipAnalysisBody,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/admin/clips/{id}/analysis".format(id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DeleteClipAnalysisResponse200 | Error:
    if response.status_code == 200:
        response_200 = DeleteClipAnalysisResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DeleteClipAnalysisResponse200 | Error]:
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
    body: DeleteClipAnalysisBody,
    origin: str,
    x_csrf_token: str,

) -> Response[DeleteClipAnalysisResponse200 | Error]:
    """  Super-administrateur seulement. Suppression d'étapes ou de toute l'analyse d'un clip, refusée
    pendant une analyse active. Les Personnes ne sont jamais supprimées.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (DeleteClipAnalysisBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeleteClipAnalysisResponse200 | Error]
     """


    kwargs = _get_kwargs(
        id=id,
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
    *,
    client: AuthenticatedClient | Client,
    body: DeleteClipAnalysisBody,
    origin: str,
    x_csrf_token: str,

) -> DeleteClipAnalysisResponse200 | Error | None:
    """  Super-administrateur seulement. Suppression d'étapes ou de toute l'analyse d'un clip, refusée
    pendant une analyse active. Les Personnes ne sont jamais supprimées.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (DeleteClipAnalysisBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeleteClipAnalysisResponse200 | Error
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,
    body: DeleteClipAnalysisBody,
    origin: str,
    x_csrf_token: str,

) -> Response[DeleteClipAnalysisResponse200 | Error]:
    """  Super-administrateur seulement. Suppression d'étapes ou de toute l'analyse d'un clip, refusée
    pendant une analyse active. Les Personnes ne sont jamais supprimées.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (DeleteClipAnalysisBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeleteClipAnalysisResponse200 | Error]
     """


    kwargs = _get_kwargs(
        id=id,
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
    *,
    client: AuthenticatedClient | Client,
    body: DeleteClipAnalysisBody,
    origin: str,
    x_csrf_token: str,

) -> DeleteClipAnalysisResponse200 | Error | None:
    """  Super-administrateur seulement. Suppression d'étapes ou de toute l'analyse d'un clip, refusée
    pendant une analyse active. Les Personnes ne sont jamais supprimées.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):
        body (DeleteClipAnalysisBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeleteClipAnalysisResponse200 | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
