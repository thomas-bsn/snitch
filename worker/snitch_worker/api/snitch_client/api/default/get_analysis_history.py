from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.analysis_run_history import AnalysisRunHistory
from ...models.error import Error
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    page: int | Unset = 1,
    q: str | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["page"] = page

    params["q"] = q


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/analysis/history",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AnalysisRunHistory | Error:
    if response.status_code == 200:
        response_200 = AnalysisRunHistory.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AnalysisRunHistory | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    page: int | Unset = 1,
    q: str | Unset = UNSET,

) -> Response[AnalysisRunHistory | Error]:
    """  Administrateurs. Historique durable de toutes les exécutions terminées (réussies, échouées,
    annulées) de tous les clips, la plus récente d'abord, avec leurs étapes ; seule partie paginée de la
    Task Queue (10 par page).

    Args:
        page (int | Unset):  Default: 1.
        q (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AnalysisRunHistory | Error]
     """


    kwargs = _get_kwargs(
        page=page,
q=q,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    page: int | Unset = 1,
    q: str | Unset = UNSET,

) -> AnalysisRunHistory | Error | None:
    """  Administrateurs. Historique durable de toutes les exécutions terminées (réussies, échouées,
    annulées) de tous les clips, la plus récente d'abord, avec leurs étapes ; seule partie paginée de la
    Task Queue (10 par page).

    Args:
        page (int | Unset):  Default: 1.
        q (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AnalysisRunHistory | Error
     """


    return sync_detailed(
        client=client,
page=page,
q=q,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    page: int | Unset = 1,
    q: str | Unset = UNSET,

) -> Response[AnalysisRunHistory | Error]:
    """  Administrateurs. Historique durable de toutes les exécutions terminées (réussies, échouées,
    annulées) de tous les clips, la plus récente d'abord, avec leurs étapes ; seule partie paginée de la
    Task Queue (10 par page).

    Args:
        page (int | Unset):  Default: 1.
        q (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AnalysisRunHistory | Error]
     """


    kwargs = _get_kwargs(
        page=page,
q=q,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    page: int | Unset = 1,
    q: str | Unset = UNSET,

) -> AnalysisRunHistory | Error | None:
    """  Administrateurs. Historique durable de toutes les exécutions terminées (réussies, échouées,
    annulées) de tous les clips, la plus récente d'abord, avec leurs étapes ; seule partie paginée de la
    Task Queue (10 par page).

    Args:
        page (int | Unset):  Default: 1.
        q (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AnalysisRunHistory | Error
     """


    return (await asyncio_detailed(
        client=client,
page=page,
q=q,

    )).parsed
