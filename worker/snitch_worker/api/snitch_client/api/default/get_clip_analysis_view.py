from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.clip_analysis_view_response import ClipAnalysisViewResponse
from ...models.error import Error
from typing import cast



def _get_kwargs(
    id: int,

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/clips/{id}/analysis".format(id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ClipAnalysisViewResponse | Error:
    if response.status_code == 200:
        response_200 = ClipAnalysisViewResponse.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ClipAnalysisViewResponse | Error]:
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

) -> Response[ClipAnalysisViewResponse | Error]:
    """  Tout utilisateur qui peut ouvrir le clip. Lecture seule : statut et code de raison stables de chaque
    étape, sans identifiant de job, nom de Worker ni code d'erreur.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipAnalysisViewResponse | Error]
     """


    kwargs = _get_kwargs(
        id=id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient | Client,

) -> ClipAnalysisViewResponse | Error | None:
    """  Tout utilisateur qui peut ouvrir le clip. Lecture seule : statut et code de raison stables de chaque
    étape, sans identifiant de job, nom de Worker ni code d'erreur.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipAnalysisViewResponse | Error
     """


    return sync_detailed(
        id=id,
client=client,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient | Client,

) -> Response[ClipAnalysisViewResponse | Error]:
    """  Tout utilisateur qui peut ouvrir le clip. Lecture seule : statut et code de raison stables de chaque
    étape, sans identifiant de job, nom de Worker ni code d'erreur.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipAnalysisViewResponse | Error]
     """


    kwargs = _get_kwargs(
        id=id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient | Client,

) -> ClipAnalysisViewResponse | Error | None:
    """  Tout utilisateur qui peut ouvrir le clip. Lecture seule : statut et code de raison stables de chaque
    étape, sans identifiant de job, nom de Worker ni code d'erreur.

    Args:
        id (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipAnalysisViewResponse | Error
     """


    return (await asyncio_detailed(
        id=id,
client=client,

    )).parsed
