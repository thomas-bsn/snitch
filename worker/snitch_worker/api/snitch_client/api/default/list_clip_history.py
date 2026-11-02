from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.clip_history_page import ClipHistoryPage
from ...models.error import Error
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["q"] = q

    params["page"] = page

    params["page_size"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/clip-history",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ClipHistoryPage | Error:
    if response.status_code == 200:
        response_200 = ClipHistoryPage.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ClipHistoryPage | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> Response[ClipHistoryPage | Error]:
    """ Historique du clip

     Administrateurs uniquement. Par clip ayant au moins une tâche, la frise des événements de chaque
    étape (statut, modèle, Worker, tentatives, cycles, erreurs et causes d'échec). Pagination et
    recherche par clip ; non regroupé par exécution. Les essais de diarisation sont exclus.

    Args:
        q (str | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 10.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipHistoryPage | Error]
     """


    kwargs = _get_kwargs(
        q=q,
page=page,
page_size=page_size,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> ClipHistoryPage | Error | None:
    """ Historique du clip

     Administrateurs uniquement. Par clip ayant au moins une tâche, la frise des événements de chaque
    étape (statut, modèle, Worker, tentatives, cycles, erreurs et causes d'échec). Pagination et
    recherche par clip ; non regroupé par exécution. Les essais de diarisation sont exclus.

    Args:
        q (str | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 10.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipHistoryPage | Error
     """


    return sync_detailed(
        client=client,
q=q,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> Response[ClipHistoryPage | Error]:
    """ Historique du clip

     Administrateurs uniquement. Par clip ayant au moins une tâche, la frise des événements de chaque
    étape (statut, modèle, Worker, tentatives, cycles, erreurs et causes d'échec). Pagination et
    recherche par clip ; non regroupé par exécution. Les essais de diarisation sont exclus.

    Args:
        q (str | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 10.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClipHistoryPage | Error]
     """


    kwargs = _get_kwargs(
        q=q,
page=page,
page_size=page_size,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,

) -> ClipHistoryPage | Error | None:
    """ Historique du clip

     Administrateurs uniquement. Par clip ayant au moins une tâche, la frise des événements de chaque
    étape (statut, modèle, Worker, tentatives, cycles, erreurs et causes d'échec). Pagination et
    recherche par clip ; non regroupé par exécution. Les essais de diarisation sont exclus.

    Args:
        q (str | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 10.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClipHistoryPage | Error
     """


    return (await asyncio_detailed(
        client=client,
q=q,
page=page,
page_size=page_size,

    )).parsed
