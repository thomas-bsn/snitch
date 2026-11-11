from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.root_scan_preview import RootScanPreview
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/video-root/scan-runs/preview",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | RootScanPreview:
    if response.status_code == 200:
        response_200 = RootScanPreview.from_dict(response.json())



        return response_200

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())



        return response_409

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | RootScanPreview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | RootScanPreview]:
    """ Estimer le scan de toute la Racine vidéo, sans le lancer (administrateur)

     Parcourt les dossiers de la Racine (jamais le contenu des vidéos) et compte, sans lister les
    chemins, les vidéos qui seraient mises en file, celles déjà à jour et ce qui est exclu par motif.
    Les Dossiers privés sont exclus pour tous les rôles, Super-administrateur compris ; un dossier
    illisible est compté, pas bloquant.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | RootScanPreview]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,

) -> Error | RootScanPreview | None:
    """ Estimer le scan de toute la Racine vidéo, sans le lancer (administrateur)

     Parcourt les dossiers de la Racine (jamais le contenu des vidéos) et compte, sans lister les
    chemins, les vidéos qui seraient mises en file, celles déjà à jour et ce qui est exclu par motif.
    Les Dossiers privés sont exclus pour tous les rôles, Super-administrateur compris ; un dossier
    illisible est compté, pas bloquant.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | RootScanPreview
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | RootScanPreview]:
    """ Estimer le scan de toute la Racine vidéo, sans le lancer (administrateur)

     Parcourt les dossiers de la Racine (jamais le contenu des vidéos) et compte, sans lister les
    chemins, les vidéos qui seraient mises en file, celles déjà à jour et ce qui est exclu par motif.
    Les Dossiers privés sont exclus pour tous les rôles, Super-administrateur compris ; un dossier
    illisible est compté, pas bloquant.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | RootScanPreview]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,

) -> Error | RootScanPreview | None:
    """ Estimer le scan de toute la Racine vidéo, sans le lancer (administrateur)

     Parcourt les dossiers de la Racine (jamais le contenu des vidéos) et compte, sans lister les
    chemins, les vidéos qui seraient mises en file, celles déjà à jour et ce qui est exclu par motif.
    Les Dossiers privés sont exclus pour tous les rôles, Super-administrateur compris ; un dossier
    illisible est compté, pas bloquant.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | RootScanPreview
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
