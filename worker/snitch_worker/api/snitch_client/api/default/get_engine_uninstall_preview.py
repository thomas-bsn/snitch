from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.engine_uninstall_preview import EngineUninstallPreview
from ...models.error import Error
from typing import cast



def _get_kwargs(
    *,
    worker_id: str,
    digest: str,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["worker_id"] = worker_id

    params["digest"] = digest


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-engines/uninstall-preview",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EngineUninstallPreview | Error:
    if response.status_code == 200:
        response_200 = EngineUninstallPreview.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EngineUninstallPreview | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    worker_id: str,
    digest: str,

) -> Response[EngineUninstallPreview | Error]:
    """  Administrateur : ce qui empêche le retrait d'une version d'engine sur un Worker (modèles liés, jobs
    engagés, transitions d'empreintes, installation en cours).

    Args:
        worker_id (str):
        digest (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EngineUninstallPreview | Error]
     """


    kwargs = _get_kwargs(
        worker_id=worker_id,
digest=digest,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    worker_id: str,
    digest: str,

) -> EngineUninstallPreview | Error | None:
    """  Administrateur : ce qui empêche le retrait d'une version d'engine sur un Worker (modèles liés, jobs
    engagés, transitions d'empreintes, installation en cours).

    Args:
        worker_id (str):
        digest (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EngineUninstallPreview | Error
     """


    return sync_detailed(
        client=client,
worker_id=worker_id,
digest=digest,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    worker_id: str,
    digest: str,

) -> Response[EngineUninstallPreview | Error]:
    """  Administrateur : ce qui empêche le retrait d'une version d'engine sur un Worker (modèles liés, jobs
    engagés, transitions d'empreintes, installation en cours).

    Args:
        worker_id (str):
        digest (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EngineUninstallPreview | Error]
     """


    kwargs = _get_kwargs(
        worker_id=worker_id,
digest=digest,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    worker_id: str,
    digest: str,

) -> EngineUninstallPreview | Error | None:
    """  Administrateur : ce qui empêche le retrait d'une version d'engine sur un Worker (modèles liés, jobs
    engagés, transitions d'empreintes, installation en cours).

    Args:
        worker_id (str):
        digest (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EngineUninstallPreview | Error
     """


    return (await asyncio_detailed(
        client=client,
worker_id=worker_id,
digest=digest,

    )).parsed
